package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/collections"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/collections"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type SettleSaleHandler struct {
	settleSale collections.SettleSale
}

func NewSettleSaleHandler(settleSale collections.SettleSale) *SettleSaleHandler {
	return &SettleSaleHandler{settleSale: settleSale}
}

func (handler *SettleSaleHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	body, err := decodeJSON[dto.SettleSaleRequest](request)
	if err != nil {
		return err
	}
	paidAt, err := parsePaymentDate(body.PaidAt)
	if err != nil {
		return err
	}
	payment, err := handler.settleSale.Execute(request.Context(), collections.SettleSaleInput{
		Actor:          collections.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		SaleID:         request.PathValue("id"),
		ExpectedAmount: body.ExpectedAmount,
		Charges:        paymentCharges(body.Charges),
		Medium:         body.Medium,
		PaidAt:         paidAt,
		Observation:    body.Observation,
	})
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusCreated, payment)
	return nil
}
