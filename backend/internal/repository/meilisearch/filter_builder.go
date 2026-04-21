package meilisearch

import (
	"fmt"
	"strings"
)

// filterBuilder accumulates filter clauses that are joined with AND.
// For OR logic between individual clauses, use addOr().
type filterBuilder struct {
	clauses []string
}

// newFilterBuilder returns an empty filterBuilder
func newFilterBuilder() *filterBuilder {
	return &filterBuilder{}
}

// add appends a raw pre-formatted filter clause.
//
//lint:ignore U1000 ignoring unused function for now
func (b *filterBuilder) add(clause string) *filterBuilder {
	if clause != "" {
		b.clauses = append(b.clauses, clause)
	}
	return b
}

// addEquals appends: field = "value"
// String values are quoted; use addEqualsInt/addEqualsBool for other types.
func (b *filterBuilder) addEquals(field, value string) *filterBuilder {
	if value != "" {
		b.clauses = append(b.clauses, fmt.Sprintf("%s = %s", field, value))
	}
	return b
}

// addEqualsInt appends: field = value (integer, no quotes)
//
//lint:ignore U1000 ignoring unused function for now
func (b *filterBuilder) addEqualsInt(field string, value int) *filterBuilder {
	b.clauses = append(b.clauses, fmt.Sprintf(`%s = %d`, field, value))
	return b
}

// addEqualsBool appends: field = true  or  field = false
func (b *filterBuilder) addEqualsBool(field string, value bool) *filterBuilder {
	b.clauses = append(b.clauses, fmt.Sprintf(`%s = %v`, field, value))
	return b
}

// addOptionalBool appends a boolean filter only if the pointer is non-nil.
// This is the idiomatic pattern for optional boolean query parameters.
func (b *filterBuilder) addOptionalBool(field string, value *bool) *filterBuilder {
	if value != nil {
		b.addEqualsBool(field, *value)
	}
	return b
}

// addGTE appends: field >= value  (for integer thresholds)
func (b *filterBuilder) addGTE(field string, value int) *filterBuilder {
	b.clauses = append(b.clauses, fmt.Sprintf(`%s >= %d`, field, value))
	return b
}

// addGTEFloat appends: field >= value  (for float thresholds like avg_tier)
func (b *filterBuilder) addGTEFloat(field string, value float64) *filterBuilder {
	b.clauses = append(b.clauses, fmt.Sprintf(`%s >= %g`, field, value))
	return b
}

// addLTEFloat appends: field <= value
func (b *filterBuilder) addLTEFloat(field string, value float64) *filterBuilder {
	b.clauses = append(b.clauses, fmt.Sprintf(`%s <= %g`, field, value))
	return b
}

// addGTEInt64 appends: field >= value  (for int64 thresholds like supporter_count)
func (b *filterBuilder) addGTEInt64(field string, value int64) *filterBuilder {
	b.clauses = append(b.clauses, fmt.Sprintf(`%s >= %d`, field, value))
	return b
}

// addGTEFloat64 appends: field >= value  (for float64 like trending_score)
func (b *filterBuilder) addGTEFloat64(field string, value float64) *filterBuilder {
	b.clauses = append(b.clauses, fmt.Sprintf(`%s >= %g`, field, value))
	return b
}

// addOptionalGTE appends a >= filter only if the pointer is non-nil.
func (b *filterBuilder) addOptionalGTE(field string, value *int) *filterBuilder {
	if value != nil {
		b.addGTE(field, *value)
	}
	return b
}

// addOptionalGTEFloat appends a >= filter only if the pointer is non-nil.
func (b *filterBuilder) addOptionalGTEFloat(field string, value *float64) *filterBuilder {
	if value != nil {
		b.addGTEFloat(field, *value)
	}
	return b
}

// addOptionalLTEFloat appends a <= filter only if the pointer is non-nil.
func (b *filterBuilder) addOptionalLTEFloat(field string, value *float64) *filterBuilder {
	if value != nil {
		b.addLTEFloat(field, *value)
	}
	return b
}

// addOptionalGTEInt64 appends a >= filter only if the pointer is non-nil.
func (b *filterBuilder) addOptionalGTEInt64(field string, value *int64) *filterBuilder {
	if value != nil {
		b.addGTEInt64(field, *value)
	}
	return b
}

// addOptionalGTEFloat64 appends a >= filter only if the pointer is non-nil.
func (b *filterBuilder) addOptionalGTEFloat64(field string, value *float64) *filterBuilder {
	if value != nil {
		b.addGTEFloat64(field, *value)
	}
	return b
}

// addInStrings appends: field IN ["val1", "val2", ...]
// This is the correct filter for filtering by a list of string IDs.
// It is a no-op when the slice is empty.
func (b *filterBuilder) addInStrings(field string, values []string) *filterBuilder {
	if len(values) == 0 {
		return b
	}
	// Build quoted, comma-separated list.
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf(`"%s"`, v)
	}
	b.clauses = append(b.clauses, fmt.Sprintf(`%s IN [%s]`, field, strings.Join(quoted, ", ")))
	return b
}

// build joins all accumulated clauses with " AND " and returns the
// final filter string ready to pass to Meilisearch.
// Returns an empty string if no clauses were added.
func (b *filterBuilder) build() string {
	if len(b.clauses) == 0 {
		return ""
	}
	// Wrap each clause in parentheses to prevent operator precedence issues
	// when Meilisearch parses complex expressions.
	wrapped := make([]string, len(b.clauses))
	for i, c := range b.clauses {
		wrapped[i] = "(" + c + ")"
	}
	return strings.Join(wrapped, " AND ")
}
