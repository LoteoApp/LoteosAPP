package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"loteosapp/backend/internal/business/domain"
)

func TestReservationErrorHelpers(t *testing.T) {
	t.Parallel()

	transition := &pgconn.PgError{ConstraintName: reservationTransitionConstraint}
	if !errors.Is(mapReservationWriteError(transition), domain.ErrReservationInvalidState) {
		t.Fatal("transition constraint should map to invalid state")
	}
	active := &pgconn.PgError{ConstraintName: reservationActiveIndex}
	if !errors.Is(mapReservationWriteError(active), domain.ErrReservationActiveConflict) {
		t.Fatal("active reservation constraint should map to conflict")
	}
	other := errors.New("other database error")
	if !errors.Is(mapReservationWriteError(other), other) {
		t.Fatal("unmapped database error should be preserved")
	}

	for _, code := range []string{"40001", "40P01", "55P03"} {
		if !isTransientDBError(&pgconn.PgError{Code: code}) {
			t.Errorf("code %s should be transient", code)
		}
	}
	if !isTransientDBError(context.Canceled) || !isTransientDBError(context.DeadlineExceeded) {
		t.Error("context cancellation should be transient")
	}
	if isTransientDBError(&pgconn.PgError{Code: "23505"}) || isTransientDBError(other) {
		t.Error("non-transient errors should not be retried")
	}
	if isInvalidUUID(other) {
		t.Error("ordinary errors are not invalid UUID errors")
	}
	if stringValue(nil) != "" {
		t.Error("nil string pointer should become empty")
	}
	clock := systemReservationClock{}
	if clock.Now().IsZero() {
		t.Error("system reservation clock should return the current time")
	}
}

func TestRepositoryErrorMappings(t *testing.T) {
	t.Parallel()
	nonConstraint := errors.New("database failure")
	if !errors.Is(mapAgencyWriteError(nonConstraint), nonConstraint) || !errors.Is(mapClienteWriteError(nonConstraint), nonConstraint) {
		t.Error("non-constraint errors should be preserved")
	}
	if !errors.Is(mapAgencyWriteError(&pgconn.PgError{Code: uniqueViolationCode, ConstraintName: cuitUniqueConstraint}), domain.ErrCUITInUse) {
		t.Error("agency CUIT conflicts should map to the domain error")
	}
	if !errors.Is(mapClienteWriteError(&pgconn.PgError{Code: uniqueViolationCode, ConstraintName: dniUniqueConstraint}), domain.ErrDNIEnUso) {
		t.Error("client DNI conflicts should map to the domain error")
	}
	if mapped := mapAgencyWriteError(&pgconn.PgError{Code: uniqueViolationCode, ConstraintName: "other"}); mapped == nil {
		t.Error("agency unique conflict should be mapped")
	}
	if mapped := mapClienteWriteError(&pgconn.PgError{Code: uniqueViolationCode, ConstraintName: "other"}); mapped == nil {
		t.Error("client unique conflict should be mapped")
	}

	if !errors.Is(mapLotStateWriteError(&pgconn.PgError{ConstraintName: lotStateTransitionConstraint}), domain.ErrInvalidLotStateTransition) {
		t.Error("lot transition conflicts should map")
	}
	if !errors.Is(mapLotStateWriteError(&pgconn.PgError{ConstraintName: lotReservationConstraint}), domain.ErrInvalidLotStateReference) {
		t.Error("lot reservation references should map")
	}
	if !errors.Is(mapLotStateWriteError(&pgconn.PgError{ConstraintName: lotSaleConstraint}), domain.ErrInvalidLotStateReference) {
		t.Error("lot sale references should map")
	}
	if !errors.Is(mapLotStateWriteError(nonConstraint), nonConstraint) {
		t.Error("unmapped lot state errors should be preserved")
	}

	if !errors.Is(mapManzanaUpdateError(&pgconn.PgError{Code: uniqueViolationCode}), domain.ErrManzanaNumberInUse) ||
		!errors.Is(mapManzanaUpdateError(&pgconn.PgError{Code: invalidTextRepresentationCode}), domain.ErrManzanaNotFound) ||
		!errors.Is(mapManzanaUpdateError(nonConstraint), nonConstraint) {
		t.Error("manzana errors should map by PostgreSQL code")
	}
	if !errors.Is(mapManzanaCalleError(&pgconn.PgError{Code: foreignKeyViolationCode}), domain.ErrUnknownCalle) ||
		!errors.Is(mapManzanaCalleError(&pgconn.PgError{Code: invalidTextRepresentationCode}), domain.ErrUnknownCalle) ||
		!errors.Is(mapManzanaCalleError(nonConstraint), nonConstraint) {
		t.Error("manzana calle errors should map by PostgreSQL code")
	}
	if !errors.Is(mapCalleUpdateError(&pgconn.PgError{Code: checkViolationCode}), domain.ErrInvalidCalleType) ||
		!errors.Is(mapCalleUpdateError(&pgconn.PgError{Code: invalidTextRepresentationCode}), domain.ErrCalleNotFound) ||
		!errors.Is(mapCalleUpdateError(nonConstraint), nonConstraint) {
		t.Error("calle errors should map by PostgreSQL code")
	}
}
