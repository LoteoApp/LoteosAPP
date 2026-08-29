package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

// This body carries the values of a single lot, so it's orders of magnitude
// smaller than a plan. The cap stops a caller from making the decoder
// allocate before the domain gets to check the field lengths.
const maxUpdateLoteBytes = 32 << 10

type UpdateLoteHandler struct {
	updateLote loteos.UpdateLote
}

func NewUpdateLoteHandler(updateLote loteos.UpdateLote) *UpdateLoteHandler {
	return &UpdateLoteHandler{updateLote: updateLote}
}

// Handle loads the values a lot only gets by hand. It must run behind
// middleware.RequireAuth.
func (handler *UpdateLoteHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	request.Body = http.MaxBytesReader(w, request.Body, maxUpdateLoteBytes)

	body, err := decodeJSON[dto.UpdateLoteRequest](request)
	if err != nil {
		return err
	}

	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	lote, err := handler.updateLote.Execute(
		request.Context(),
		actor,
		request.PathValue("loteoId"),
		request.PathValue("loteId"),
		domain.LoteData{
			Number:   body.Number,
			Price:    body.Price,
			Currency: body.Currency,
			Area:     body.Area,
			Features: body.Features,
		},
	)
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusOK, lote)
	return nil
}
