package server

import (
	"fmt"
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/middleware"
	"github.com/Yugsolanki/standfor-me/internal/pkg/pagination"
	"github.com/Yugsolanki/standfor-me/internal/pkg/response"
	"github.com/go-chi/chi/v5"
)

// --- Request Bodies ---

type createMovementRequest struct {
	Slug             string  `json:"slug" validate:"required,min=3,max=100,slug"`
	Name             string  `json:"name" validate:"required,min=3,max=200"`
	ShortDescription string  `json:"short_description" validate:"required,min=10,max=500"`
	LongDescription  *string `json:"long_description" validate:"omitempty,max=5000"`
	ImageURL         *string `json:"image_url" validate:"omitempty,url,max=2048"`
	IconURL          *string `json:"icon_url" validate:"omitempty,url,max=2048"`
	WebsiteURL       *string `json:"website_url" validate:"omitempty,url,max=2048"`
}

type updateMovementRequest struct {
	Slug             *string `json:"slug" validate:"omitempty,min=3,max=100,slug"`
	Name             *string `json:"name" validate:"omitempty,min=3,max=200"`
	ShortDescription *string `json:"short_description" validate:"omitempty,min=10,max=500"`
	LongDescription  *string `json:"long_description" validate:"omitempty,max=5000"`
	ImageURL         *string `json:"image_url" validate:"omitempty,url,max=2048"`
	IconURL          *string `json:"icon_url" validate:"omitempty,url,max=2048"`
	WebsiteURL       *string `json:"website_url" validate:"omitempty,url,max=2048"`
}

type updateMovementStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active archived rejected pending_review"`
}

// --- Response Shapes ---

