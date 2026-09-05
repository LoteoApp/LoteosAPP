package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type listLoteoArchivosStub struct {
	archivos   []domain.Archivo
	err        error
	gotLoteoID string
}

func (stub *listLoteoArchivosStub) Execute(_ context.Context, _ loteos.Actor, loteoID string) ([]domain.Archivo, error) {
	stub.gotLoteoID = loteoID
	return stub.archivos, stub.err
}

func TestListLoteoArchivosHandlerListsTheArchivos(t *testing.T) {
	stub := &listLoteoArchivosStub{archivos: []domain.Archivo{{ID: "archivo-1", Categoria: "foto"}}}
	h := handler.NewListLoteoArchivosHandler(stub)
	requireAuth := middleware.RequireAuth(administradorVerifier())
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/loteos/{loteoId}/archivos", requireAuth(handler.Adapt(h, 5*time.Second)))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/loteos/loteo-9/archivos", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", recorder.Code, http.StatusOK, recorder.Body)
	}
	if stub.gotLoteoID != "loteo-9" {
		t.Fatalf("loteo id = %q, want loteo-9", stub.gotLoteoID)
	}

	var payload struct {
		Archivos []map[string]any `json:"archivos"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(payload.Archivos) != 1 || payload.Archivos[0]["id"] != "archivo-1" {
		t.Fatalf("archivos = %#v, want one archivo-1", payload.Archivos)
	}
}

func TestListLoteoArchivosHandlerMapsANotFound(t *testing.T) {
	stub := &listLoteoArchivosStub{err: domain.ErrLoteoNotFound}
	h := handler.NewListLoteoArchivosHandler(stub)
	requireAuth := middleware.RequireAuth(administradorVerifier())
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/loteos/{loteoId}/archivos", requireAuth(handler.Adapt(h, 5*time.Second)))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/loteos/loteo-9/archivos", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
