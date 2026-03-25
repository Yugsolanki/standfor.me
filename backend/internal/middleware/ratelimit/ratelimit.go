package ratelimit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Standard rate limit response headers
const (
	HeaderRateLimitLimit     = "X-RateLimit-Limit"
	HeaderRateLimitRemaining = "X-RateLimit-Remaining"
	HeaderRateLimitReset     = "X-RateLimit-Reset"
	HeaderRetryAfter         = "RetryAfter"
)

// RateLimiter provides HTTP middleware for rate limiting using redis as the backend store
type RateLimiter struct {
	client *redis.Client
	config Config
	logger *slog.Logger

	// Preloaded lua script SHA's for performance
	slidingWindowSHA string
	fixedWindowSHA   string
	tokenBucketSHA   string
}

// New creates a new rate limiter with the given redis client and config
func New(client *redis.Client, config Config, logger *slog.Logger) (*RateLimiter, error) {
	if client == nil {
		return nil, fmt.Errorf("ratelimit: redis client must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}
	config.applyDefaults()

	rl := &RateLimiter{
		client: client,
		config: config,
		logger: logger,
	}

	// Preload lua script into redis for better performance
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rl.loadScripts(ctx); err != nil {
		logger.Warn("failed to pre-load rate limit lua scripts; will use EVAl at runtime", slog.String("error", err.Error()))
	}

	return rl, nil
}

// loadScripts is used to load lua scripts into redis
func (rl *RateLimiter) loadScripts(ctx context.Context) error {
	var err error

	rl.slidingWindowSHA, err = rl.client.ScriptLoad(ctx, slidingWindowScript).Result()
	if err != nil {
		return fmt.Errorf("loading sliding window script: %w", err)
	}

	rl.fixedWindowSHA, err = rl.client.ScriptLoad(ctx, fixedWindowScript).Result()
	if err != nil {
		return fmt.Errorf("loading fixed window script: %w", err)
	}

	rl.tokenBucketSHA, err = rl.client.ScriptLoad(ctx, tokenBucketScript).Result()
	if err != nil {
		return fmt.Errorf("loading token bucket script: %w", err)
	}

	return nil
}

func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check if this request should skip rate limiting
		if rl.config.SkipFunc != nil && rl.config.SkipFunc(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract Key
		key, err := rl.config.KeyFunc(r)
		if err != nil {
			rl.logger.Error("failed to extract rate limit key",
				slog.String("error", err.Error()),
				slog.String("remote_addr", r.RemoteAddr))

			// if we can't identify the client, apply rate limiting with a fallback way
			key = fmt.Sprintf("unknown:%s", r.RemoteAddr)
		}

		// Build full redis key with prefix
		redisKey := fmt.Sprintf("%s:%s:%s", rl.config.Prefix, rl.config.Strategy, key)

		// Execute rate limit check
		result, err := rl.check(r.Context(), redisKey)
		if err != nil {
			rl.logger.Error("rate limit check failed",
				slog.String("error", err.Error()),
				slog.String("key", key))

			if rl.config.FallbackToAllow {
				// Open circuit: allow the circuit through
				next.ServeHTTP(w, r)
				return
			}

			// Closed circuit: deny the request
			rl.handleExceed(w, r, &RateLimitResult{
				Allowed:    false,
				Limit:      rl.config.Limit,
				Remaining:  0,
				RetryAfter: time.Second,
				ResetAt:    time.Now().Add(time.Second),
			})
			return
		}

		// Set rate limit headers
		if rl.config.headersEnabled() {
			rl.setHeaders(w, result)
		}

		if !result.Allowed {
			rl.handleExceed(w, r, result)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// check performs the rate limit check against Redis.
func (rl *RateLimiter) check(ctx context.Context, key string) (*RateLimitResult, error) {
	switch rl.config.Strategy {
	case SlidingWindow:
		return rl.checkSlidingWindow(ctx, key)
	case FixedWindow:
		return rl.checkFixedWindow(ctx, key)
	case TokenBucket:
		return rl.checkTokenBucket(ctx, key)
	default:
		return nil, fmt.Errorf("ratelimit: unsupported strategy: %s", rl.config.Strategy)
	}
}

// checkSlidingWindow implements the sliding window rate limit check.
func (rl *RateLimiter) checkSlidingWindow(ctx context.Context, key string) (*RateLimitResult, error) {
	now := time.Now()
	nowMicro := now.UnixMicro()
	windowStartMicro := now.Add(-rl.config.Window).UnixMicro()
	ttlSeconds := int64(math.Ceil(rl.config.Window.Seconds())) + 1 // +1 for safety margin
	requestID := uuid.New().String()

	var res []interface{}
	var err error

	// Try EVALSHA first, fall back to EVAL
	if rl.slidingWindowSHA != "" {
		res, err = rl.evalSHA(ctx, rl.slidingWindowSHA, slidingWindowScript,
			[]string{key},
			nowMicro, windowStartMicro, rl.config.Limit, ttlSeconds, requestID,
		)
	} else {
		res, err = rl.client.Eval(ctx, slidingWindowScript,
			[]string{key},
			nowMicro, windowStartMicro, rl.config.Limit, ttlSeconds, requestID,
		).Slice()
		if err != nil {
			return nil, fmt.Errorf("sliding window eval: %w", err)
		}
	}
	if err != nil {
		return nil, err
	}

	allowed := res[0].(int64) == 1
	currentCount := int(res[1].(int64))
	remaining := rl.config.Limit - currentCount
	remaining = max(remaining, 0)

	result := &RateLimitResult{
		Allowed:   allowed,
		Limit:     rl.config.Limit,
		Remaining: remaining,
		ResetAt:   now.Add(rl.config.Window),
	}

	if !allowed {
		retryAfterMicro := res[2].(int64)
		if retryAfterMicro > 0 {
			result.RetryAfter = time.Duration(retryAfterMicro) * time.Microsecond
		} else {
			result.RetryAfter = rl.config.Window
		}
	}

	return result, nil
}

// checkFixedWindow implements the fixed window rate limit check.
func (rl *RateLimiter) checkFixedWindow(ctx context.Context, key string) (*RateLimitResult, error) {
	now := time.Now()
	windowSeconds := int64(math.Ceil(rl.config.Window.Seconds()))

	// Add window identifier to key for fixed windows
	windowID := now.Unix() / windowSeconds
	windowKey := fmt.Sprintf("%s:%d", key, windowID)

	var res []interface{}
	var err error

	if rl.fixedWindowSHA != "" {
		res, err = rl.evalSHA(ctx, rl.fixedWindowSHA, fixedWindowScript,
			[]string{windowKey},
			rl.config.Limit, windowSeconds,
		)
	} else {
		res, err = rl.client.Eval(ctx, fixedWindowScript,
			[]string{windowKey},
			rl.config.Limit, windowSeconds,
		).Slice()
		if err != nil {
			return nil, fmt.Errorf("fixed window eval: %w", err)
		}
	}
	if err != nil {
		return nil, err
	}

	allowed := res[0].(int64) == 1
	currentCount := int(res[1].(int64))
	remainingTTLms := res[2].(int64)

	remaining := rl.config.Limit - currentCount
	remaining = max(remaining, 0)

	var resetAt time.Time
	if remainingTTLms > 0 {
		resetAt = now.Add(time.Duration(remainingTTLms) * time.Millisecond)
	} else {
		resetAt = now.Add(rl.config.Window)
	}

	result := &RateLimitResult{
		Allowed:   allowed,
		Limit:     rl.config.Limit,
		Remaining: remaining,
		ResetAt:   resetAt,
	}

	if !allowed {
		result.RetryAfter = time.Duration(remainingTTLms) * time.Millisecond
	}

	return result, nil
}

// checkTokenBucket implements the token bucket rate limit check.
func (rl *RateLimiter) checkTokenBucket(ctx context.Context, key string) (*RateLimitResult, error) {
	now := time.Now()
	nowMicro := now.UnixMicro()
	maxTokens := rl.config.BurstLimit
	refillRate := float64(rl.config.Limit) / rl.config.Window.Seconds()
	ttlSeconds := int64(math.Ceil(rl.config.Window.Seconds())) * 2 // 2x window for safety

	var res []interface{}
	var err error

	if rl.tokenBucketSHA != "" {
		res, err = rl.evalSHA(ctx, rl.tokenBucketSHA, tokenBucketScript,
			[]string{key},
			maxTokens, refillRate, nowMicro, ttlSeconds,
		)
	} else {
		res, err = rl.client.Eval(ctx, tokenBucketScript,
			[]string{key},
			maxTokens, refillRate, nowMicro, ttlSeconds,
		).Slice()
		if err != nil {
			return nil, fmt.Errorf("token bucket eval: %w", err)
		}
	}
	if err != nil {
		return nil, err
	}

	allowed := res[0].(int64) == 1
	remainingTokens := int(res[1].(int64))

	result := &RateLimitResult{
		Allowed:   allowed,
		Limit:     maxTokens,
		Remaining: remainingTokens,
		ResetAt:   now.Add(time.Duration(float64(time.Second) * float64(maxTokens) / refillRate)),
	}

	if !allowed {
		retryAfterMicro := res[2].(int64)
		if retryAfterMicro > 0 {
			result.RetryAfter = time.Duration(retryAfterMicro) * time.Microsecond
		} else {
			// Estimate: time for 1 token to be refilled
			result.RetryAfter = time.Duration(float64(time.Second) / refillRate)
		}
	}

	return result, nil
}

// evalSHA tries EVALSHA and falls back to EVAL if the script is not cached
func (rl *RateLimiter) evalSHA(ctx context.Context, sha, script string, keys []string, args ...interface{}) ([]interface{}, error) {
	res, err := rl.client.EvalSha(ctx, sha, keys, args...).Result()
	if err != nil {
		// NOSCRIPT error - script not in redis cache, fall back EVAL
		if isNoScript(err) {
			rl.logger.Debug("EVALSHA miss, falling back to EVAL")
			res, err = rl.client.Eval(ctx, script, keys, args...).Result()
			if err != nil {
				return nil, fmt.Errorf("eval fallback: %w", err)
			}
		} else {
			return nil, fmt.Errorf("evalsha: %w", err)
		}
	}

	// The lua scripts return array/tables
	switch v := res.(type) {
	case []interface{}:
		return v, nil
	default:
		return nil, fmt.Errorf("unexpected result type from lua script: %T", res)
	}
}

// isNoScript check if redis error is no NOSCRIPT error
func isNoScript(err error) bool {
	return err != nil && redis.HasErrorPrefix(err, "NOSCRIPT")
}

// setHeaders sets the standard rate limiting headers on the response.
func (rl *RateLimiter) setHeaders(w http.ResponseWriter, result *RateLimitResult) {
	w.Header().Set(HeaderRateLimitLimit, strconv.Itoa(result.Limit))
	w.Header().Set(HeaderRateLimitRemaining, strconv.Itoa(result.Remaining))
	w.Header().Set(HeaderRateLimitReset, strconv.FormatInt(result.ResetAt.Unix(), 10))

	if !result.Allowed && result.RetryAfter > 0 {
		retrySeconds := int(math.Ceil(result.RetryAfter.Seconds()))
		retrySeconds = max(retrySeconds, 1)
		w.Header().Set(HeaderRetryAfter, strconv.Itoa(retrySeconds))
	}
}

// handleExceed handles the case when the rate limit is exceeded
func (rl *RateLimiter) handleExceed(w http.ResponseWriter, r *http.Request, result *RateLimitResult) {
	// Set header before calling custom handler
	if rl.config.headersEnabled() {
		rl.setHeaders(w, result)
	}

	if rl.config.ExceededHandler != nil {
		rl.config.ExceededHandler(w, r)
		return
	}

	// Default 429 JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	retryAfterSecs := int(math.Ceil(result.RetryAfter.Seconds()))
	retryAfterSecs = max(retryAfterSecs, 1)

	resp := map[string]interface{}{
		"error":            "rate limit exceeded",
		"message":          fmt.Sprintf("Too many requests. Please retry after %d seconds", retryAfterSecs),
		"retry_after_secs": retryAfterSecs,
		"limit":            result.Limit,
		"remaining":        result.Remaining,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		rl.logger.Error("failed to write rate limit response",
			slog.String("error", err.Error()))
	}
}

// Reset clears the rate limit state for a specific key.
// Useful for administrative purposes.
func (rl *RateLimiter) Reset(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("%s:%s:%s", rl.config.Prefix, rl.config.Strategy, key)
	return rl.client.Del(ctx, redisKey).Err()
}

// Status retrieves the current rate limit status for a key without consuming a request.
func (rl *RateLimiter) Status(ctx context.Context, key string) (*RateLimitResult, error) {
	redisKey := fmt.Sprintf("%s:%s:%s", rl.config.Prefix, rl.config.Strategy, key)
	now := time.Now()

	switch rl.config.Strategy {
	case SlidingWindow:
		windowStart := now.Add(-rl.config.Window).UnixMicro()
		// Remove expired and count
		pipe := rl.client.Pipeline()
		pipe.ZRemRangeByScore(ctx, redisKey, "-inf", strconv.FormatInt(windowStart, 10))
		cardCmd := pipe.ZCard(ctx, redisKey)
		_, err := pipe.Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("sliding window status: %w", err)
		}
		count := int(cardCmd.Val())
		remaining := rl.config.Limit - count
		remaining = max(remaining, 0)

		return &RateLimitResult{
			Allowed:   count < rl.config.Limit,
			Limit:     rl.config.Limit,
			Remaining: remaining,
			ResetAt:   now.Add(rl.config.Window),
		}, nil

	case FixedWindow:
		windowSeconds := int64(math.Ceil(rl.config.Window.Seconds()))
		windowID := now.Unix() / windowSeconds
		windowKey := fmt.Sprintf("%s:%d", redisKey, windowID)
		val, err := rl.client.Get(ctx, windowKey).Int()
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("fixed window status: %w", err)
		}
		remaining := rl.config.Limit - val
		remaining = max(remaining, 0)

		return &RateLimitResult{
			Allowed:   val < rl.config.Limit,
			Limit:     rl.config.Limit,
			Remaining: remaining,
			ResetAt:   now.Add(rl.config.Window),
		}, nil

	case TokenBucket:
		vals, err := rl.client.HMGet(ctx, redisKey, "tokens", "last_refill").Result()
		if err != nil {
			return nil, fmt.Errorf("token bucket status: %w", err)
		}
		tokens := float64(rl.config.BurstLimit)
		if vals[0] != nil {
			if t, err := strconv.ParseFloat(vals[0].(string), 64); err == nil {
				tokens = t
			}
		}
		remaining := int(tokens)
		remaining = max(remaining, 0)

		return &RateLimitResult{
			Allowed:   remaining > 0,
			Limit:     rl.config.BurstLimit,
			Remaining: remaining,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported strategy: %s", rl.config.Strategy)
	}
}
