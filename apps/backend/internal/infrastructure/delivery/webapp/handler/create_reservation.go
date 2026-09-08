package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/reservations"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/reservations"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type CreateReservationHandler struct {
	createReservation reservations.CreateReservation
}

func NewCreateReservationHandler(createReservation reservations.CreateReservation) *CreateReservationHandler {
	return &CreateReservationHandler{createReservation: createReservation}
}

func (handler *CreateReservationHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	body, err := decodeJSON[dto.CreateReservationRequest](request)
	if err != nil {
		return err
	}
	reservation, err := handler.createReservation.Execute(request.Context(), reservations.CreateReservationInput{
		Actor:          reservations.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		LoteoID:        request.PathValue("loteoId"),
		LoteID:         request.PathValue("loteId"),
		ClienteID:      body.ClienteID,
		VendedorID:     body.VendedorID,
		IdempotencyKey: request.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusCreated, reservation)
	return nil
}
