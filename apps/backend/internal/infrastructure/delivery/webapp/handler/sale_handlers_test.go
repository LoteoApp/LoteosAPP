package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/sales"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type createSaleHandlerStub struct {
	input  sales.CreateSaleInput
	result domain.Sale
	err    error
}

func (stub *createSaleHandlerStub) Execute(_ context.Context, input sales.CreateSaleInput) (domain.Sale, error) {
	stub.input = input
	return stub.result, stub.err
}

type listSalesHandlerStub struct {
	input  sales.ListSalesInput
	result domain.SalePage
	err    error
}

func (stub *listSalesHandlerStub) Execute(_ context.Context, input sales.ListSalesInput) (domain.SalePage, error) {
	stub.input = input
	return stub.result, stub.err
}

type getSaleHandlerStub struct {
	actor  sales.Actor
	id     string
	result domain.Sale
	err    error
}

func (stub *getSaleHandlerStub) Execute(_ context.Context, actor sales.Actor, id string) (domain.Sale, error) {
	stub.actor = actor
	stub.id = id
	return stub.result, stub.err
}

func TestCreateSaleHandler(t *testing.T) {
	stub := &createSaleHandlerStub{result: domain.Sale{ID: "sale-1", Monto: 100000, Moneda: "USD"}}
	mux := reservationHandlerMux(t, http.MethodPost, "/api/v1/loteos/{loteoId}/lotes/{loteId}/ventas", handler.NewCreateSaleHandler(stub))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/loteos/development-1/lotes/lot-1/ventas", strings.NewReader(`{"clienteId":"client-1","vendedorId":"seller-1","modalidadPago":"contado"}`))
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if stub.input.LoteoID != "development-1" || stub.input.LoteID != "lot-1" || stub.input.ClienteID != "client-1" || stub.input.VendedorID != "seller-1" || stub.input.PaymentMethod != "contado" {
		t.Errorf("input = %#v", stub.input)
	}
	if stub.input.Actor.AuthProviderID != reservationHandlerPrincipal.Subject || len(stub.input.Actor.Roles) != 1 {
		t.Errorf("actor = %#v", stub.input.Actor)
	}
	var got struct {
		ID    string  `json:"id"`
		Monto float64 `json:"monto"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "sale-1" || got.Monto != 100000 {
		t.Errorf("response = %#v", got)
	}
	if stub.input.PaymentPlan != nil {
		t.Errorf("plan = %#v, want none for contado", stub.input.PaymentPlan)
	}
}

func TestCreateSaleHandlerPassesThePaymentPlan(t *testing.T) {
	due := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	stub := &createSaleHandlerStub{result: domain.Sale{
		ID: "sale-2", Monto: 100000, Moneda: "USD", ModalidadPago: domain.PaymentMethodDownAndFi,
		PlanPago: &domain.PaymentPlan{
			ID: "plan-1", MontoEntrega: 20000, CantidadCuotas: 2, TasaInteres: 10, Periodicidad: domain.PaymentPeriodMonthly,
			Moneda: "USD", MontoFinanciado: 80000, MontoCuota: 44000, MontoTotal: 88000,
			Cuotas: []domain.Installment{
				{ID: "c-1", Numero: 1, Monto: 44000, Estado: domain.InstallmentStatePending, FechaVencimiento: due},
				{ID: "c-2", Numero: 2, Monto: 44000, Estado: domain.InstallmentStatePending, FechaVencimiento: due.AddDate(0, 1, 0)},
			},
		},
	}}
	mux := reservationHandlerMux(t, http.MethodPost, "/api/v1/loteos/{loteoId}/lotes/{loteId}/ventas", handler.NewCreateSaleHandler(stub))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/loteos/development-1/lotes/lot-1/ventas", strings.NewReader(
		`{"clienteId":"client-1","vendedorId":"seller-1","modalidadPago":"entrega_financiada",
		  "planPago":{"cantidadCuotas":2,"tasaInteres":10,"periodicidad":"mensual","montoEntrega":20000}}`))
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	want := sales.PaymentPlanInput{CantidadCuotas: 2, TasaInteres: 10, Periodicidad: "mensual", MontoEntrega: 20000}
	if stub.input.PaymentMethod != "entrega_financiada" || stub.input.PaymentPlan == nil || *stub.input.PaymentPlan != want {
		t.Errorf("input = %#v, plan = %#v", stub.input, stub.input.PaymentPlan)
	}
	var got struct {
		PlanPago struct {
			CantidadCuotas int     `json:"cantidadCuotas"`
			MontoEntrega   float64 `json:"montoEntrega"`
			TasaInteres    float64 `json:"tasaInteres"`
			Periodicidad   string  `json:"periodicidad"`
			MontoCuota     float64 `json:"montoCuota"`
			MontoTotal     float64 `json:"montoTotal"`
			Cuotas         []struct {
				Numero           int     `json:"numero"`
				Monto            float64 `json:"monto"`
				Estado           string  `json:"estado"`
				FechaVencimiento string  `json:"fechaVencimiento"`
			} `json:"cuotas"`
		} `json:"planPago"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	plan := got.PlanPago
	if plan.CantidadCuotas != 2 || plan.MontoEntrega != 20000 || plan.TasaInteres != 10 || plan.Periodicidad != "mensual" || plan.MontoCuota != 44000 || plan.MontoTotal != 88000 {
		t.Errorf("response plan = %#v", plan)
	}
	if len(plan.Cuotas) != 2 || plan.Cuotas[0].Numero != 1 || plan.Cuotas[0].Estado != "pendiente" || plan.Cuotas[0].FechaVencimiento != "2026-10-15T00:00:00Z" {
		t.Errorf("response cuotas = %#v", plan.Cuotas)
	}
}

func TestCreateSaleHandlerRejectsInvalidBodyAndMapsErrors(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		stub := &createSaleHandlerStub{}
		mux := reservationHandlerMux(t, http.MethodPost, "/api/v1/loteos/{loteoId}/lotes/{loteId}/ventas", handler.NewCreateSaleHandler(stub))
		request := httptest.NewRequest(http.MethodPost, "/api/v1/loteos/development-1/lotes/lot-1/ventas", strings.NewReader("not-json"))
		request.Header.Set("Authorization", "Bearer token")
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if stub.input.LoteoID != "" {
			t.Error("use case should not be called with an invalid body")
		}
	})

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "lot unavailable", err: domain.ErrSaleLotUnavailable, wantStatus: http.StatusConflict, wantCode: "sale_lot_unavailable"},
		{name: "seller not eligible", err: domain.ErrSaleSellerNotEligible, wantStatus: http.StatusForbidden, wantCode: "sale_seller_not_eligible"},
		{name: "payment plan required", err: domain.ErrSalePaymentPlanRequired, wantStatus: http.StatusBadRequest, wantCode: "sale_payment_plan_required"},
		{name: "invalid down payment", err: domain.ErrSaleInvalidDownPayment, wantStatus: http.StatusBadRequest, wantCode: "invalid_sale_down_payment"},
		{name: "unexpected", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stub := &createSaleHandlerStub{err: test.err}
			mux := reservationHandlerMux(t, http.MethodPost, "/api/v1/loteos/{loteoId}/lotes/{loteId}/ventas", handler.NewCreateSaleHandler(stub))
			recorder := performAuthorizedRequest(t, mux, http.MethodPost, "/api/v1/loteos/development-1/lotes/lot-1/ventas", map[string]string{"clienteId": "client-1", "vendedorId": "seller-1"})

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			var got response.ErrorResponse
			if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.Code != test.wantCode {
				t.Errorf("error code = %q, want %q", got.Code, test.wantCode)
			}
		})
	}
}

