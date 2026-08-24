package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"loteosapp/backend/internal/business/usecase/system"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/system"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type CheckDatabaseReadinessHandler struct {
	checkDatabaseReadiness system.CheckDatabaseReadiness
}

func NewCheckDatabaseReadinessHandler(checkDatabaseReadiness system.CheckDatabaseReadiness) *CheckDatabaseReadinessHandler {
	return &CheckDatabaseReadinessHandler{checkDatabaseReadiness: checkDatabaseReadiness}
}

func (handler *CheckDatabaseReadinessHandler) Ready(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()

	if err := handler.checkDatabaseReadiness.Execute(ctx); err != nil {
		slog.ErrorContext(request.Context(), "database readiness check failed", "error", err)
		response.WriteJSON(w, http.StatusServiceUnavailable, response.ErrorResponse{
			Code:    "database_unavailable",
			Message: "La base de datos no está disponible",
		})
		return
	}

	response.WriteJSON(w, http.StatusOK, dto.StatusResponse{Status: "ok"})
}
