package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/collections"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type getDebtStatementStub struct {
	actor  collections.Actor
	id     string
	result domain.DebtStatement
	err    error
}

func (stub *getDebtStatementStub) Execute(_ context.Context, actor collections.Actor, id string) (domain.DebtStatement, error) {
	stub.actor = actor
	stub.id = id
	return stub.result, stub.err
}

type listDueInstallmentsStub struct {
	input  collections.ListDueInstallmentsInput
	result domain.DueInstallmentPage
	err    error
}

func (stub *listDueInstallmentsStub) Execute(_ context.Context, input collections.ListDueInstallmentsInput) (domain.DueInstallmentPage, error) {
	stub.input = input
	return stub.result, stub.err
}

type registerPaymentStub struct {
	input  collections.RegisterPaymentInput
	calls  int
	result domain.Payment
	err    error
}

func (stub *registerPaymentStub) Execute(_ context.Context, input collections.RegisterPaymentInput) (domain.Payment, error) {
	stub.calls++
	stub.input = input
	return stub.result, stub.err
}

type settleSaleStub struct {
	input  collections.SettleSaleInput
	calls  int
	result domain.Payment
	err    error
}

func (stub *settleSaleStub) Execute(_ context.Context, input collections.SettleSaleInput) (domain.Payment, error) {
	stub.calls++
	stub.input = input
	return stub.result, stub.err
}

func TestGetDebtStatementHandler(t *testing.T) {
	stub := &getDebtStatementStub{result: domain.DebtStatement{
		Venta:   domain.Sale{ID: "sale-1"},
		Entrega: &domain.DownPayment{Monto: 40000, Estado: domain.InstallmentStatePending},
		Cuotas:  []domain.Installment{{ID: "c-1", Numero: 1, Monto: 20000, Estado: domain.InstallmentStateOverdue}},
		Resumen: domain.DebtSummary{MontoTotal: 60000, MontoPendiente: 60000, MontoVencido: 20000, CuotasVencidas: 1},
		Cobros:  []domain.Payment{},
	}}
	mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/ventas/{id}/estado-deuda", handler.NewGetDebtStatementHandler(stub))
	recorder := performRequest(t, mux, http.MethodGet, "/api/v1/ventas/sale-1/estado-deuda", "token", nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if stub.id != "sale-1" || stub.actor.AuthProviderID != reservationHandlerPrincipal.Subject {
		t.Errorf("stub = %#v", stub)
	}
	var got struct {
		Venta   struct{ ID string }
		Entrega struct {
			Monto  float64
			Estado string
		}
		Cuotas  []struct{ Estado string }
		Resumen struct{ MontoVencido float64 }
		Cobros  []any
	}
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Venta.ID != "sale-1" || got.Entrega.Monto != 40000 || got.Entrega.Estado != "pendiente" || len(got.Cuotas) != 1 || got.Cuotas[0].Estado != "vencida" || got.Resumen.MontoVencido != 20000 || got.Cobros == nil {
		t.Errorf("response = %#v", got)
	}

	stub.err = domain.ErrSaleNotFinanced
	recorder = performRequest(t, mux, http.MethodGet, "/api/v1/ventas/sale-1/estado-deuda", "token", nil)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("not financed status = %d, want 400", recorder.Code)
	}
	stub.err = domain.ErrSaleNotFound
	recorder = performRequest(t, mux, http.MethodGet, "/api/v1/ventas/sale-1/estado-deuda", "token", nil)
	if recorder.Code != http.StatusNotFound {
		t.Errorf("not found status = %d, want 404", recorder.Code)
	}
}