func TestSaleHandlersParseRequestsAndPrincipal(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		stub := &listSalesHandlerStub{result: domain.SalePage{Page: 2, Limit: 10, Items: []domain.Sale{}}}
		mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/ventas", handler.NewListSalesHandler(stub))
		recorder := performAuthorizedRequest(t, mux, http.MethodGet, "/api/v1/ventas?estado=activa,cancelada&loteoId=development-1&loteId=lot-1&q=Ana&pagina=2&porPagina=10", nil)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		filter := stub.input.Filter
		if len(filter.States) != 2 || filter.States[0] != domain.SaleStateActive || filter.States[1] != domain.SaleStateCancelled {
			t.Errorf("states = %#v", filter.States)
		}
		if filter.LoteoID != "development-1" || filter.LoteID != "lot-1" || filter.Search != "Ana" || filter.Page != 2 || filter.Limit != 10 {
			t.Errorf("filter = %#v", filter)
		}
		if stub.input.Actor.AuthProviderID != reservationHandlerPrincipal.Subject {
			t.Errorf("actor = %#v", stub.input.Actor)
		}
		if !strings.Contains(recorder.Body.String(), `"ventas":[]`) {
			t.Errorf("body = %s", recorder.Body.String())
		}
	})

	t.Run("list rejects a non-numeric page", func(t *testing.T) {
		for _, query := range []string{"pagina=dos", "porPagina=x"} {
			stub := &listSalesHandlerStub{}
			mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/ventas", handler.NewListSalesHandler(stub))
			recorder := performAuthorizedRequest(t, mux, http.MethodGet, "/api/v1/ventas?"+query, nil)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("%s: status = %d, want %d", query, recorder.Code, http.StatusBadRequest)
			}
			if !strings.Contains(recorder.Body.String(), `"code":"invalid_sale_page"`) {
				t.Errorf("%s: body = %s", query, recorder.Body.String())
			}
		}
	})

	t.Run("detail", func(t *testing.T) {
		stub := &getSaleHandlerStub{result: domain.Sale{ID: "sale-1"}}
		mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/ventas/{id}", handler.NewGetSaleHandler(stub))
		recorder := performAuthorizedRequest(t, mux, http.MethodGet, "/api/v1/ventas/sale-1", nil)

		if recorder.Code != http.StatusOK || stub.id != "sale-1" || stub.actor.AuthProviderID != reservationHandlerPrincipal.Subject {
			t.Errorf("status = %d, id = %q, actor = %#v", recorder.Code, stub.id, stub.actor)
		}
	})

	t.Run("detail not found", func(t *testing.T) {
		stub := &getSaleHandlerStub{err: domain.ErrSaleNotFound}
		mux := reservationHandlerMux(t, http.MethodGet, "/api/v1/ventas/{id}", handler.NewGetSaleHandler(stub))
		recorder := performAuthorizedRequest(t, mux, http.MethodGet, "/api/v1/ventas/missing", nil)

		if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), `"code":"sale_not_found"`) {
			t.Errorf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
	})
}
