package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

const maxUpdateManzanaBytes = 32 << 10

type UpdateManzanaHandler struct {
	updateManzana loteos.UpdateManzana
}

func NewUpdateManzanaHandler(updateManzana loteos.UpdateManzana) *UpdateManzanaHandler {
	return &UpdateManzanaHandler{updateManzana: updateManzana}
}

func (handler *UpdateManzanaHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())

	request.Body = http.MaxBytesReader(w, request.Body, maxUpdateManzanaBytes)

	body, err := decodeJSON[dto.UpdateManzanaRequest](request)
	if err != nil {
		return err
	}

	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	manzana, err := handler.updateManzana.Execute(
		request.Context(),
		actor,
		request.PathValue("loteoId"),
		request.PathValue("manzanaId"),
		domain.ManzanaData{
			Number:   body.Number,
			HasWater: body.HasWater,
			HasSewer: body.HasSewer,
			HasPower: body.HasPower,
			HasGas:   body.HasGas,
			CalleIDs: body.CalleIDs,
		},
	)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, manzana)
	return nil
}
