package ratelimit

import (
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
)

// Strategy defines the rate limiting algorithm to use.
type Strategy int

const (
	// SlidingWindow uses a sliding window counter algorithm.
	// It provides a good balance between accuracy and performance.
	SlidingWindow Strategy = iota

	// TokenBucket uses a token bucket algorithm.
	// Best for allowing controlled bursts of traffic.
	TokenBucket

	// FixedWindow uses a fixed window counter algorithm.
	// Simplest approach but susceptible to boundary bursts.
	FixedWindow
)

func (s Strategy) String() string {
	switch s {
	case SlidingWindow:
		return "sliding_window"
	case TokenBucket:
		return "token_bucket"
	case FixedWindow:
		return "fixed_window"
	default:
		return "unknown"
	}
}

// Config holds the configuration for a rate limiter instance.
type Config struct {
	// Limit is the maximum number of requests allowed within the Window.
	Limit int `validate:"required,gt=0"`

	// Window is the time duration for the rate limit window.
	Window time.Duration `validate:"required,gt=0"`

	// Strategy is the rate limiting algorithm to use.
	Strategy Strategy

	// KeyFunc extracts the rate limiting key from the request.
	// If nil, defaults to IP-based key extraction.
	KeyFunc KeyFunc

	// Prefix is prepended to all Redis keys for namespacing.
	// Defaults to "rl" if empty.
	Prefix string

	// ExceededHandler is called when the rate limit is exceeded.
	// If nil, a default JSON 429 response is sent.
	ExceededHandler http.HandlerFunc

	// SkipFunc determines whether to skip rate limiting for a request.
	// If nil, no requests are skipped.
	SkipFunc func(r *http.Request) bool

	// EnableHeaders controls whether rate limit headers are added to responses.
	// Defaults to true.
	EnableHeaders *bool

	// TrustProxy indicates whether to trust X-Forwarded-For / X-Real-IP headers.
	// Set to true when behind a reverse proxy.
	TrustProxy bool

	// BurstLimit is used only with TokenBucket strategy.
	// It defines the maximum burst size. Defaults to Limit if zero.
	BurstLimit int `validate:"omitempty,gte=0"`

	// FallbackToAllow determines behavior when Redis is unavailable.
	// If true, requests are allowed (open circuit). If false, requests are denied.
	FallbackToAllow bool
}

func (c *Config) Validate() error {
	v := validator.New()
	return v.Struct(c)
}

func (c *Config) applyDefaults() {
	if c.KeyFunc == nil {
		c.KeyFunc = KeyByIP(c.TrustProxy)
	}
	if c.Prefix == "" {
		c.Prefix = "rl"
	}
	if c.EnableHeaders == nil {
		enable := true
		c.EnableHeaders = &enable
	}
	if c.BurstLimit == 0 && c.Strategy == TokenBucket {
		c.BurstLimit = c.Limit
	}
}

// headersEnabled returns whether rate limit headers should be added
func (c *Config) headersEnabled() bool {
	return c.EnableHeaders == nil || *c.EnableHeaders
}

type RateLimitResult struct {
	// Allowed indicated whether the request is allowed or not
	Allowed bool

	// Limit is the maximum number of the requests allowed
	Limit int

	// Remaining is the number of requests remaining in the current window
	Remaining int

	// RetryAfter is the duration after which the client should retry.
	// This is only set when the request is denied
	RetryAfter time.Duration

	// RestAt is the time when the rate limit window resets
	ResetAt time.Time
}
