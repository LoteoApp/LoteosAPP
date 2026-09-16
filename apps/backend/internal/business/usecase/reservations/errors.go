package reservations

import (
	"errors"

	"loteosapp/backend/internal/business/domain"
)

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
