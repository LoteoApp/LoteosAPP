package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/sales"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
)

const convertReservationRoute = "/api/v1/reservas/{id}/convertir"

type convertReservationHandlerStub struct {
	calls  int
	input  sales.ConvertReservationToSaleInput
	result domain.Sale
	err    error
}

func (stub *convertReservationHandlerStub) Execute(_ context.Context, input sales.ConvertReservationToSaleInput) (domain.Sale, error) {
	stub.calls++
	stub.input = input
	return stub.result, stub.err
}

func serveConversion(t *testing.T, stub *convertReservationHandlerStub, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := reservationHandlerMux(t, http.MethodPost, convertReservationRoute, handler.NewConvertReservationToSaleHandler(stub))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/reservas/reservation-1/convertir", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Idempotency-Key", "convert-key-1")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	return recorder
}

func TestConvertReservationToSaleHandlerReturnsTheCreatedSale(t *testing.T) {
	reservationID := "reservation-1"
	stub := &convertReservationHandlerStub{result: domain.Sale{ID: "sale-1", Monto: 100000, Moneda: "USD", ReservaID: &reservationID}}
	recorder := serveConversion(t, stub, `{
		"modalidadPago":"financiado",
		"planPago":{"cantidadCuotas":12,"tasaInteres":0,"periodicidad":"mensual","montoEntrega":0},
		"clienteId":"tampered-client","vendedorId":"tampered-seller","loteId":"tampered-lot","monto":1,"moneda":"ARS"
	}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	input := stub.input
	if input.ReservationID != "reservation-1" || input.IdempotencyKey != "convert-key-1" || input.PaymentMethod != "financiado" {
		t.Errorf("input = %#v", input)
	}
	if input.Actor.AuthProviderID != reservationHandlerPrincipal.Subject || len(input.Actor.Roles) != 1 {
		t.Errorf("actor = %#v", input.Actor)
	}
	if input.PaymentPlan == nil || input.PaymentPlan.Installments != 12 || input.PaymentPlan.Period != "mensual" {
		t.Errorf("plan = %#v", input.PaymentPlan)
	}
	var got struct {
		ID        string  `json:"id"`
		ReservaID string  `json:"reservaId"`
		Monto     float64 `json:"monto"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "sale-1" || got.ReservaID != "reservation-1" || got.Monto != 100000 {
		t.Errorf("response = %#v", got)
	}
}

func TestConvertReservationToSaleHandlerMapsErrors(t *testing.T) {
	for name, test := range map[string]struct {
		body   string
		err    error
		status int
		code   string
	}{
		"malformed json":  {body: `{"modalidadPago":`, status: http.StatusBadRequest, code: "invalid_body"},
		"invalid plan":    {body: `{}`, err: domain.ErrSalePaymentPlanRequired, status: http.StatusBadRequest, code: "sale_payment_plan_required"},
		"not the seller":  {body: `{}`, err: domain.ErrReservationConvertForbidden, status: http.StatusForbidden, code: "reservation_convert_forbidden"},
		"out of scope":    {body: `{}`, err: domain.ErrReservationNotFound, status: http.StatusNotFound, code: "reservation_not_found"},
		"expired":         {body: `{}`, err: domain.ErrReservationConversionExpired, status: http.StatusConflict, code: "reservation_conversion_expired"},
		"converted":       {body: `{}`, err: domain.ErrReservationConverted, status: http.StatusConflict, code: "reservation_converted"},
		"key reused":      {body: `{}`, err: domain.ErrSaleIdempotencyConflict, status: http.StatusConflict, code: "idempotency_key_conflict"},
		"database failed": {body: `{}`, err: domain.ErrDatabaseUnavailable.WithCause(errors.New("SQLSTATE 08006 secret detail")), status: http.StatusServiceUnavailable, code: "database_unavailable"},
	} {
		t.Run(name, func(t *testing.T) {
			stub := &convertReservationHandlerStub{err: test.err}
			recorder := serveConversion(t, stub, test.body)
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d, body = %s", recorder.Code, test.status, recorder.Body.String())
			}
			var got struct {
				Code string `json:"code"`
			}
			if err := json.NewDecoder(strings.NewReader(recorder.Body.String())).Decode(&got); err != nil {
				t.Fatalf("decode error body: %v", err)
			}
			if got.Code != test.code {
				t.Errorf("code = %q, want %q", got.Code, test.code)
			}
			if strings.Contains(recorder.Body.String(), "SQLSTATE") {
				t.Errorf("body leaks the cause: %s", recorder.Body.String())
			}
		})
	}
}
