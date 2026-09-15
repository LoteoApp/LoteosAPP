package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/sales"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/sales"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type CreateSaleHandler struct {
	createSale sales.CreateSale
}

func NewCreateSaleHandler(createSale sales.CreateSale) *CreateSaleHandler {
	return &CreateSaleHandler{createSale: createSale}
}

func (handler *CreateSaleHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	body, err := decodeJSON[dto.CreateSaleRequest](request)
	if err != nil {
		return err
	}
	sale, err := handler.createSale.Execute(request.Context(), sales.CreateSaleInput{
		Actor:          sales.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		LoteoID:        request.PathValue("loteoId"),
		LoteID:         request.PathValue("loteId"),
		ClienteID:      body.ClienteID,
		VendedorID:     body.VendedorID,
		PaymentMethod:  body.ModalidadPago,
		IdempotencyKey: request.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusCreated, sale)
	return nil
}
