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

type listLoteoFilesStub struct {
	files      []domain.File
	err        error
	gotLoteoID string
}

func (stub *listLoteoFilesStub) Execute(_ context.Context, _ loteos.Actor, loteoID string) ([]domain.File, error) {
	stub.gotLoteoID = loteoID
	return stub.files, stub.err
}

func TestListLoteoFilesHandlerListsTheFiles(t *testing.T) {
	stub := &listLoteoFilesStub{files: []domain.File{{ID: "file-1", Category: "foto"}}}
	h := handler.NewListLoteoFilesHandler(stub)
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
		Files []map[string]any `json:"archivos"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(payload.Files) != 1 || payload.Files[0]["id"] != "file-1" {
		t.Fatalf("files = %#v, want one file-1", payload.Files)
	}
}

func TestListLoteoFilesHandlerMapsANotFound(t *testing.T) {
	stub := &listLoteoFilesStub{err: domain.ErrLoteoNotFound}
	h := handler.NewListLoteoFilesHandler(stub)
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
