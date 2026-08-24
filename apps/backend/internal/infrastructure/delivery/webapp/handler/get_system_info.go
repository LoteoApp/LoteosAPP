package handler

import (
	"context"
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

func (handler *GetSystemInfoHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()

	info, err := handler.getSystemInfo.Execute(ctx)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, info)
	return nil
}
