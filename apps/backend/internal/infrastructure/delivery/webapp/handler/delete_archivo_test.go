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

type deleteFileStub struct {
	err        error
	gotLoteoID string
	gotFileID  string
}

func (stub *deleteFileStub) Execute(_ context.Context, _ loteos.Actor, loteoID, archivoID string) error {
	stub.gotLoteoID = loteoID
	stub.gotFileID = archivoID
	return stub.err
}

func deleteFileMux(stub *deleteFileStub) *http.ServeMux {
	h := handler.NewDeleteFileHandler(stub)
	requireAuth := middleware.RequireAuth(administradorVerifier())
	mux := http.NewServeMux()
	mux.Handle("DELETE /api/v1/loteos/{loteoId}/archivos/{archivoId}", requireAuth(handler.Adapt(h, 5*time.Second)))
	return mux
}

func TestDeleteFileHandlerDeletesTheFile(t *testing.T) {
	stub := &deleteFileStub{}
	mux := deleteFileMux(stub)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/loteos/loteo-9/archivos/archivo-1", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if stub.gotLoteoID != "loteo-9" || stub.gotFileID != "archivo-1" {
		t.Fatalf("ids = (%q, %q), want (loteo-9, archivo-1)", stub.gotLoteoID, stub.gotFileID)
	}
}

func TestDeleteFileHandlerMapsANotFound(t *testing.T) {
	stub := &deleteFileStub{err: domain.ErrFileNotFound}
	mux := deleteFileMux(stub)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/loteos/loteo-9/archivos/archivo-1", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestDeleteFileHandlerRejectsRequestsWithoutAToken(t *testing.T) {
	stub := &deleteFileStub{}
	mux := deleteFileMux(stub)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/loteos/loteo-9/archivos/archivo-1", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
