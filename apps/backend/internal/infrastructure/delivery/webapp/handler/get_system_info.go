package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/system"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/system"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type GetSystemInfoHandler struct {
	getSystemInfo system.GetSystemInfo
}

func NewGetSystemInfoHandler(getSystemInfo system.GetSystemInfo) *GetSystemInfoHandler {
	return &GetSystemInfoHandler{getSystemInfo: getSystemInfo}
}

func (handler *GetSystemInfoHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	info, err := handler.getSystemInfo.Execute(request.Context())
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, dto.GetSystemInfoResponse{Info: info})
	return nil
}
