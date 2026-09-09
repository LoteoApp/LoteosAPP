package handler

import (
	"net/http"
	"strconv"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/reservations"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListReservationsHandler struct {
	listReservations reservations.ListReservations
}

func NewListReservationsHandler(listReservations reservations.ListReservations) *ListReservationsHandler {
	return &ListReservationsHandler{listReservations: listReservations}
}

func (handler *ListReservationsHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	filter, err := reservationFilter(request)
	if err != nil {
		return err
	}
	page, err := handler.listReservations.Execute(request.Context(), reservations.ListReservationsInput{
		Actor:  reservations.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		Filter: filter,
	})
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusOK, page)
	return nil
}

func reservationFilter(request *http.Request) (domain.ReservationListFilter, error) {
	query := request.URL.Query()
	filter := domain.ReservationListFilter{
		LoteoID: query.Get("loteoId"),
		LoteID:  query.Get("loteId"),
		Search:  query.Get("q"),
	}
	for _, rawState := range query["estado"] {
		for _, state := range strings.Split(rawState, ",") {
			filter.States = append(filter.States, domain.ReservationState(strings.TrimSpace(state)))
		}
	}
	if raw := query.Get("pagina"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return domain.ReservationListFilter{}, domain.ErrReservationInvalidPage
		}
		filter.Page = value
	}
	if raw := query.Get("porPagina"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return domain.ReservationListFilter{}, domain.ErrReservationInvalidPage
		}
		filter.Limit = value
	}
	return filter, nil
}
