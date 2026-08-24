package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"loteosapp/backend/internal/business/usecase/system"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type GetSystemInfoHandler struct {
	getSystemInfo system.GetSystemInfo
}

func NewGetSystemInfoHandler(getSystemInfo system.GetSystemInfo) *GetSystemInfoHandler {
	return &GetSystemInfoHandler{getSystemInfo: getSystemInfo}
}

func (handler *GetSystemInfoHandler) Info(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()

	info, err := handler.getSystemInfo.Execute(ctx)
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
