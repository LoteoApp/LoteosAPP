package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type listLoteArchivosStub struct {
	archivos   []domain.Archivo
	err        error
	gotLoteoID string
	gotLoteID  string
}

func (stub *listLoteArchivosStub) Execute(
	_ context.Context, _ loteos.Actor, loteoID, loteID string,
) ([]domain.Archivo, error) {
	stub.gotLoteoID = loteoID
	stub.gotLoteID = loteID
	return stub.archivos, stub.err
}

func TestListLoteArchivosHandlerListsTheArchivos(t *testing.T) {
	stub := &listLoteArchivosStub{archivos: []domain.Archivo{{ID: "archivo-1"}}}
	h := handler.NewListLoteArchivosHandler(stub)
	requireAuth := middleware.RequireAuth(administradorVerifier())
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/loteos/{loteoId}/lotes/{loteId}/archivos", requireAuth(handler.Adapt(h, 5*time.Second)))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/loteos/loteo-9/lotes/lote-3/archivos", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", recorder.Code, http.StatusOK, recorder.Body)
	}
	if stub.gotLoteoID != "loteo-9" || stub.gotLoteID != "lote-3" {
		t.Fatalf("ids = (%q, %q), want (loteo-9, lote-3)", stub.gotLoteoID, stub.gotLoteID)
	}
}
