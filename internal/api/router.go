package api

import (
	"encoding/json"
	"net/http"

	_ "github.com/blaisecz/sleep-tracker/docs"
	"github.com/blaisecz/sleep-tracker/internal/api/handler"
	"github.com/blaisecz/sleep-tracker/internal/api/middleware"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Router struct {
	userHandler     *handler.UserHandler
	sleepLogHandler *handler.SleepLogHandler
	insightsHandler *handler.InsightsHandler
}

func NewRouter(userHandler *handler.UserHandler, sleepLogHandler *handler.SleepLogHandler, insightsHandler *handler.InsightsHandler) *Router {
	return &Router{
		userHandler:     userHandler,
		sleepLogHandler: sleepLogHandler,
		insightsHandler: insightsHandler,
	}
}

func (rt *Router) Setup() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Recovery)
	r.Use(middleware.Tracing)
	r.Use(middleware.Logger)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("list"),
		httpSwagger.DomID("swagger-ui"),
	))

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Users
		r.Route("/users", func(r chi.Router) {
			r.Post("/", rt.userHandler.Create)
			r.Get("/{userId}", rt.userHandler.GetByID)

			// Sleep logs (nested under users)
			r.Route("/{userId}/sleep-logs", func(r chi.Router) {
				r.Post("/", rt.sleepLogHandler.Create)
				r.Get("/", rt.sleepLogHandler.List)
				r.Put("/{logId}", rt.sleepLogHandler.Update)
			})

			// Sleep insights (nested under users)
			r.Route("/{userId}/sleep", func(r chi.Router) {
				r.Get("/chronotype", rt.insightsHandler.GetChronotype)
				r.Get("/metrics", rt.insightsHandler.GetMetrics)
				r.Get("/insights", rt.insightsHandler.GetInsights)
				r.Post("/insights/feedback", rt.insightsHandler.PostFeedback)
			})
		})
	})

	return r
}
