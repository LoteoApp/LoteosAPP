package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/sales"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type GetSaleHandler struct {
	getSale sales.GetSale
}

func NewGetSaleHandler(getSale sales.GetSale) *GetSaleHandler {
	return &GetSaleHandler{getSale: getSale}
}

func (handler *GetSaleHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	sale, err := handler.getSale.Execute(request.Context(), sales.Actor{
		AuthProviderID: principal.Subject,
		Roles:          principal.Roles,
	}, request.PathValue("id"))
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusOK, sale)
	return nil
}
