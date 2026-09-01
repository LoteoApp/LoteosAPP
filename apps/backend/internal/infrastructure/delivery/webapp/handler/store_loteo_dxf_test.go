package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type storeLoteoDxfStub struct {
	file     domain.LoteoDxfFile
	err      error
	called   bool
	gotActor loteos.Actor
	gotInput loteos.StoreLoteoDxfInput
	gotBytes []byte
}

func (stub *storeLoteoDxfStub) Execute(
	_ context.Context,
	actor loteos.Actor,
	input loteos.StoreLoteoDxfInput,
) (domain.LoteoDxfFile, error) {
	stub.called = true
	stub.gotActor = actor
	stub.gotInput = input
	if input.Content != nil {
		stub.gotBytes, _ = io.ReadAll(input.Content)
	}
	return stub.file, stub.err
}

func storeLoteoDxfMux(stub *storeLoteoDxfStub, verifier userVerifierStub) *http.ServeMux {
	h := handler.NewStoreLoteoDxfHandler(stub)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("PUT /api/v1/loteos/{loteoId}/dxf", requireAuth(handler.Adapt(h, 5*time.Second)))
	return mux
}

func multipartDxfBody(t *testing.T, field, fileName string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if field != "" {
		part, err := writer.CreateFormFile(field, fileName)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("write part: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	return &body, writer.FormDataContentType()
}

func TestStoreLoteoDxfHandlerStoresTheFile(t *testing.T) {
	stub := &storeLoteoDxfStub{file: domain.LoteoDxfFile{
		ID: "archivo-1", StorageKey: "loteos/loteo-9/dxf/version.dxf",
		OriginalName: "plano.dxf", MimeType: "application/dxf", Sha256: "abc123",
	}}
	mux := storeLoteoDxfMux(stub, administradorVerifier())

	content := []byte("0\nSECTION\n2\nENTITIES\n0\nENDSEC\n0\nEOF\n")
	body, contentType := multipartDxfBody(t, "archivo", "plano.dxf", content)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/loteos/loteo-9/dxf", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (%s)", recorder.Code, http.StatusCreated, recorder.Body)
	}
	if !stub.called {
		t.Fatal("use case not called")
	}
	if stub.gotInput.LoteoID != "loteo-9" || stub.gotInput.FileName != "plano.dxf" {
		t.Fatalf("input = %+v, want loteo id loteo-9 and file plano.dxf", stub.gotInput)
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
	if _, exposed := payload["storageKey"]; exposed {
		t.Fatal("response exposes the internal object-storage key")
	}
	if payload["mimeType"] != "application/dxf" || payload["hashSha256"] != "abc123" {
		t.Fatalf("body = %#v, want normalized DXF metadata", payload)
	}
}

func TestStoreLoteoDxfHandlerRejectsMissingFilePart(t *testing.T) {
	stub := &storeLoteoDxfStub{}
	mux := storeLoteoDxfMux(stub, administradorVerifier())

	body, contentType := multipartDxfBody(t, "", "", nil)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/loteos/loteo-9/dxf", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if stub.called {
		t.Fatal("use case called for a request without a file part")
	}
}

func TestStoreLoteoDxfHandlerRejectsBodyThatIsNotMultipart(t *testing.T) {
	stub := &storeLoteoDxfStub{}
	mux := storeLoteoDxfMux(stub, administradorVerifier())

	request := httptest.NewRequest(http.MethodPut, "/api/v1/loteos/loteo-9/dxf", bytes.NewReader([]byte("not multipart")))
	request.Header.Set("Content-Type", "multipart/form-data")
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if stub.called {
		t.Fatal("use case called for a non-multipart body")
	}
}

func TestStoreLoteoDxfHandlerMapsUseCaseErrors(t *testing.T) {
	cases := map[string]struct {
		err  error
		want int
	}{
		"not found": {domain.ErrLoteoNotFound, http.StatusNotFound},
		"forbidden": {domain.ErrNoAutorizado, http.StatusForbidden},
		"invalid":   {domain.ErrInvalidDxfFile, http.StatusBadRequest},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			stub := &storeLoteoDxfStub{err: tc.err}
			mux := storeLoteoDxfMux(stub, administradorVerifier())

			body, contentType := multipartDxfBody(t, "archivo", "plano.dxf", []byte("dxf"))
			request := httptest.NewRequest(http.MethodPut, "/api/v1/loteos/loteo-9/dxf", body)
			request.Header.Set("Content-Type", contentType)
			request.Header.Set("Authorization", "Bearer token")
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)

			if recorder.Code != tc.want {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.want)
			}
		})
	}
}

func TestStoreLoteoDxfHandlerRejectsRequestsWithoutAToken(t *testing.T) {
	stub := &storeLoteoDxfStub{}
	mux := storeLoteoDxfMux(stub, administradorVerifier())

	body, contentType := multipartDxfBody(t, "archivo", "plano.dxf", []byte("dxf"))
	request := httptest.NewRequest(http.MethodPut, "/api/v1/loteos/loteo-9/dxf", body)
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if stub.called {
		t.Fatal("use case called for an unauthenticated request")
	}
}
