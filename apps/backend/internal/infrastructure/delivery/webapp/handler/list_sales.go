package handler

import (
	"net/http"
	"strconv"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/sales"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListSalesHandler struct {
	listSales sales.ListSales
}

func NewListSalesHandler(listSales sales.ListSales) *ListSalesHandler {
	return &ListSalesHandler{listSales: listSales}
}

func (handler *ListSalesHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	filter, err := saleFilter(request)
	if err != nil {
		return err
	}
	page, err := handler.listSales.Execute(request.Context(), sales.ListSalesInput{
		Actor:  sales.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		Filter: filter,
	})
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusOK, page)
	return nil
}

func saleFilter(request *http.Request) (domain.SaleListFilter, error) {
	query := request.URL.Query()
	filter := domain.SaleListFilter{
		LoteoID: query.Get("loteoId"),
		LoteID:  query.Get("loteId"),
		Search:  query.Get("q"),
	}
	for _, rawState := range query["estado"] {
		for _, state := range strings.Split(rawState, ",") {
			filter.States = append(filter.States, domain.SaleState(strings.TrimSpace(state)))
		}
	}
	if raw := query.Get("pagina"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return domain.SaleListFilter{}, domain.ErrSaleInvalidPage
		}
		filter.Page = value
	}
	if raw := query.Get("porPagina"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return domain.SaleListFilter{}, domain.ErrSaleInvalidPage
		}
		filter.Limit = value
	}
	return filter, nil
}
