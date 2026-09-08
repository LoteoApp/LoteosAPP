package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/reservations"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/reservations"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type CancelReservationHandler struct {
	cancelReservation reservations.CancelReservation
}

func NewCancelReservationHandler(cancelReservation reservations.CancelReservation) *CancelReservationHandler {
	return &CancelReservationHandler{cancelReservation: cancelReservation}
}

func (handler *CancelReservationHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	body, err := decodeJSON[dto.CancelReservationRequest](request)
	if err != nil {
		return err
	}
	reservation, err := handler.cancelReservation.Execute(request.Context(), reservations.CancelReservationInput{
		Actor:         reservations.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		ReservationID: request.PathValue("id"),
		Reason:        body.Razon,
	})
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusOK, reservation)
	return nil
}
