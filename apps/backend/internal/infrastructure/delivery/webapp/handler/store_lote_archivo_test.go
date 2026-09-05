package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type storeLoteArchivoStub struct {
	archivo  domain.Archivo
	err      error
	called   bool
	gotInput loteos.StoreLoteArchivoInput
	gotBytes []byte
}

func (stub *storeLoteArchivoStub) Execute(
	_ context.Context,
	_ loteos.Actor,
	input loteos.StoreLoteArchivoInput,
) (domain.Archivo, error) {
	stub.called = true
	stub.gotInput = input
	if input.Content != nil {
		stub.gotBytes, _ = io.ReadAll(input.Content)
	}
	return stub.archivo, stub.err
}

func storeLoteArchivoMux(stub *storeLoteArchivoStub, verifier userVerifierStub) *http.ServeMux {
	h := handler.NewStoreLoteArchivoHandler(stub)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/loteos/{loteoId}/lotes/{loteId}/archivos", requireAuth(handler.Adapt(h, 5*time.Second)))
	return mux
}

func TestStoreLoteArchivoHandlerStoresTheFile(t *testing.T) {
	stub := &storeLoteArchivoStub{archivo: domain.Archivo{ID: "archivo-1", Categoria: "plano"}}
	mux := storeLoteArchivoMux(stub, administradorVerifier())

	content := []byte("%PDF-1.4")
	body, contentType := multipartArchivoBody(t, "plano", "plano.pdf", "application/pdf", content)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/loteos/loteo-9/lotes/lote-3/archivos", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (%s)", recorder.Code, http.StatusCreated, recorder.Body)
	}
	if stub.gotInput.LoteoID != "loteo-9" || stub.gotInput.LoteID != "lote-3" {
		t.Fatalf("ids = (%q, %q), want (loteo-9, lote-3)", stub.gotInput.LoteoID, stub.gotInput.LoteID)
	}
	if !bytes.Equal(stub.gotBytes, content) {
		t.Fatalf("received bytes = %q, want %q", stub.gotBytes, content)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if payload["id"] != "archivo-1" {
		t.Fatalf("body id = %v, want archivo-1", payload["id"])
	}
}

func TestStoreLoteArchivoHandlerMapsALoteNotFound(t *testing.T) {
	stub := &storeLoteArchivoStub{err: domain.ErrLoteNotFound}
	mux := storeLoteArchivoMux(stub, administradorVerifier())

	body, contentType := multipartArchivoBody(t, "plano", "plano.pdf", "application/pdf", []byte("x"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/loteos/loteo-9/lotes/lote-3/archivos", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
