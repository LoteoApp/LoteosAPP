package handler

import (
	"net/http"
	"strings"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/collections"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/collections"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type RegisterPaymentHandler struct {
	registerPayment collections.RegisterPayment
}

func NewRegisterPaymentHandler(registerPayment collections.RegisterPayment) *RegisterPaymentHandler {
	return &RegisterPaymentHandler{registerPayment: registerPayment}
}

func (handler *RegisterPaymentHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	body, err := decodeJSON[dto.RegisterPaymentRequest](request)
	if err != nil {
		return err
	}
	paidAt, err := parsePaymentDate(body.FechaPago)
	if err != nil {
		return err
	}
	payment, err := handler.registerPayment.Execute(request.Context(), collections.RegisterPaymentInput{
		Actor:              collections.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		SaleID:             request.PathValue("id"),
		InstallmentIDs:     body.CuotaIDs,
		IncludeDownPayment: body.IncluirEntrega,
		Medium:             body.MedioPago,
		PaidAt:             paidAt,
		Observation:        body.Observacion,
	})
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusCreated, payment)
	return nil
}

func parsePaymentDate(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, domain.ErrPaymentInvalidDate.WithCause(err)
	}
	return &value, nil
}
