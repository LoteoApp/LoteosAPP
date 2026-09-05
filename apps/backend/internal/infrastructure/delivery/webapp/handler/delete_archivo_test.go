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

type deleteArchivoStub struct {
	err          error
	gotLoteoID   string
	gotArchivoID string
}

func (stub *deleteArchivoStub) Execute(_ context.Context, _ loteos.Actor, loteoID, archivoID string) error {
	stub.gotLoteoID = loteoID
	stub.gotArchivoID = archivoID
	return stub.err
}

func deleteArchivoMux(stub *deleteArchivoStub) *http.ServeMux {
	h := handler.NewDeleteArchivoHandler(stub)
	requireAuth := middleware.RequireAuth(administradorVerifier())
	mux := http.NewServeMux()
	mux.Handle("DELETE /api/v1/loteos/{loteoId}/archivos/{archivoId}", requireAuth(handler.Adapt(h, 5*time.Second)))
	return mux
}

func TestDeleteArchivoHandlerDeletesTheArchivo(t *testing.T) {
	stub := &deleteArchivoStub{}
	mux := deleteArchivoMux(stub)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/loteos/loteo-9/archivos/archivo-1", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if stub.gotLoteoID != "loteo-9" || stub.gotArchivoID != "archivo-1" {
		t.Fatalf("ids = (%q, %q), want (loteo-9, archivo-1)", stub.gotLoteoID, stub.gotArchivoID)
	}
}

func TestDeleteArchivoHandlerMapsANotFound(t *testing.T) {
	stub := &deleteArchivoStub{err: domain.ErrArchivoNotFound}
	mux := deleteArchivoMux(stub)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/loteos/loteo-9/archivos/archivo-1", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestDeleteArchivoHandlerRejectsRequestsWithoutAToken(t *testing.T) {
	stub := &deleteArchivoStub{}
	mux := deleteArchivoMux(stub)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/loteos/loteo-9/archivos/archivo-1", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
