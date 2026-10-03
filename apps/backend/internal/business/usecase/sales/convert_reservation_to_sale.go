package sales

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

const reservationToSaleHashVersion = "reservation-to-sale:v1"

type ConvertReservationToSaleInput struct {
	Actor          Actor
	ReservationID  string
	PaymentMethod  string
	IdempotencyKey string
	PaymentPlan    *PaymentPlanInput
}

// ConvertReservationToSale turns an active reserva into a venta for the same
// lote, cliente and vendedor. Its seller, an administrador or an
// administrativo may convert it; the request carries just the payment terms. A retry with the same
// Idempotency-Key and terms returns the venta already registered.
type ConvertReservationToSale interface {
	Execute(ctx context.Context, input ConvertReservationToSaleInput) (domain.Sale, error)
}

type convertReservationToSaleUseCase struct {
	repository gateway.SaleRepository
	users      gateway.UserRepository
}

func NewConvertReservationToSale(repository gateway.SaleRepository, users gateway.UserRepository) ConvertReservationToSale {
	return &convertReservationToSaleUseCase{repository: repository, users: users}
}

func (useCase *convertReservationToSaleUseCase) Execute(ctx context.Context, input ConvertReservationToSaleInput) (domain.Sale, error) {
	if !hasSaleRole(input.Actor.Roles) {
		return domain.Sale{}, domain.ErrNoAutorizado
	}
	key, err := domain.NormalizeIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		return domain.Sale{}, domain.ErrSaleIdempotencyRequired
	}
	reservationID, ok := canonicalUUID(input.ReservationID)
	if !ok {
		return domain.Sale{}, domain.ErrReservationNotFound
	}
	method, plan, err := normalizePaymentTerms(input.PaymentMethod, input.PaymentPlan)
	if err != nil {
		return domain.Sale{}, err
	}

	actor, err := resolveActor(ctx, useCase.users, input.Actor)
	if err != nil {
		return domain.Sale{}, fromRepository(err)
	}
	if !hasSaleRole([]string{string(actor.Rol)}) {
		return domain.Sale{}, domain.ErrNoAutorizado
	}

	sale, err := useCase.repository.ConvertReservation(ctx, gateway.ConvertReservationCommand{
		ReservationID:          reservationID,
		ActorID:                actor.ID,
		ActorAuthProviderID:    input.Actor.AuthProviderID,
		PaymentMethod:          method,
		PaymentPlan:            plan,
		IdempotencyKey:         key,
		IdempotencyPayloadHash: conversionPayloadHash(reservationID, method, plan),
	})
	if err != nil {
		return domain.Sale{}, fromRepository(err)
	}
	return sale, nil
}

// canonicalUUID accepts only the hyphenated hex form, lowercased: the id is
// hashed into the idempotency key and compared with what PostgreSQL returns,
// so other spellings of the same UUID must not count as another reserva.
func canonicalUUID(raw string) (string, bool) {
	id := strings.ToLower(strings.TrimSpace(raw))
	if len(id) != 36 {
		return "", false
	}
	for i, char := range id {
		switch i {
		case 8, 13, 18, 23:
			if char != '-' {
				return "", false
			}
		default:
			if !strings.ContainsRune("0123456789abcdef", char) {
				return "", false
			}
		}
	}
	return id, true
}

func conversionPayloadHash(reservationID string, method domain.PaymentMethod, plan *domain.PaymentPlanInput) string {
	payload := fmt.Sprintf("%s|%d:%s", reservationToSaleHashVersion, len(reservationID), reservationID) + paymentTermsPayload(method, plan)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}
