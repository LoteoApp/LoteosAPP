package systemhttp

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"loteosapp/backend/internal/platform/httpx"
	"loteosapp/backend/internal/system"
)

type service interface {
	Info(ctx context.Context) (system.Info, error)
	Ready(ctx context.Context) error
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", handler.live)
	mux.HandleFunc("GET /readyz", handler.ready)
	mux.HandleFunc("GET /api/v1/system", handler.info)
}

func (handler *Handler) live(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (handler *Handler) ready(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()

	if err := handler.service.Ready(ctx); err != nil {
		slog.ErrorContext(request.Context(), "database readiness check failed", "error", err)
		httpx.WriteJSON(w, http.StatusServiceUnavailable, httpx.ErrorResponse{
			Code:    "database_unavailable",
			Message: "La base de datos no está disponible",
		})
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (handler *Handler) info(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()

	info, err := handler.service.Info(ctx)
	if err != nil {
		slog.ErrorContext(request.Context(), "database diagnostic failed", "error", err)
		httpx.WriteJSON(w, http.StatusServiceUnavailable, httpx.ErrorResponse{
			Code:    "database_diagnostic_failed",
			Message: "No se pudo consultar PostgreSQL",
		})
		return
	}

	httpx.WriteJSON(w, http.StatusOK, info)
}