func TestListDueInstallmentsHandlerParsesTheQuery(t *testing.T) {
	due := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	stub := &listDueInstallmentsStub{result: domain.DueInstallmentPage{
		Items:   []domain.DueInstallment{{ID: "c-1", VentaID: "sale-1", Numero: 2, Monto: 100, Moneda: "USD", Estado: domain.InstallmentStatePending, FechaVencimiento: due, LoteoNombre: "Las Acacias"}},
		Resumen: domain.DueSummary{CuotasVencidas: 3, CuotasProximas: 2},
		Page:    2, Limit: 10, Total: 11, TotalPages: 2,
	}}
	mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/cobranzas/vencimientos", handler.NewListDueInstallmentsHandler(stub))
	recorder := performRequest(t, mux, http.MethodGet, "/api/v1/cobranzas/vencimientos?estado=vencida,pendiente&loteoId=loteo-1&q=ana&desde=2026-10-01&hasta=2026-10-31&pagina=2&porPagina=10", "token", nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	filter := stub.input.Filter
	if len(filter.States) != 2 || filter.States[0] != domain.InstallmentStateOverdue || filter.States[1] != domain.InstallmentStatePending {
		t.Errorf("states = %#v", filter.States)
	}
	if filter.DevelopmentID != "loteo-1" || filter.Search != "ana" || filter.Page != 2 || filter.Limit != 10 {
		t.Errorf("filter = %#v", filter)
	}
	// Calendar dates cover the whole day in Argentina (UTC-3).
	wantFrom := time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 11, 1, 3, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
	if filter.From == nil || !filter.From.Equal(wantFrom) || filter.To == nil || !filter.To.Equal(wantTo) {
		t.Errorf("period = %v .. %v, want %s .. %s", filter.From, filter.To, wantFrom, wantTo)
	}
	if stub.input.Actor.AuthProviderID != reservationHandlerPrincipal.Subject {
		t.Errorf("actor = %#v", stub.input.Actor)
	}
	var got struct {
		Cuotas  []struct{ LoteoNombre string }
		Resumen struct{ CuotasVencidas int }
		Pagina  int
		Total   int
	}
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Cuotas) != 1 || got.Cuotas[0].LoteoNombre != "Las Acacias" || got.Resumen.CuotasVencidas != 3 || got.Pagina != 2 || got.Total != 11 {
		t.Errorf("response = %#v", got)
	}

	recorder = performRequest(t, mux, http.MethodGet, "/api/v1/cobranzas/vencimientos?desde=2026-10-01T12:00:00-03:00", "token", nil)
	if recorder.Code != http.StatusOK || stub.input.Filter.From == nil || !stub.input.Filter.From.Equal(time.Date(2026, 10, 1, 15, 0, 0, 0, time.UTC)) {
		t.Errorf("RFC 3339 desde = %v (status %d)", stub.input.Filter.From, recorder.Code)
	}
	recorder = performRequest(t, mux, http.MethodGet, "/api/v1/cobranzas/vencimientos", "token", nil)
	if recorder.Code != http.StatusOK || stub.input.Filter.From != nil || stub.input.Filter.To != nil || len(stub.input.Filter.States) != 0 {
		t.Errorf("bare query filter = %#v (status %d)", stub.input.Filter, recorder.Code)
	}
}

func TestListDueInstallmentsHandlerRejectsBadQueries(t *testing.T) {
	stub := &listDueInstallmentsStub{}
	mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/cobranzas/vencimientos", handler.NewListDueInstallmentsHandler(stub))
	for _, query := range []string{"?desde=ayer", "?hasta=31/10/2026", "?pagina=dos", "?porPagina=x"} {
		recorder := performRequest(t, mux, http.MethodGet, "/api/v1/cobranzas/vencimientos"+query, "token", nil)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s status = %d, want 400", query, recorder.Code)
		}
	}
	stub.err = domain.ErrNoAutorizado
	recorder := performRequest(t, mux, http.MethodGet, "/api/v1/cobranzas/vencimientos", "token", nil)
	if recorder.Code != http.StatusForbidden {
		t.Errorf("forbidden status = %d, want 403", recorder.Code)
	}
}

