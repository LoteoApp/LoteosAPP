package users

import (
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
)

// assertDatabaseUnavailable checks that err is what fromRepository turns an
// unmapped repository/identity failure into: KindUnavailable, carrying cause
// unchanged so it can still be logged.
func assertDatabaseUnavailable(t *testing.T, err error, cause error) {
	t.Helper()

	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("Execute() error = %v (%T), want a *domain.Error", err, err)
	}
	if domainErr.Kind != domain.KindUnavailable {
		t.Errorf("Execute() error kind = %v, want %v", domainErr.Kind, domain.KindUnavailable)
	}
	if domainErr.Cause != cause {
		t.Errorf("Execute() error cause = %v, want %v", domainErr.Cause, cause)
	}
}
