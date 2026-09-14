package handler_test

import (
	"bytes"
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

type getFileContentStub struct {
	content    loteos.FileContent
	err        error
	gotLoteoID string
	gotFileID  string
}

func (stub *getFileContentStub) Execute(
	_ context.Context, _ loteos.Actor, loteoID, archivoID string,
) (loteos.FileContent, error) {
	stub.gotLoteoID = loteoID
	stub.gotFileID = archivoID
	return stub.content, stub.err
}

func getFileContentMux(stub *getFileContentStub) *http.ServeMux {
	h := handler.NewGetFileContentHandler(stub)
	requireAuth := middleware.RequireAuth(administradorVerifier())
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/loteos/{loteoId}/archivos/{archivoId}", requireAuth(handler.Adapt(h, 5*time.Second)))
	return mux
}

func TestGetFileContentHandlerStreamsTheBytes(t *testing.T) {
	stub := &getFileContentStub{content: loteos.FileContent{
		File: domain.File{ID: "archivo-1", MimeType: "image/jpeg", OriginalName: "foto.jpg"},
		Body: io.NopCloser(strings.NewReader("hola")),
	}}
	mux := getFileContentMux(stub)

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
	if stub.gotLoteoID != "loteo-9" || stub.gotFileID != "archivo-1" {
		t.Fatalf("ids = (%q, %q), want (loteo-9, archivo-1)", stub.gotLoteoID, stub.gotFileID)
	}
}

// partialThenBrokenStream returns some bytes on its first Read, then an
// error on every Read after that — simulating an R2/network failure partway
// through a download.
type partialThenBrokenStream struct {
	data []byte
	sent bool
}

func (stream *partialThenBrokenStream) Read(p []byte) (int, error) {
	if !stream.sent {
		stream.sent = true
		return copy(p, stream.data), nil
	}

	return 0, io.ErrUnexpectedEOF
}

func (stream *partialThenBrokenStream) Close() error { return nil }

func TestGetFileContentHandlerAbortsTheConnectionWhenTheStreamBreaksMidway(t *testing.T) {
	// Large enough that net/http flushes it to the client before the second
	// Read fails, the same way a real multi-megabyte foto/plano would: the
	// client gets to see the 200 and part of the body before the break.
	partial := bytes.Repeat([]byte("a"), 64*1024)
	stub := &getFileContentStub{content: loteos.FileContent{
		File: domain.File{ID: "archivo-1", MimeType: "image/jpeg", OriginalName: "foto.jpg"},
		Body: &partialThenBrokenStream{data: partial},
		Size: int64(len(partial)) * 10,
	}}
	mux := getFileContentMux(stub)
	server := httptest.NewServer(mux)
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/loteos/loteo-9/archivos/archivo-1", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	request.Header.Set("Authorization", "Bearer token")

	// The break can surface either as the request itself failing (the abort
	// beat the writer's first flush) or as a 200 whose body can't be read in
	// full (the 64KB chunk was already on the wire). Either way, it must
	// never look like a complete, successful download.
	response, doErr := server.Client().Do(request)
	if doErr != nil {
		return
	}
	defer response.Body.Close()

	if _, err := io.ReadAll(response.Body); err == nil {
		t.Fatal("ReadAll() error = nil, want an error for the truncated body")
	}
}

func TestGetFileContentHandlerMapsANotFound(t *testing.T) {
	stub := &getFileContentStub{err: domain.ErrFileNotFound}
	mux := getFileContentMux(stub)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/loteos/loteo-9/archivos/archivo-1", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
