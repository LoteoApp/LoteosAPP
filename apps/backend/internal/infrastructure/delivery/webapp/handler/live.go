package handler

import (
	"net/http"

	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/system"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

func Live(w http.ResponseWriter, _ *http.Request) {
	response.WriteJSON(w, http.StatusOK, dto.StatusResponse{Status: "ok"})
}
