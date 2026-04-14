package pagination

import (
	"math"
	"net/http"
	"strconv"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// Params holds pagination parameters.
type Params struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// Offset calculates the SQL OFFSET value.
func (p *Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// Limit returns the SQL LIMIT value (same as PageSize).
func (p *Params) Limit() int {
	return p.PerPage
}

// Response wraps paginated data with metadata the client needs
type Response struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
	HasMore    bool        `json:"has_more"`
}

// New Response creates a pagination response
func NewResponse(data interface{}, params Params, totalItems int64) *Response {
	totalPages := int(math.Ceil(float64(totalItems) / float64(params.PerPage)))

	return &Response{
		Data:       data,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasMore:    params.Page < totalPages,
	}
}

// FromRequest extracts pagination parameters from HTTP query strings.
// Example URL: /api/v1/users?page=2&per_page=50
func FromRequest(r *http.Request) Params {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return Params{
		Page:    page,
		PerPage: perPage,
	}
}
