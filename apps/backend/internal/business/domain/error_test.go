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

func TestErrorIsMatchesOnCode(t *testing.T) {
	t.Parallel()

	sentinel := &domain.Error{Kind: domain.KindNotFound, Code: "object_not_found", Message: "no existe"}

	t.Run("matches another error with the same code", func(t *testing.T) {
		same := &domain.Error{Kind: domain.KindNotFound, Code: "object_not_found", Message: "otro mensaje"}

		if !errors.Is(same, sentinel) {
			t.Error("errors.Is() = false, want true for the same code")
		}
	})

	t.Run("does not match a different code", func(t *testing.T) {
		other := &domain.Error{Kind: domain.KindNotFound, Code: "user_not_found", Message: "no existe"}

		if errors.Is(other, sentinel) {
			t.Error("errors.Is() = true, want false for a different code")
		}
	})

	t.Run("does not match a plain error", func(t *testing.T) {
		if errors.Is(errors.New("object_not_found"), sentinel) {
			t.Error("errors.Is() = true, want false for a non-domain error")
		}
	})
}

func TestErrorWithCause(t *testing.T) {
	t.Parallel()

	sentinel := &domain.Error{Kind: domain.KindUnavailable, Code: "storage_unavailable", Message: "no disponible"}
	cause := errors.New("dial tcp: connection refused")

	withCause := sentinel.WithCause(cause)

	if !errors.Is(withCause, cause) {
		t.Error("errors.Is(withCause, cause) = false, want the cause reachable")
	}
	if !errors.Is(withCause, sentinel) {
		t.Error("errors.Is(withCause, sentinel) = false, want it to still match the sentinel")
	}
	if withCause.Kind != sentinel.Kind || withCause.Message != sentinel.Message {
		t.Errorf("WithCause() = %+v, want it to keep Kind and Message", withCause)
	}
	if sentinel.Cause != nil {
		t.Error("WithCause() mutated the sentinel, want the shared value left untouched")
	}
}
