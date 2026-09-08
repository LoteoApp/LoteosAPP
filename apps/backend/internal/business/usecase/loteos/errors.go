package loteos

import (
	"errors"

	"loteosapp/backend/internal/business/domain"
)

// fromRepository turns whatever the repository returned into the error a
// caller of this package can act on: a *domain.Error travels unchanged, and
// anything else (a dropped connection, a constraint nobody mapped) becomes an
// unavailable-kind error carrying the original as Cause.
func fromRepository(err error) error {
	if err == nil {
		return nil
	}

	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		return err
	}

	return domain.ErrDatabaseUnavailable.WithCause(err)
}

func fromStorage(err error) error {
	if err == nil {
		return nil
	}

	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		return err
	}

	return domain.ErrStorageUnavailable.WithCause(err)
}
