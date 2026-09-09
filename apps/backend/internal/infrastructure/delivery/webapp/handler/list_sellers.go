package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/reservations"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/reservations"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListEligibleSellersHandler struct {
	listEligibleSellers reservations.ListEligibleSellers
}

func NewListEligibleSellersHandler(listEligibleSellers reservations.ListEligibleSellers) *ListEligibleSellersHandler {
	return &ListEligibleSellersHandler{listEligibleSellers: listEligibleSellers}
}

func (handler *ListEligibleSellersHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	sellers, err := handler.listEligibleSellers.Execute(request.Context(), reservations.ListEligibleSellersInput{
		Actor:   reservations.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		LoteoID: request.PathValue("loteoId"),
	})
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusOK, dto.SellersResponse{Vendedores: sellers})
	return nil
}
