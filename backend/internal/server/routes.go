package server

import (
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/middleware"
	internaljwt "github.com/Yugsolanki/standfor-me/internal/pkg/jwt"
	"github.com/Yugsolanki/standfor-me/internal/pkg/response"
	"github.com/go-chi/chi/v5"
)

func (s *Server) setupRoutes(jwtSvc *internaljwt.Service) {
	s.router.Get("/", s.rootHandler)
	s.router.Get("/health", s.healthHandler)

	s.router.Route("/api/v1", func(r chi.Router) {
		// --- Authentication ---
		r.Route("/auth", func(r chi.Router) {
			// Public endpoints
			r.Post("/register", s.handle(s.registerHandler))
			r.Post("/login", s.handle(s.loginHandler))
			r.Post("/refresh", s.handle(s.refreshHandler))

			// Authenticated endpoints
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(jwtSvc))
				r.Post("/logout", s.handle(s.logoutHandler))
				r.Post("/logout-all", s.handle(s.logoutAllHandler))
				r.Get("/me", s.handle(s.meHandler))
			})
		})

		// --- Users ---
		r.Route("/users", func(r chi.Router) {
			// Public profile lookup, visibility enforced inside handler.
			r.Get("/{username}", s.handle(s.getUserHandler))

			// Authenticated self-service
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(jwtSvc))
				r.Patch("/me", s.handle(s.updateMeHandler))
				r.Post("/me/password", s.handle(s.changePasswordHandler))
				r.Delete("/me", s.handle(s.deleteMeHandler))
			})
		})

		// --- Admin ---
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.RequireAuth(jwtSvc))

			// Moderator+ — read access and status changes
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireMinRole(domain.RoleModerator))
				r.Get("/users", s.handle(s.adminListUsersHandler))
				r.Get("/users/{id}", s.handle(s.adminGetUserHandler))
				r.Patch("/users/{id}/status", s.handle(s.adminUpdateStatusHandler))
			})

			// Admin+ — role assignment and hard delete
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireMinRole(domain.RoleAdmin))
				r.Patch("/users/{id}/role", s.handle(s.adminUpdateRoleHandler))
				r.Delete("/users/{id}", s.handle(s.adminDeleteUserHandler))
			})
		})
	})

	// Catch-all for unmatched routes
	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.JSONError(w, r, domain.NewNotFoundError("router", "route not found"))
	})
	s.router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		response.JSONError(w, r, domain.NewBadRequestError("router", "method not allowed"))
	})
}
