package handler_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type getArchivoContentStub struct {
	content      loteos.ArchivoContent
	err          error
	gotLoteoID   string
	gotArchivoID string
}

func (stub *getArchivoContentStub) Execute(
	_ context.Context, _ loteos.Actor, loteoID, archivoID string,
) (loteos.ArchivoContent, error) {
	stub.gotLoteoID = loteoID
	stub.gotArchivoID = archivoID
	return stub.content, stub.err
}

func getArchivoContentMux(stub *getArchivoContentStub) *http.ServeMux {
	h := handler.NewGetArchivoContentHandler(stub)
	requireAuth := middleware.RequireAuth(administradorVerifier())
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/loteos/{loteoId}/archivos/{archivoId}", requireAuth(handler.Adapt(h, 5*time.Second)))
	return mux
}

func TestGetArchivoContentHandlerStreamsTheBytes(t *testing.T) {
	stub := &getArchivoContentStub{content: loteos.ArchivoContent{
		Archivo: domain.Archivo{ID: "archivo-1", MimeType: "image/jpeg", OriginalName: "foto.jpg"},
		Body:    io.NopCloser(strings.NewReader("hola")),
	}}
	mux := getArchivoContentMux(stub)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/loteos/loteo-9/archivos/archivo-1", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "hola" {
		t.Fatalf("body = %q, want hola", recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("Content-Type = %q, want image/jpeg", recorder.Header().Get("Content-Type"))
	}
	if stub.gotLoteoID != "loteo-9" || stub.gotArchivoID != "archivo-1" {
		t.Fatalf("ids = (%q, %q), want (loteo-9, archivo-1)", stub.gotLoteoID, stub.gotArchivoID)
	}
}

func TestGetArchivoContentHandlerMapsANotFound(t *testing.T) {
	stub := &getArchivoContentStub{err: domain.ErrArchivoNotFound}
	mux := getArchivoContentMux(stub)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/loteos/loteo-9/archivos/archivo-1", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
