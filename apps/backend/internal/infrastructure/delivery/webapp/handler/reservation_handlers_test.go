package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/reservations"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

var reservationHandlerPrincipal = supabase.Principal{
	Subject: "auth-user-1",
	Roles:   []string{domain.RolAdministrativo},
}

type createReservationHandlerStub struct {
	input  reservations.CreateReservationInput
	result domain.Reservation
	err    error
}

func (stub *createReservationHandlerStub) Execute(_ context.Context, input reservations.CreateReservationInput) (domain.Reservation, error) {
	stub.input = input
	return stub.result, stub.err
}

type listReservationsHandlerStub struct {
	input  reservations.ListReservationsInput
	result domain.ReservationPage
	err    error
}

func (stub *listReservationsHandlerStub) Execute(_ context.Context, input reservations.ListReservationsInput) (domain.ReservationPage, error) {
	stub.input = input
	return stub.result, stub.err
}

type getReservationHandlerStub struct {
	actor  reservations.Actor
	id     string
	result domain.Reservation
	err    error
}

func (stub *getReservationHandlerStub) Execute(_ context.Context, actor reservations.Actor, id string) (domain.Reservation, error) {
	stub.actor = actor
	stub.id = id
	return stub.result, stub.err
}

type cancelReservationHandlerStub struct {
	input  reservations.CancelReservationInput
	result domain.Reservation
	err    error
}

func (stub *cancelReservationHandlerStub) Execute(_ context.Context, input reservations.CancelReservationInput) (domain.Reservation, error) {
	stub.input = input
	return stub.result, stub.err
}

type listEligibleSellersHandlerStub struct {
	input  reservations.ListEligibleSellersInput
	result []domain.SellerOption
	err    error
}

func (stub *listEligibleSellersHandlerStub) Execute(_ context.Context, input reservations.ListEligibleSellersInput) ([]domain.SellerOption, error) {
	stub.input = input
	return stub.result, stub.err
}

func reservationHandlerMux(t *testing.T, method, pattern string, routeHandler handler.HTTPHandler) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	verifier := userVerifierStub{principal: reservationHandlerPrincipal}
	mux.Handle(method+" "+pattern, middleware.RequireAuth(verifier)(handler.Adapt(routeHandler, time.Second)))
	return mux
}

func TestCreateReservationHandler(t *testing.T) {
	stub := &createReservationHandlerStub{result: domain.Reservation{ID: "reservation-1"}}
	mux := reservationHandlerMux(t, http.MethodPost, "/api/v1/loteos/{loteoId}/lotes/{loteId}/reservas", handler.NewCreateReservationHandler(stub))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/loteos/development-1/lotes/lot-1/reservas", strings.NewReader(`{"clienteId":"client-1","vendedorId":"seller-1"}`))
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Idempotency-Key", "request-1")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if stub.input.LoteoID != "development-1" || stub.input.LoteID != "lot-1" || stub.input.ClienteID != "client-1" || stub.input.VendedorID != "seller-1" || stub.input.IdempotencyKey != "request-1" {
		t.Errorf("input = %#v", stub.input)
	}
}

func TestReservationHandlersParseRequestsAndPrincipal(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		stub := &listReservationsHandlerStub{result: domain.ReservationPage{Page: 2, Limit: 10}}
		mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/reservas", handler.NewListReservationsHandler(stub))
		recorder := performAuthorizedRequest(t, mux, http.MethodGet, "/api/v1/reservas?estado=activa,vencida&loteoId=development-1&q=Ana&pagina=2&porPagina=10", nil)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if len(stub.input.Filter.States) != 2 || stub.input.Filter.States[0] != domain.ReservationStateActive || stub.input.Filter.States[1] != domain.ReservationStateExpired || stub.input.Filter.Page != 2 || stub.input.Filter.Limit != 10 {
			t.Errorf("filter = %#v", stub.input.Filter)
		}
		if stub.input.Actor.AuthProviderID != reservationHandlerPrincipal.Subject {
			t.Errorf("actor = %#v", stub.input.Actor)
		}
	})

	t.Run("detail", func(t *testing.T) {
		stub := &getReservationHandlerStub{}
		mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/reservas/{id}", handler.NewGetReservationHandler(stub))
		recorder := performAuthorizedRequest(t, mux, http.MethodGet, "/api/v1/reservas/reservation-1", nil)

		if recorder.Code != http.StatusOK || stub.id != "reservation-1" || stub.actor.AuthProviderID != reservationHandlerPrincipal.Subject {
			t.Errorf("status = %d, id = %q, actor = %#v", recorder.Code, stub.id, stub.actor)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		stub := &cancelReservationHandlerStub{}
		mux := reservationHandlerMux(t, http.MethodPost, "/api/v1/reservas/{id}/cancelar", handler.NewCancelReservationHandler(stub))
		recorder := performAuthorizedRequest(t, mux, http.MethodPost, "/api/v1/reservas/reservation-1/cancelar", map[string]string{"razon": "Cliente desistió"})

		if recorder.Code != http.StatusOK || stub.input.ReservationID != "reservation-1" || stub.input.Reason != "Cliente desistió" || stub.input.Actor.AuthProviderID != reservationHandlerPrincipal.Subject {
			t.Errorf("status = %d, input = %#v", recorder.Code, stub.input)
		}
	})

	t.Run("eligible sellers", func(t *testing.T) {
		stub := &listEligibleSellersHandlerStub{result: []domain.SellerOption{{ID: "seller-1"}}}
		mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/loteos/{loteoId}/vendedores", handler.NewListEligibleSellersHandler(stub))
		recorder := performAuthorizedRequest(t, mux, http.MethodGet, "/api/v1/loteos/development-1/vendedores", nil)

		if recorder.Code != http.StatusOK || stub.input.LoteoID != "development-1" || len(stub.result) != 1 {
			t.Errorf("status = %d, input = %#v, result = %#v", recorder.Code, stub.input, stub.result)
		}
	})
}

func TestCreateReservationHandlerRejectsUnauthenticatedRequest(t *testing.T) {
	stub := &createReservationHandlerStub{}
	mux := reservationHandlerMux(t, http.MethodPost, "/api/v1/loteos/{loteoId}/lotes/{loteId}/reservas", handler.NewCreateReservationHandler(stub))
	recorder := performRequest(t, mux, http.MethodPost, "/api/v1/loteos/development-1/lotes/lot-1/reservas", "", map[string]string{"clienteId": "client-1"})

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if stub.input.ClienteID != "" {
		t.Error("handler should not execute without authentication")
	}
}

func performAuthorizedRequest(t *testing.T, mux *http.ServeMux, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return performRequest(t, mux, method, path, "token", body)
}
