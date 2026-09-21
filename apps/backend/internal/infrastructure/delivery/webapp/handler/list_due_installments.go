package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/collections"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type ListDueInstallmentsHandler struct {
	listDueInstallments collections.ListDueInstallments
}

func NewListDueInstallmentsHandler(listDueInstallments collections.ListDueInstallments) *ListDueInstallmentsHandler {
	return &ListDueInstallmentsHandler{listDueInstallments: listDueInstallments}
}

func (handler *ListDueInstallmentsHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	principal, _ := middleware.PrincipalFromContext(request.Context())
	filter, err := dueInstallmentFilter(request)
	if err != nil {
		return err
	}
	page, err := handler.listDueInstallments.Execute(request.Context(), collections.ListDueInstallmentsInput{
		Actor:  collections.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles},
		Filter: filter,
	})
	if err != nil {
		return err
	}
	response.WriteJSON(w, http.StatusOK, page)
	return nil
}

// dueInstallmentFilter reads the query: `desde`/`hasta` are calendar dates
// (YYYY-MM-DD) covering their whole day, or RFC 3339 instants.
func dueInstallmentFilter(request *http.Request) (domain.DueInstallmentFilter, error) {
	query := request.URL.Query()
	filter := domain.DueInstallmentFilter{
		DevelopmentID: query.Get("loteoId"),
		Search:        query.Get("q"),
	}
	for _, rawState := range query["estado"] {
		for _, state := range strings.Split(rawState, ",") {
			if state = strings.TrimSpace(state); state != "" {
				filter.States = append(filter.States, domain.InstallmentState(state))
			}
		}
	}
	var err error
	if filter.From, err = parseDueBoundary(query.Get("desde"), false); err != nil {
		return domain.DueInstallmentFilter{}, err
	}
	if filter.To, err = parseDueBoundary(query.Get("hasta"), true); err != nil {
		return domain.DueInstallmentFilter{}, err
	}
	if raw := query.Get("pagina"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return domain.DueInstallmentFilter{}, domain.ErrDueInstallmentInvalidPage
		}
		filter.Page = value
	}
	if raw := query.Get("porPagina"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return domain.DueInstallmentFilter{}, domain.ErrDueInstallmentInvalidPage
		}
		filter.Limit = value
	}
	return filter, nil
}

func parseDueBoundary(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if value, err := time.Parse(time.RFC3339, raw); err == nil {
		value = value.UTC()
		return &value, nil
	}
	day, err := time.ParseInLocation("2006-01-02", raw, receiptArgentinaLocation)
	if err != nil {
		return nil, domain.ErrDueInstallmentInvalidPeriod
	}
	if endOfDay {
		day = day.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	day = day.UTC()
	return &day, nil
}
