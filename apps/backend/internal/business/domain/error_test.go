package domain_test

import (
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
)

func TestErrorMessage(t *testing.T) {
	t.Parallel()

	err := &domain.Error{Kind: domain.KindInvalid, Code: "invalid_x", Message: "inválido"}

	if err.Error() != "inválido" {
		t.Errorf("Error() = %q, want %q", err.Error(), "inválido")
	}
}

func TestErrorUnwrapReturnsCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection refused")
	err := &domain.Error{Kind: domain.KindUnavailable, Code: "database_unavailable", Message: "no disponible", Cause: cause}

	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false, want true")
	}
}

func TestErrorUnwrapWithoutCauseReturnsNil(t *testing.T) {
	t.Parallel()

	err := &domain.Error{Kind: domain.KindInvalid, Code: "invalid_x", Message: "inválido"}

	if err.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", err.Unwrap())
	}
}
