package ratelimit

import (
	"net/http"
	"strings"
)

type MatchFunc func(r *http.Request) bool

// ConditionalLimiter is a rate limiter that can apply different rate limits to different requests.
type ConditionalLimiter struct {
	rules          []*rule
	defaultLimiter *RateLimiter
}

// rule is a rule that matches a request to a rate limiter.
type rule struct {
	match   MatchFunc
	limiter *RateLimiter
}

func NewConditional(defaultLimiter *RateLimiter) *ConditionalLimiter {
	return &ConditionalLimiter{
		rules:          make([]*rule, 0),
		defaultLimiter: defaultLimiter,
	}
}

// AddRule adds a new rule to the conditional limiter.
// The rules are evaluated in order, and the first rule that matches is used.
func (c *ConditionalLimiter) AddRule(match MatchFunc, limiter *RateLimiter) *ConditionalLimiter {
	c.rules = append(c.rules, &rule{
		match:   match,
		limiter: limiter,
	})
	return c
}

// Handler returns an http.Handler that applies the conditional rate limiting.
func (c *ConditionalLimiter) Handler(next http.Handler) http.Handler {
	// If no default limiter is provided, use the next handler directly.
	defaultHandler := next
	if c.defaultLimiter != nil {
		defaultHandler = c.defaultLimiter.Handler(next)
	}

	// Pre-wrap all rule handlers
	ruleHandlers := make([]http.Handler, len(c.rules))
	for i, r := range c.rules {
		ruleHandlers[i] = r.limiter.Handler(next)
	}

	// Return the handler that applies the conditional rate limiting.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check each rule in order
		for i, rule := range c.rules {
			if rule.match(r) {
				ruleHandlers[i].ServeHTTP(w, r)
				return
			}
		}

		// Apply default limiter if no rule matched
		defaultHandler.ServeHTTP(w, r)
	})
}

// --- Helper Matchers ---

// MatchPathPrefix limits routes that start with a specific prefix (e.g. "/api/v1/users/")
func MatchPathPrefix(prefix string) MatchFunc {
	return func(r *http.Request) bool {
		return strings.HasPrefix(r.URL.Path, prefix)
	}
}

// MatchExactPath exactly matches a URL path (e.g. "/login")
func MatchExactPath(path string) MatchFunc {
	return func(r *http.Request) bool {
		return r.URL.Path == path
	}
}

// MatchMethodAndPath limits based on HTTP Method and Path (e.g. "POST", "/login")
func MatchMethodAndPath(method, path string) MatchFunc {
	return func(r *http.Request) bool {
		return r.Method == method && r.URL.Path == path
	}
}
