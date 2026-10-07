package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/sales"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/sales"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ConvertReservationToSaleHandler struct {
	convertReservation sales.ConvertReservationToSale
}

func NewConvertReservationToSaleHandler(convertReservation sales.ConvertReservationToSale) *ConvertReservationToSaleHandler {
	return &ConvertReservationToSaleHandler{convertReservation: convertReservation}
}

func (handler *ConvertReservationToSaleHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	body, err := decodeJSON[dto.ConvertReservationRequest](request)
	if err != nil {
		return err
	}
	sale, err := handler.convertReservation.Execute(request.Context(), sales.ConvertReservationToSaleInput{
		Actor:          sales.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		ReservationID:  request.PathValue("id"),
		PaymentMethod:  body.ModalidadPago,
		IdempotencyKey: request.Header.Get("Idempotency-Key"),
		PaymentPlan:    paymentPlanInput(body.PlanPago),
	})
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusCreated, sale)
	return nil
}
