package clients

import (
	"errors"

	"loteosapp/backend/internal/business/domain"
)

// wrapGatewayError classifies an error coming back from a gateway (the
// cliente repository or the user repository). A known *domain.Error (e.g.
// domain.ErrDNIEnUso, domain.ErrClienteNoEncontrado) is returned unchanged
// so its Kind and Code reach the HTTP layer as intended. Anything else — a
// raw driver, network or other unexpected failure — is wrapped as
// KindUnavailable so it maps to a 503 instead of leaking through as an
// unclassified 500, while Cause preserves the original error for logging
// (see response.WriteError).
func wrapGatewayError(err error) error {
	if err == nil {
		return nil
	}

	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		return err
	}

	return &domain.Error{
		Kind:    domain.KindUnavailable,
		Code:    "client_gateway_unavailable",
		Message: "No se pudo completar la operación, intentá nuevamente",
		Cause:   err,
	}
}
