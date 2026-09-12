package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/reservations"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type GetReservationHandler struct {
	getReservation reservations.GetReservation
}

func NewGetReservationHandler(getReservation reservations.GetReservation) *GetReservationHandler {
	return &GetReservationHandler{getReservation: getReservation}
}

func (handler *GetReservationHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	reservation, err := handler.getReservation.Execute(request.Context(), reservations.Actor{
		AuthProviderID: principal.Subject,
		Roles:          principal.Roles,
	}, request.PathValue("id"))
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusOK, reservation)
	return nil
}
