package handler

import (
	"context"
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

func (handler *CheckDatabaseReadinessHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()

	if err := handler.checkDatabaseReadiness.Execute(ctx); err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.StatusResponse{Status: "ok"})
	return nil
}