type publicMovementResponse struct {
	ID               string  `json:"id"`
	Slug             string  `json:"slug"`
	Name             string  `json:"name"`
	ShortDescription string  `json:"short_description"`
	LongDescription  *string `json:"long_description,omitempty"`
	ImageURL         *string `json:"image_url,omitempty"`
	IconURL          *string `json:"icon_url,omitempty"`
	WebsiteURL       *string `json:"website_url,omitempty"`
	SupporterCount   int     `json:"supporter_count"`
	TrendingScore    float64 `json:"trending_score"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type adminMovementResponse struct {
	publicMovementResponse
	CreatedByUserID  *string `json:"created_by_user_id,omitempty"`
	ReviewedByUserID *string `json:"reviewed_by_user_id,omitempty"`
	ReviewedAt       *string `json:"reviewed_at,omitempty"`
}

func toPublicMovementResponse(m *domain.Movement) publicMovementResponse {
	return publicMovementResponse{
		ID:               m.ID.String(),
		Slug:             m.Slug,
		Name:             m.Name,
		ShortDescription: m.ShortDescription,
		LongDescription:  m.LongDescription,
		ImageURL:         m.ImageURL,
		IconURL:          m.IconURL,
		WebsiteURL:       m.WebsiteURL,
		SupporterCount:   m.SupporterCount,
		TrendingScore:    m.TrendingScore,
		Status:           m.Status,
		CreatedAt:        m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        m.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toAdminMovementResponse(m *domain.Movement) adminMovementResponse {
	resp := adminMovementResponse{
		publicMovementResponse: toPublicMovementResponse(m),
	}
	if m.CreatedByUserID != nil {
		id := m.CreatedByUserID.String()
		resp.CreatedByUserID = &id
	}
	if m.ReviewedByUserID != nil {
		id := m.ReviewedByUserID.String()
		resp.ReviewedByUserID = &id
	}
	if m.ReviewedAt != nil {
		ts := m.ReviewedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ReviewedAt = &ts
	}
	return resp
}

// --- Public Handlers ---

// listMovementsHandler handles GET /api/v1/movements.
// Public — returns paginated list of active movements.
//
//	@Summary		List active movements
//	@Description	Returns a paginated list of active movements
//	@Tags			movements
//	@Produce		json
//	@Param			page		query		int		false	"Page number (default 1)"
//	@Param			per_page	query		int		false	"Items per page (default 20, max 100)"
//	@Param			sort_by		query		string	false	"Sort by (created_at, updated_at, supporter_count, trending_score)"
//	@Param			order		query		string	false	"Order (asc, desc)"
//	@Success		200			{object}	pagination.Response{data=[]publicMovementResponse}
//	@Failure		400			{object}	response.ErrorResponse
//	@Failure		500			{object}	response.ErrorResponse
//	@Router			/api/v1/movements [get]
func (s *Server) listMovementsHandler(w http.ResponseWriter, r *http.Request) error {
	p := pagination.FromRequest(r)

	sortBy := r.URL.Query().Get("sort_by")
	if sortBy == "" {
		sortBy = "created_at"
	}
	order := r.URL.Query().Get("order")
	if order == "" {
		order = "desc"
	}

	movementSvc := s.services.Movement

	movements, total, err := movementSvc.ListActive(r.Context(), domain.ListMovementsParams{
		Limit:  p.Limit(),
		Offset: p.Offset(),
		Status: domain.MovementStatusActive,
		SortBy: sortBy,
		Order:  order,
	})
	if err != nil {
		return err
	}

	items := make([]publicMovementResponse, len(movements))
	for i := range movements {
		items[i] = toPublicMovementResponse(&movements[i])
	}

	paginatedResponse := pagination.NewResponse(items, p, int64(total))

	response.JSON(w, r, http.StatusOK, paginatedResponse)

	return nil
}

// listTrendingMovementsHandler handles GET /api/v1/movements/trending.
// Public — returns list of movements sorted by trending score.
//
//	@Summary		List trending movements
//	@Description	Returns a list of movements sorted by trending score
//	@Tags			movements
//	@Produce		json
//	@Param			limit	query		int	false	"Number of items (default 10, max 50)"
//	@Success		200		{object}	[]publicMovementResponse
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/api/v1/movements/trending [get]
func (s *Server) listTrendingMovementsHandler(w http.ResponseWriter, r *http.Request) error {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := parsePositiveInt(l); err == nil {
			if n > 50 {
				n = 50
			}
			limit = n
		}
	}

	movementSvc := s.services.Movement

	movements, err := movementSvc.GetTrending(r.Context(), limit)
	if err != nil {
		return err
	}

	items := make([]publicMovementResponse, len(movements))
	for i := range movements {
		items[i] = toPublicMovementResponse(&movements[i])
	}

	response.JSON(w, r, http.StatusOK, items)

	return nil
}

// listPopularMovementsHandler handles GET /api/v1/movements/popular.
// Public — returns list of movements sorted by supporter count.
//
//	@Summary		List popular movements
//	@Description	Returns a list of movements sorted by supporter count
//	@Tags			movements
//	@Produce		json
//	@Param			limit	query		int	false	"Number of items (default 10, max 50)"
//	@Success		200		{object}	[]publicMovementResponse
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/api/v1/movements/popular [get]
func (s *Server) listPopularMovementsHandler(w http.ResponseWriter, r *http.Request) error {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := parsePositiveInt(l); err == nil {
			if n > 50 {
				n = 50
			}
			limit = n
		}
	}

	movementSvc := s.services.Movement

	movements, err := movementSvc.GetPopular(r.Context(), limit)
	if err != nil {
		return err
	}

	items := make([]publicMovementResponse, len(movements))
	for i := range movements {
		items[i] = toPublicMovementResponse(&movements[i])
	}

	response.JSON(w, r, http.StatusOK, items)

	return nil
}

// searchMovementsHandler handles GET /api/v1/movements/search.
// Public — searches movements by name using trigram similarity.
//
//	@Summary		Search movements
//	@Description	Searches movements by name using trigram similarity
//	@Tags			movements
//	@Produce		json
//	@Param			q			query		string	true	"Search query"
//	@Param			page		query		int		false	"Page number (default 1)"
//	@Param			per_page	query		int		false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	pagination.Response{data=[]publicMovementResponse}
//	@Failure		400			{object}	response.ErrorResponse
//	@Failure		500			{object}	response.ErrorResponse
//	@Router			/api/v1/movements/search [get]
func (s *Server) searchMovementsHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.searchMovementsHandler"

	query := r.URL.Query().Get("q")
	if query == "" {
		return domain.NewBadRequestError(op, "search query is required")
	}

	if len(query) < 2 {
		return domain.NewBadRequestError(op, "search query must be at least 2 characters")
	}

	p := pagination.FromRequest(r)

	movementSvc := s.services.Movement

	movements, total, err := movementSvc.SearchMovements(r.Context(), domain.SearchMovementsParams{
		Query:  query,
		Limit:  p.Limit(),
		Offset: p.Offset(),
	})
	if err != nil {
		return err
	}

	items := make([]publicMovementResponse, len(movements))
	for i := range movements {
		items[i] = toPublicMovementResponse(&movements[i])
	}

	paginatedResponse := pagination.NewResponse(items, p, int64(total))

	response.JSON(w, r, http.StatusOK, paginatedResponse)

	return nil
}

// getMovementBySlugHandler handles GET /api/v1/movements/{slug}.
// Public — returns a single movement by slug.
// Only active movements are returned publicly.
//
//	@Summary		Get movement by slug
//	@Description	Returns a single movement by slug
//	@Tags			movements
//	@Produce		json
//	@Param			slug	path		string	true	"Movement slug"
//	@Success		200		{object}	publicMovementResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		404		{object}	response.ErrorResponse
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/api/v1/movements/{slug} [get]
func (s *Server) getMovementBySlugHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.getMovementBySlugHandler"

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		return domain.NewBadRequestError(op, "slug is required")
	}

	movementSvc := s.services.Movement

	movement, err := movementSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		return err
	}

	claims := middleware.ClaimsFromContext(r.Context())
	isAdmin := claims != nil && (claims.Role == domain.RoleAdmin || claims.Role == domain.RoleSuperAdmin || claims.Role == domain.RoleModerator)

	if movement.Status != domain.MovementStatusActive && !isAdmin {
		return domain.NewNotFoundError(op, "movement not found")
	}

	response.JSON(w, r, http.StatusOK, toPublicMovementResponse(movement))

	return nil
}

// getMovementSupportersHandler handles GET /api/v1/movements/{slug}/supporters.
// Public — returns list of supporters for a movement.
//
//	@Summary		Get movement supporters
//	@Description	Returns a list of supporters for a movement
//	@Tags			movements
//	@Produce		json
//	@Param			slug		path		string	true	"Movement slug"
//	@Param			page		query		int		false	"Page number (default 1)"
//	@Param			per_page	query		int		false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	pagination.Response{data=[]publicUserResponse}
//	@Failure		400			{object}	response.ErrorResponse
//	@Failure		404			{object}	response.ErrorResponse
//	@Failure		500			{object}	response.ErrorResponse
//	@Router			/api/v1/movements/{slug}/supporters [get]
func (s *Server) getMovementSupportersHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.getMovementSupportersHandler"

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		return domain.NewBadRequestError(op, "slug is required")
	}

	movementSvc := s.services.Movement

	movement, err := movementSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		return err
	}

	if movement.Status != domain.MovementStatusActive {
		return domain.NewNotFoundError(op, "movement not found")
	}

	p := pagination.FromRequest(r)

	supporters, total, err := movementSvc.GetMovementSupporters(r.Context(), movement.ID, domain.ListSupportersParams{
		Limit:  p.Limit(),
		Offset: p.Offset(),
	})
	if err != nil {
		return err
	}

	items := make([]publicUserResponse, len(supporters))
	for i := range supporters {
		items[i] = toPublishUserResponse(&supporters[i])
	}

	paginatedResponse := pagination.NewResponse(items, p, int64(total))

	response.JSON(w, r, http.StatusOK, paginatedResponse)

	return nil
}

// --- Authenticated Handlers ---

// createMovementHandler handles POST /api/v1/movements.
// Authenticated — creates a new movement as pending_review.
//
//	@Summary		Create movement
//	@Description	Creates a new movement (requires approval)
//	@Tags			movements
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		createMovementRequest	true	"Movement details"
//	@Success		201		{object}	publicMovementResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Failure		422		{object}	response.ErrorResponse
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/api/v1/movements [post]
func (s *Server) createMovementHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.createMovementHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	body, err := Decode[createMovementRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		response.JSONValidationError(w, r, errs)
		return nil
	}

	exists, err := s.services.Movement.SlugExists(r.Context(), body.Slug)
	if err != nil {
		return err
	}
	if exists {
		return domain.NewConflictError(op, "a movement with this slug already exists")
	}

	userID := claims.UserID

	movement, err := s.services.Movement.Create(r.Context(), domain.CreateMovementParams{
		Slug:             body.Slug,
		Name:             body.Name,
		ShortDescription: body.ShortDescription,
		LongDescription:  body.LongDescription,
		ImageURL:         body.ImageURL,
		IconURL:          body.IconURL,
		WebsiteURL:       body.WebsiteURL,
		CreatedByUserID:  &userID,
	})
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusCreated, toPublicMovementResponse(movement))

	return nil
}

// listMyMovementsHandler handles GET /api/v1/me/movements.
// Authenticated — returns movements submitted by current user.
//
//	@Summary		Get my movements
//	@Description	Returns movements submitted by the current user
//	@Tags			movements
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number (default 1)"
//	@Param			per_page	query		int	false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	pagination.Response{data=[]adminMovementResponse}
//	@Failure		401			{object}	response.ErrorResponse
//	@Failure		500			{object}	response.ErrorResponse
//	@Router			/api/v1/me/movements [get]
func (s *Server) listMyMovementsHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.listMyMovementsHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	p := pagination.FromRequest(r)

	movements, err := s.services.Movement.GetUserMovements(r.Context(), claims.UserID)
	if err != nil {
		return err
	}

	items := make([]publicMovementResponse, len(movements))
	for i := range movements {
		items[i] = toPublicMovementResponse(&movements[i])
	}

	paginatedResponse := pagination.NewResponse(items, p, int64(len(movements)))

	response.JSON(w, r, http.StatusOK, paginatedResponse)

	return nil
}

// --- Admin Handlers ---

// adminListMovementsHandler handles GET /api/v1/admin/movements.
// Moderator+ — returns paginated list of all movements (all statuses).
//
//	@Summary		List all movements (admin)
//	@Description	Returns a paginated list of all movements with admin details
//	@Tags			admin-movements
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int		false	"Page number (default 1)"
//	@Param			per_page	query		int		false	"Items per page (default 20, max 100)"
//	@Param			status		query		string	false	"Filter by status"
//	@Success		200			{object}	pagination.Response{data=[]adminMovementResponse}
//	@Failure		401			{object}	response.ErrorResponse
//	@Failure		403			{object}	response.ErrorResponse
//	@Failure		500			{object}	response.ErrorResponse
//	@Router			/api/v1/admin/movements [get]
func (s *Server) adminListMovementsHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.adminListMovementsHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	if claims.Role != domain.RoleModerator && claims.Role != domain.RoleAdmin && claims.Role != domain.RoleSuperAdmin {
		return domain.NewForbiddenError(op, "only moderators and admins can list movements")
	}

	p := pagination.FromRequest(r)
	status := r.URL.Query().Get("status")

	movementSvc := s.services.Movement

	var movements []domain.Movement
	var total int
	var err error

	if status != "" {
		movements, total, err = movementSvc.List(r.Context(), domain.ListMovementsParams{
			Limit:  p.Limit(),
			Offset: p.Offset(),
			Status: status,
		})
	} else {
		movements, total, err = movementSvc.List(r.Context(), domain.ListMovementsParams{
			Limit:  p.Limit(),
			Offset: p.Offset(),
		})
	}
	if err != nil {
		return err
	}

	items := make([]adminMovementResponse, len(movements))
	for i := range movements {
		items[i] = toAdminMovementResponse(&movements[i])
	}

	paginatedResponse := pagination.NewResponse(items, p, int64(total))

	response.JSON(w, r, http.StatusOK, paginatedResponse)

	return nil
}

// adminListPendingMovementsHandler handles GET /api/v1/admin/movements/pending.
// Moderator+ — returns paginated list of movements pending review.
//
//	@Summary		List pending movements (admin)
//	@Description	Returns a paginated list of movements pending review
//	@Tags			admin-movements
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number (default 1)"
//	@Param			per_page	query		int	false	"Items per page (default 20, max 100)"
//	@Success		200			{object}	pagination.Response{data=[]adminMovementResponse}
//	@Failure		401			{object}	response.ErrorResponse
//	@Failure		403			{object}	response.ErrorResponse
//	@Failure		500			{object}	response.ErrorResponse
//	@Router			/api/v1/admin/movements/pending [get]
func (s *Server) adminListPendingMovementsHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.adminListPendingMovementsHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	if claims.Role != domain.RoleModerator && claims.Role != domain.RoleAdmin && claims.Role != domain.RoleSuperAdmin {
		return domain.NewForbiddenError(op, "only moderators and admins can list pending movements")
	}

	p := pagination.FromRequest(r)

	movements, total, err := s.services.Movement.ListPendingReview(r.Context(), domain.ListMovementsParams{
		Limit:  p.Limit(),
		Offset: p.Offset(),
	})
	if err != nil {
		return err
	}

	items := make([]adminMovementResponse, len(movements))
	for i := range movements {
		items[i] = toAdminMovementResponse(&movements[i])
	}

	paginatedResponse := pagination.NewResponse(items, p, int64(total))

	response.JSON(w, r, http.StatusOK, paginatedResponse)

	return nil
}

// adminGetMovementHandler handles GET /api/v1/admin/movements/{id}.
// Moderator+ — returns any movement by ID.
//
//	@Summary		Get movement by ID (admin)
//	@Description	Returns full admin details for a movement by ID
//	@Tags			admin-movements
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Movement UUID"
//	@Success		200	{object}	adminMovementResponse
//	@Failure		400	{object}	response.ErrorResponse
//	@Failure		401	{object}	response.ErrorResponse
//	@Failure		403	{object}	response.ErrorResponse
//	@Failure		404	{object}	response.ErrorResponse
//	@Failure		500	{object}	response.ErrorResponse
//	@Router			/api/v1/admin/movements/{id} [get]
func (s *Server) adminGetMovementHandler(w http.ResponseWriter, r *http.Request) error {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		return err
	}

	movementSvc := s.services.Movement

	movement, err := movementSvc.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusOK, toAdminMovementResponse(movement))

	return nil
}

// adminUpdateMovementHandler handles PATCH /api/v1/admin/movements/{id}.
// Moderator+ — updates movement details.
//
//	@Summary		Update movement (admin)
//	@Description	Updates movement details
//	@Tags			admin-movements
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Movement UUID"
//	@Param			request	body		updateMovementRequest	true	"Fields to update"
//	@Success		200		{object}	adminMovementResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Failure		403		{object}	response.ErrorResponse
//	@Failure		404		{object}	response.ErrorResponse
//	@Failure		422		{object}	response.ErrorResponse
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/api/v1/admin/movements/{id} [patch]
func (s *Server) adminUpdateMovementHandler(w http.ResponseWriter, r *http.Request) error {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		return err
	}

	body, err := Decode[updateMovementRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		response.JSONValidationError(w, r, errs)
		return nil
	}

	movementSvc := s.services.Movement

	movement, err := movementSvc.Update(r.Context(), id, domain.UpdateMovementParams{
		Slug:             body.Slug,
		Name:             body.Name,
		ShortDescription: body.ShortDescription,
		LongDescription:  body.LongDescription,
		ImageURL:         body.ImageURL,
		IconURL:          body.IconURL,
		WebsiteURL:       body.WebsiteURL,
	})
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusOK, toAdminMovementResponse(movement))

	return nil
}

// adminUpdateMovementStatusHandler handles PATCH /api/v1/admin/movements/{id}/status.
// Moderator+ — approves, rejects, or archives a movement.
//
//	@Summary		Update movement status (admin)
//	@Description	Changes a movement's status (active, archived, rejected, pending_review)
//	@Tags			admin-movements
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"Movement UUID"
//	@Param			request	body		updateMovementStatusRequest	true	"New status"
//	@Success		200		{object}	adminMovementResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Failure		403		{object}	response.ErrorResponse
//	@Failure		404		{object}	response.ErrorResponse
//	@Failure		422		{object}	response.ErrorResponse
//	@Failure		500		{object}	response.ErrorResponse
//	@Router			/api/v1/admin/movements/{id}/status [patch]
func (s *Server) adminUpdateMovementStatusHandler(w http.ResponseWriter, r *http.Request) error {
	const op = "server.adminUpdateMovementStatusHandler"

	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return domain.NewUnauthorizedError(op, "authentication required")
	}

	id, err := parseUUIDParam(r, "id")
	if err != nil {
		return err
	}

	body, err := Decode[updateMovementStatusRequest](r)
	if err != nil {
		return err
	}

	if errs := s.validator.Validate(body); errs != nil {
		response.JSONValidationError(w, r, errs)
		return nil
	}

	movementSvc := s.services.Movement

	var movement *domain.Movement
	switch body.Status {
	case domain.MovementStatusActive:
		movement, err = movementSvc.Review(r.Context(), id, claims.UserID, true)
	case domain.MovementStatusRejected:
		movement, err = movementSvc.Review(r.Context(), id, claims.UserID, false)
	default:
		movement, err = movementSvc.UpdateStatus(r.Context(), id, body.Status)
	}
	if err != nil {
		return err
	}

	response.JSON(w, r, http.StatusOK, toAdminMovementResponse(movement))

	return nil
}

// adminDeleteMovementHandler handles DELETE /api/v1/admin/movements/{id}.
// Moderator+ — soft-deletes a movement.
//
//	@Summary		Delete movement (admin)
//	@Description	Soft-deletes a movement
//	@Tags			admin-movements
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Movement UUID"
//	@Success		200	{object}	response.SuccessResponse
//	@Failure		400	{object}	response.ErrorResponse
//	@Failure		401	{object}	response.ErrorResponse
//	@Failure		403	{object}	response.ErrorResponse
//	@Failure		404	{object}	response.ErrorResponse
//	@Failure		500	{object}	response.ErrorResponse
//	@Router			/api/v1/admin/movements/{id} [delete]
func (s *Server) adminDeleteMovementHandler(w http.ResponseWriter, r *http.Request) error {
	id, err := parseUUIDParam(r, "id")
	if err != nil {
		return err
	}

	movementSvc := s.services.Movement

	if err = movementSvc.SoftDelete(r.Context(), id); err != nil {
		return err
	}

	response.JSONMessage(w, r, http.StatusOK, "movement deleted")

	return nil
}

// --- Helpers ---

func parsePositiveInt(s string) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("must be positive")
	}
	return n, nil
}