func TestRegisterPaymentHandler(t *testing.T) {
	paidAt := time.Date(2026, 9, 20, 15, 0, 0, 0, time.UTC)
	stub := &registerPaymentStub{result: domain.Payment{ID: "cobro-1", Monto: 60000, Moneda: "USD", IncluyeEntrega: true, Cuotas: []domain.Installment{{Numero: 1}}}}
	mux := reservationHandlerMux(t, http.MethodPost, "/api/v1/ventas/{id}/cobros", handler.NewRegisterPaymentHandler(stub))
	recorder := performRequest(t, mux, http.MethodPost, "/api/v1/ventas/sale-1/cobros", "token",
		`{"cuotaIds":["c-1","c-2"],"incluirEntrega":true,"medioPago":"transferencia","fechaPago":"2026-09-20T12:00:00-03:00","observacion":"Comprobante 5"}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	input := stub.input
	if input.SaleID != "sale-1" || len(input.InstallmentIDs) != 2 || !input.IncludeDownPayment || input.Medium != "transferencia" || input.Observation != "Comprobante 5" {
		t.Errorf("input = %#v", input)
	}
	if input.PaidAt == nil || !input.PaidAt.Equal(paidAt) {
		t.Errorf("paid at = %v, want %s", input.PaidAt, paidAt)
	}
	if input.Actor.AuthProviderID != reservationHandlerPrincipal.Subject || len(input.Actor.Roles) != 1 {
		t.Errorf("actor = %#v", input.Actor)
	}
	var got struct {
		ID             string
		Monto          float64
		IncluyeEntrega bool
		Cuotas         []struct{ Numero int }
	}
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != "cobro-1" || got.Monto != 60000 || !got.IncluyeEntrega || len(got.Cuotas) != 1 {
		t.Errorf("response = %#v", got)
	}

	recorder = performRequest(t, mux, http.MethodPost, "/api/v1/ventas/sale-1/cobros", "token", `{"cuotaIds":["c-1"],"medioPago":"efectivo"}`)
	if recorder.Code != http.StatusCreated || stub.input.PaidAt != nil {
		t.Errorf("without date: status = %d, paid at = %v", recorder.Code, stub.input.PaidAt)
	}
}

func TestRegisterPaymentHandlerRejectsBadBodies(t *testing.T) {
	stub := &registerPaymentStub{}
	mux := reservationHandlerMux(t, http.MethodPost, "/api/v1/ventas/{id}/cobros", handler.NewRegisterPaymentHandler(stub))

	recorder := performRequest(t, mux, http.MethodPost, "/api/v1/ventas/sale-1/cobros", "token", `{"cuotaIds":`)
	if recorder.Code != http.StatusBadRequest || stub.calls != 0 {
		t.Errorf("malformed body: status = %d, calls = %d", recorder.Code, stub.calls)
	}
	recorder = performRequest(t, mux, http.MethodPost, "/api/v1/ventas/sale-1/cobros", "token", `{"cuotaIds":["c-1"],"medioPago":"efectivo","fechaPago":"ayer"}`)
	if recorder.Code != http.StatusBadRequest || stub.calls != 0 {
		t.Errorf("bad date: status = %d, calls = %d", recorder.Code, stub.calls)
	}
	var body response.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != domain.ErrPaymentInvalidDate.Code {
		t.Errorf("error code = %q, want %q", body.Code, domain.ErrPaymentInvalidDate.Code)
	}

	for _, tc := range []struct {
		err    error
		status int
	}{
		{domain.ErrPaymentInstallmentOrder, http.StatusBadRequest},
		{domain.ErrPaymentInstallmentPaid, http.StatusConflict},
		{domain.ErrSaleNotActive, http.StatusConflict},
		{domain.ErrSaleNotFound, http.StatusNotFound},
		{domain.ErrNoAutorizado, http.StatusForbidden},
	} {
		stub.err = tc.err
		recorder = performRequest(t, mux, http.MethodPost, "/api/v1/ventas/sale-1/cobros", "token", `{"cuotaIds":["c-1"],"medioPago":"efectivo"}`)
		if recorder.Code != tc.status {
			t.Errorf("%v status = %d, want %d", tc.err, recorder.Code, tc.status)
		}
	}
}

func TestSettleSaleHandler(t *testing.T) {
	stub := &settleSaleStub{result: domain.Payment{ID: "cobro-2", Tipo: domain.PaymentTypeSettlement, Monto: 40000}}
	mux := reservationHandlerMux(t, http.MethodPost, "/api/v1/ventas/{id}/cancelacion-total", handler.NewSettleSaleHandler(stub))
	recorder := performRequest(t, mux, http.MethodPost, "/api/v1/ventas/sale-1/cancelacion-total", "token",
		`{"montoEsperado":40000,"medioPago":"cheque","fechaPago":"2026-09-20T15:00:00Z","observacion":"Cheque 9"}`)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	input := stub.input
	if input.SaleID != "sale-1" || input.ExpectedAmount == nil || *input.ExpectedAmount != 40000 || input.Medium != "cheque" || input.Observation != "Cheque 9" || input.PaidAt == nil {
		t.Errorf("input = %#v", input)
	}
	if !strings.Contains(recorder.Body.String(), `"tipo":"cancelacion_total"`) {
		t.Errorf("body = %s", recorder.Body.String())
	}

	recorder = performRequest(t, mux, http.MethodPost, "/api/v1/ventas/sale-1/cancelacion-total", "token", `{"medioPago":"efectivo"}`)
	if recorder.Code != http.StatusCreated || stub.input.ExpectedAmount != nil || stub.input.PaidAt != nil {
		t.Errorf("minimal body: status = %d, input = %#v", recorder.Code, stub.input)
	}

	recorder = performRequest(t, mux, http.MethodPost, "/api/v1/ventas/sale-1/cancelacion-total", "token", `not json`)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("malformed body status = %d", recorder.Code)
	}
	recorder = performRequest(t, mux, http.MethodPost, "/api/v1/ventas/sale-1/cancelacion-total", "token", `{"medioPago":"efectivo","fechaPago":"2026-13-40"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("bad date status = %d", recorder.Code)
	}
	stub.err = domain.ErrSettlementAmountMismatch
	recorder = performRequest(t, mux, http.MethodPost, "/api/v1/ventas/sale-1/cancelacion-total", "token", `{"medioPago":"efectivo"}`)
	if recorder.Code != http.StatusConflict {
		t.Errorf("mismatch status = %d, want 409", recorder.Code)
	}
}

func TestCollectionHandlersRequireAuth(t *testing.T) {
	mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/cobranzas/vencimientos", handler.NewListDueInstallmentsHandler(&listDueInstallmentsStub{}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cobranzas/vencimientos", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", recorder.Code)
	}
}
