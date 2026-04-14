package server

import (
	"log/slog"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/middleware/ratelimit"
	"github.com/redis/go-redis/v9"
)

func (s *Server) setupRateLimiters(redisClient *redis.Client, logger *slog.Logger) (*ratelimit.ConditionalLimiter, error) {
	// --- Auth: Strict limits to prevent brute force / spam ---

	// POST /api/v1/auth/register — sliding window, tight cap per IP
	registerLimiter, err := ratelimit.New(redisClient, ratelimit.Config{
		Strategy: ratelimit.SlidingWindow,
		Limit:    5,
		Window:   10 * time.Minute,
		Prefix:   "rl:register",
		KeyFunc:  ratelimit.KeyByIP(true),
	}, logger)
	if err != nil {
		return nil, err
	}

	// POST /api/v1/auth/login — sliding window, slightly more lenient but still strict
	loginLimiter, err := ratelimit.New(redisClient, ratelimit.Config{
		Strategy: ratelimit.SlidingWindow,
		Limit:    10,
		Window:   5 * time.Minute,
		Prefix:   "rl:login",
		KeyFunc:  ratelimit.KeyByIP(true),
	}, logger)
	if err != nil {
		return nil, err
	}

	// POST /api/v1/auth/refresh — token bucket, allows short bursts (tab reload, multi-tab)
	// but throttles sustained hammering
	refreshLimiter, err := ratelimit.New(redisClient, ratelimit.Config{
		Strategy:   ratelimit.TokenBucket,
		Limit:      20, // max tokens in bucket
		BurstLimit: 20,
		Window:     1 * time.Minute,
		Prefix:     "rl:refresh",
		KeyFunc:    ratelimit.KeyByIP(true),
	}, logger)
	if err != nil {
		return nil, err
	}

	// POST /api/v1/auth/logout and /logout-all — fixed window, low limit (authenticated)
	// Keyed by user ID so one user can't spam invalidation
	logoutLimiter, err := ratelimit.New(redisClient, ratelimit.Config{
		Strategy: ratelimit.FixedWindow,
		Limit:    10,
		Window:   1 * time.Minute,
		Prefix:   "rl:logout",
		KeyFunc:  ratelimit.KeyByUserID("userID", true),
	}, logger)
	if err != nil {
		return nil, err
	}

	// --- Users ---

	// GET /api/v1/users/{username} — public profile, sliding window per IP
	// Generous but protects against scrapers
	getUserLimiter, err := ratelimit.New(redisClient, ratelimit.Config{
		Strategy: ratelimit.SlidingWindow,
		Limit:    30,
		Window:   1 * time.Minute,
		Prefix:   "rl:get_user",
		KeyFunc:  ratelimit.KeyByIP(true),
	}, logger)
	if err != nil {
		return nil, err
	}

	// PATCH /me, POST /me/password, DELETE /me — authenticated mutations, fixed window per user
	// Grouped: these are rare intentional actions, keep it tight
	userMutateLimiter, err := ratelimit.New(redisClient, ratelimit.Config{
		Strategy: ratelimit.SlidingWindow,
		Limit:    15,
		Window:   15 * time.Minute,
		Prefix:   "rl:user_mutate",
		KeyFunc:  ratelimit.ComposeKeys(ratelimit.KeyByUserID("userID", true), ratelimit.KeyByIP(true)),
	}, logger)
	if err != nil {
		return nil, err
	}

	// POST /me/password — extra tight, separate limiter layered on top
	changePasswordLimiter, err := ratelimit.New(redisClient, ratelimit.Config{
		Strategy: ratelimit.SlidingWindow,
		Limit:    5,
		Window:   15 * time.Minute,
		Prefix:   "rl:change_password",
		KeyFunc:  ratelimit.ComposeKeys(ratelimit.KeyByUserID("userID", true), ratelimit.KeyByIP(true)),
	}, logger)
	if err != nil {
		return nil, err
	}

	// --- Admin ---

	// All admin routes — fixed window per user ID, relaxed since these are privileged trusted users
	adminLimiter, err := ratelimit.New(redisClient, ratelimit.Config{
		Strategy: ratelimit.FixedWindow,
		Limit:    120,
		Window:   1 * time.Minute,
		Prefix:   "rl:admin",
		KeyFunc:  ratelimit.KeyByUserID("userID", true),
	}, logger)
	if err != nil {
		return nil, err
	}

	// --- Global fallback — token bucket per IP for anything not matched ---
	globalLimiter, err := ratelimit.New(redisClient, ratelimit.Config{
		Strategy:   ratelimit.TokenBucket,
		Limit:      60,
		BurstLimit: 60,
		Window:     1 * time.Minute,
		Prefix:     "rl:global",
		KeyFunc:    ratelimit.KeyByIP(true),
	}, logger)
	if err != nil {
		return nil, err
	}

	// --- Wire up ConditionalLimiter ---
	cl := ratelimit.NewConditional(globalLimiter).
		// --- Auth ---
		AddRule(ratelimit.MatchMethodAndPath("POST", "/api/v1/auth/register"), registerLimiter).
		AddRule(ratelimit.MatchMethodAndPath("POST", "/api/v1/auth/login"), loginLimiter).
		AddRule(ratelimit.MatchMethodAndPath("POST", "/api/v1/auth/refresh"), refreshLimiter).
		AddRule(ratelimit.MatchPathPrefix("/api/v1/auth/logout"), logoutLimiter).
		AddRule(ratelimit.MatchPathPrefix("/api/v1/auth/logout-all"), logoutLimiter).
		// --- Users ---
		AddRule(ratelimit.MatchMethodAndPath("GET", "/api/v1/users/"), getUserLimiter).
		AddRule(ratelimit.MatchMethodAndPath("POST", "/api/v1/users/me/password"), changePasswordLimiter).
		AddRule(ratelimit.MatchPathPrefix("/api/v1/users/me"), userMutateLimiter).
		// --- Admin ---
		AddRule(ratelimit.MatchPathPrefix("/api/v1/admin"), adminLimiter)

	return cl, nil
}
