package surveyors

import (
	"errors"

	"loteosapp/backend/internal/business/domain"
)

// fromRepository turns whatever a gateway returned into the error a caller of
// this package can act on: a *domain.Error travels unchanged, and anything
// else (a dropped connection, a constraint nobody mapped) becomes an
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

// asSurveyorNotFound reports a missing user as a missing agrimensor, so a
// caller of the agrimensores routes never learns whether the id it sent
// belongs to a user of another rol.
func asSurveyorNotFound(err error) error {
	if errors.Is(err, domain.ErrUsuarioNoEncontrado) {
		return domain.ErrAgrimensorNoEncontrado
	}

	return fromRepository(err)
}
