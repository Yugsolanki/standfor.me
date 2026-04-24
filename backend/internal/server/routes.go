package server

import (
	"net/http"

	_ "github.com/Yugsolanki/standfor-me/docs"
	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/Yugsolanki/standfor-me/internal/middleware"
	internaljwt "github.com/Yugsolanki/standfor-me/internal/pkg/jwt"
	"github.com/Yugsolanki/standfor-me/internal/pkg/response"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (s *Server) setupRoutes(jwtSvc *internaljwt.Service) {
	s.router.Get("/", s.rootHandler)
	s.router.Get("/health", s.healthHandler)

	// --- Swagger UI ---
	s.router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	s.router.Route("/api/v1", func(r chi.Router) {
		// --- Search ---
		r.Route("/search", func(r chi.Router) {
			r.Get("/movements", s.handle(s.SearchMovements))
			r.Get("/users", s.handle(s.SearchUsers))
			r.Get("/organizations", s.handle(s.SearchOrganizations))
		})

		// --- Authentication ---
		r.Route("/auth", func(r chi.Router) {
			// Public endpoints
			r.Post("/register", s.handle(s.registerHandler))
			r.Post("/login", s.handle(s.loginHandler))
			r.Post("/refresh-token", s.handle(s.refreshHandler))

			// Authenticated endpoints
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(jwtSvc))
				// * Implementation to be done with background workers and Resend.
				// TODO: r.Post("/forgot-password", s.handle(s.forgotPasswordHandler))
				// TODO: r.Post("/verify-email", s.handle(s.verifyEmailHandler))
				// TODO: r.Post("/resend-verification-email", s.handle(s.resendVerificationEmailHandler))
				r.Post("/reset-password", s.handle(s.changePasswordHandler))
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
				r.Get("/me", s.handle(s.meHandler))
				r.Patch("/me", s.handle(s.updateMeHandler))
				r.Delete("/me", s.handle(s.deleteMeHandler))
				r.Post("/me/password", s.handle(s.changePasswordHandler))
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

		// --- Movements ---
		r.Route("/movements", func(r chi.Router) {
			// Public endpoints
			r.Get("/", s.handle(s.listMovementsHandler))
			r.Get("/trending", s.handle(s.listTrendingMovementsHandler))
			r.Get("/popular", s.handle(s.listPopularMovementsHandler))
			r.Get("/search", s.handle(s.searchMovementsHandler))
			r.Get("/{slug}", s.handle(s.getMovementBySlugHandler))
			r.Get("/{slug}/supporters", s.handle(s.getMovementSupportersHandler))

			// Authenticated endpoints
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAuth(jwtSvc))
				r.Post("/", s.handle(s.createMovementHandler))
			})
		})

		// --- User Movements (authenticated) ---
		r.Route("/me/movements", func(r chi.Router) {
			r.Use(middleware.RequireAuth(jwtSvc))
			r.Get("/", s.handle(s.listMyMovementsHandler))
		})

		// --- Admin Movements ---
		r.Route("/admin/movements", func(r chi.Router) {
			r.Use(middleware.RequireAuth(jwtSvc))
			r.Use(middleware.RequireMinRole(domain.RoleModerator))

			r.Get("/", s.handle(s.adminListMovementsHandler))
			r.Get("/pending", s.handle(s.adminListPendingMovementsHandler))
			r.Get("/{id}", s.handle(s.adminGetMovementHandler))
			r.Patch("/{id}", s.handle(s.adminUpdateMovementHandler))
			r.Patch("/{id}/status", s.handle(s.adminUpdateMovementStatusHandler))
			r.Delete("/{id}", s.handle(s.adminDeleteMovementHandler))
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
