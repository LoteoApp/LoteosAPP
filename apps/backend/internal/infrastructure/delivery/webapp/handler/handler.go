package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type service interface {
	Info(ctx context.Context) (domain.Info, error)
	Ready(ctx context.Context) error
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Live(w http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (handler *Handler) Ready(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()

	if err := handler.service.Ready(ctx); err != nil {
		slog.ErrorContext(request.Context(), "database readiness check failed", "error", err)
		response.WriteJSON(w, http.StatusServiceUnavailable, response.ErrorResponse{
			Code:    "database_unavailable",
			Message: "La base de datos no está disponible",
		})
		return
	}

	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (handler *Handler) Info(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()

	info, err := handler.service.Info(ctx)
	if err != nil {
		slog.ErrorContext(request.Context(), "database diagnostic failed", "error", err)
		response.WriteJSON(w, http.StatusServiceUnavailable, response.ErrorResponse{
			Code:    "database_diagnostic_failed",
			Message: "No se pudo consultar PostgreSQL",
		})
		return
	}

	response.WriteJSON(w, http.StatusOK, info)
}
