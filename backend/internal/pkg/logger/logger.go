package logger

import (
	"context"
	"maps"
	"sync"
)

type contextKey string

const canonicalFieldsKey contextKey = "canonical_fields"

// CanonicalFields hold all the fields that will be emitted in the single
// canonical log line at the end of a request's lifecycle
type CanonicalFields struct {
	mu     sync.Mutex
	fields map[string]any
}

// NewCanonicalFields returns a new instance of CanonicalFields
func NewCanonicalFields() *CanonicalFields {
	return &CanonicalFields{
		fields: make(map[string]any),
	}
}

// Set adds a key-value pair to the canonical fields
func (cf *CanonicalFields) Set(key string, value any) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	cf.fields[key] = value
}

// Get retrieves a value from the canonical fields
// Returns nil and false if not found
func (cf *CanonicalFields) Get(key string) (any, bool) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	val, ok := cf.fields[key]
	return val, ok
}

// All returns a snapshot of all canonical fields
func (cf *CanonicalFields) All() map[string]any {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	snapshot := make(map[string]any, len(cf.fields))
	maps.Copy(snapshot, cf.fields)
	return snapshot
}

// WithCanonicalFields attaches a CanonicalFields instance to a context.
// This is typically called in middleware to initialize the canonical fields for a request.
func WithCanonicalFields(ctx context.Context, cf *CanonicalFields) context.Context {
	return context.WithValue(ctx, canonicalFieldsKey, cf)
}

// GetCanonicalFields retrieves a CanonicalFields instance from a context.
// If no CanonicalFields is found, it returns a new instance to avoid nil pointer dereference.
func GetCanonicalFields(ctx context.Context) *CanonicalFields {
	val, ok := ctx.Value(canonicalFieldsKey).(*CanonicalFields)
	if !ok {
		return NewCanonicalFields()
	}
	return val
}

// AddField is a convenience function that adds a field to the canonical log
// line from anywhere in the call stack that has access to the request context.
// This is the primary function your handlers and services will call.
//
// Usage:
//
//	logger.AddField(r.Context(), "user_id", "usr_abc123")
//	logger.AddField(r.Context(), "cause_id", "cause_xyz")
//	logger.AddField(r.Context(), "action", "profile_view")
func AddField(ctx context.Context, key string, value any) {
	cf := GetCanonicalFields(ctx)
	cf.Set(key, value)
}

// AddFields is a convenience function to add multiple fields at once.
func AddFields(ctx context.Context, fields map[string]any) {
	cf := GetCanonicalFields(ctx)
	for k, v := range fields {
		cf.Set(k, v)
	}
}
