package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

func TestWriteErrorMapsDomainErrorsByKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid", err: &domain.Error{Kind: domain.KindInvalid, Code: "invalid_x", Message: "inválido"}, wantStatus: http.StatusBadRequest, wantCode: "invalid_x"},
		{name: "forbidden", err: &domain.Error{Kind: domain.KindForbidden, Code: "forbidden", Message: "no autorizado"}, wantStatus: http.StatusForbidden, wantCode: "forbidden"},
		{name: "conflict", err: &domain.Error{Kind: domain.KindConflict, Code: "in_use", Message: "ya existe"}, wantStatus: http.StatusConflict, wantCode: "in_use"},
		{name: "not found", err: &domain.Error{Kind: domain.KindNotFound, Code: "not_found", Message: "no existe"}, wantStatus: http.StatusNotFound, wantCode: "not_found"},
		{name: "known domain sentinel", err: domain.ErrEmailEnUso, wantStatus: http.StatusConflict, wantCode: "email_in_use"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)

			response.WriteError(recorder, request, "unused", test.err)

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}

			var got response.ErrorResponse
			if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if got.Code != test.wantCode {
				t.Errorf("code = %q, want %q", got.Code, test.wantCode)
			}
		})
	}
}

func TestWriteErrorHidesUnclassifiedErrors(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	response.WriteError(recorder, request, "unexpected failure", errors.New("connection string leaked"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	var got response.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != "internal_error" {
		t.Errorf("code = %q, want %q", got.Code, "internal_error")
	}
	if got.Message == "connection string leaked" {
		t.Error("response exposes the internal error")
	}
}

func TestWriteBadRequest(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	response.WriteBadRequest(recorder, "invalid_body", "Cuerpo de la solicitud inválido")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	var got response.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != "invalid_body" {
		t.Errorf("code = %q, want %q", got.Code, "invalid_body")
	}
}
