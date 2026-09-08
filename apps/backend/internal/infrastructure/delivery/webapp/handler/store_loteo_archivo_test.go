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

type storeLoteoFileStub struct {
	file     domain.File
	err      error
	called   bool
	gotInput loteos.StoreLoteoFileInput
	gotBytes []byte
}

func (stub *storeLoteoFileStub) Execute(
	_ context.Context,
	_ loteos.Actor,
	input loteos.StoreLoteoFileInput,
) (domain.File, error) {
	stub.called = true
	stub.gotInput = input
	if input.Content != nil {
		stub.gotBytes, _ = io.ReadAll(input.Content)
	}
	return stub.file, stub.err
}

func storeLoteoFileMux(stub *storeLoteoFileStub, verifier userVerifierStub) *http.ServeMux {
	h := handler.NewStoreLoteoFileHandler(stub)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/loteos/{loteoId}/archivos", requireAuth(handler.Adapt(h, 5*time.Second)))
	return mux
}

func multipartFileBody(t *testing.T, category, fileName, contentType string, content []byte) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if category != "" {
		if err := writer.WriteField("categoria", category); err != nil {
			t.Fatalf("write categoria field: %v", err)
		}
	}
	if fileName != "" {
		header := make(map[string][]string)
		header["Content-Disposition"] = []string{`form-data; name="archivo"; filename="` + fileName + `"`}
		if contentType != "" {
			header["Content-Type"] = []string{contentType}
		}
		part, err := writer.CreatePart(header)
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

func TestStoreLoteoFileHandlerStoresTheFile(t *testing.T) {
	stub := &storeLoteoFileStub{file: domain.File{
		ID: "file-1", Category: "foto", OriginalName: "foto.jpg", MimeType: "image/jpeg", Sha256: "abc123",
	}}
	mux := storeLoteoFileMux(stub, administradorVerifier())

	content := []byte("fake-jpeg-bytes")
	body, contentType := multipartFileBody(t, "foto", "foto.jpg", "image/jpeg", content)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/loteos/loteo-9/archivos", body)
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
	if stub.gotInput.LoteoID != "loteo-9" || stub.gotInput.Category != "foto" || stub.gotInput.MimeType != "image/jpeg" {
		t.Fatalf("input = %+v, want loteo-9/foto/image/jpeg", stub.gotInput)
	}
	if !bytes.Equal(stub.gotBytes, content) {
		t.Fatalf("received bytes = %q, want %q", stub.gotBytes, content)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if payload["id"] != "file-1" || payload["categoria"] != "foto" {
		t.Fatalf("body = %#v, want file-1/foto", payload)
	}
	if _, exposed := payload["storageKey"]; exposed {
		t.Fatal("response exposes the internal object-storage key")
	}
}

func TestStoreLoteoFileHandlerRejectsMissingFilePart(t *testing.T) {
	stub := &storeLoteoFileStub{}
	mux := storeLoteoFileMux(stub, administradorVerifier())

	body, contentType := multipartFileBody(t, "foto", "", "", nil)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/loteos/loteo-9/archivos", body)
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

func TestStoreLoteoFileHandlerMapsUseCaseErrors(t *testing.T) {
	cases := map[string]struct {
		err  error
		want int
	}{
		"not found": {domain.ErrLoteoNotFound, http.StatusNotFound},
		"forbidden": {domain.ErrNoAutorizado, http.StatusForbidden},
		"invalid":   {domain.ErrInvalidFile, http.StatusBadRequest},
		"too many":  {domain.ErrTooManyFiles, http.StatusBadRequest},
		"conflict":  {domain.ErrLoteNumberInUse, http.StatusConflict},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			stub := &storeLoteoFileStub{err: tc.err}
			mux := storeLoteoFileMux(stub, administradorVerifier())

			body, contentType := multipartFileBody(t, "foto", "foto.jpg", "image/jpeg", []byte("x"))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/loteos/loteo-9/archivos", body)
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

func TestStoreLoteoFileHandlerRejectsRequestsWithoutAToken(t *testing.T) {
	stub := &storeLoteoFileStub{}
	mux := storeLoteoFileMux(stub, administradorVerifier())

	body, contentType := multipartFileBody(t, "foto", "foto.jpg", "image/jpeg", []byte("x"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/loteos/loteo-9/archivos", body)
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
