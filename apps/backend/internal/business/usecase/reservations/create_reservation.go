package reservations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

type CreateReservationInput struct {
	Actor          Actor
	LoteoID        string
	LoteID         string
	ClienteID      string
	VendedorID     string
	IdempotencyKey string
}

type CreateReservation interface {
	Execute(ctx context.Context, input CreateReservationInput) (domain.Reservation, error)
}

type createReservationUseCase struct {
	repository gateway.ReservationRepository
	users      gateway.UserRepository
	clock      Clock
}

func NewCreateReservation(repository gateway.ReservationRepository, users gateway.UserRepository, clocks ...Clock) CreateReservation {
	clock := Clock(SystemClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &createReservationUseCase{repository: repository, users: users, clock: clock}
}

func (useCase *createReservationUseCase) Execute(ctx context.Context, input CreateReservationInput) (domain.Reservation, error) {
	if !hasReservationWriteRole(input.Actor.Roles) {
		return domain.Reservation{}, domain.ErrNoAutorizado
	}
	key, err := domain.NormalizeIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		return domain.Reservation{}, err
	}
	loteoID := strings.TrimSpace(input.LoteoID)
	loteID := strings.TrimSpace(input.LoteID)
	clienteID := strings.TrimSpace(input.ClienteID)
	if loteoID == "" || loteID == "" {
		return domain.Reservation{}, domain.ErrLoteNotFound
	}
	if clienteID == "" {
		return domain.Reservation{}, domain.ErrReservationInvalidClient
	}

	actor, err := resolveActor(ctx, useCase.users, input.Actor)
	if err != nil {
		return domain.Reservation{}, fromRepository(err)
	}
	if !hasReservationWriteRole([]string{string(actor.Rol)}) {
		return domain.Reservation{}, domain.ErrNoAutorizado
	}

	sellerID := strings.TrimSpace(input.VendedorID)
	sellerIsActor := false
	if actor.Rol == domain.RolInmobiliaria {
		sellerID = actor.ID
		sellerIsActor = true
	} else if sellerID == "" {
		return domain.Reservation{}, domain.ErrReservationSellerRequired
	}

	now := useCase.clock.Now().UTC()
	reservation, err := useCase.repository.Create(ctx, gateway.CreateReservationCommand{
		LoteoID:                loteoID,
		LoteID:                 loteID,
		ClienteID:              clienteID,
		VendedorID:             sellerID,
		ActorID:                actor.ID,
		ActorAuthProviderID:    input.Actor.AuthProviderID,
		SellerIsActor:          sellerIsActor,
		IdempotencyKey:         key,
		IdempotencyPayloadHash: reservationPayloadHash(loteoID, loteID, clienteID, sellerID),
		CreatedAt:              now,
	})
	if err != nil {
		return domain.Reservation{}, fromRepository(err)
	}
	return reservation, nil
}

func reservationPayloadHash(loteoID, loteID, clienteID, sellerID string) string {
	payload := fmt.Sprintf("%d:%s%d:%s%d:%s%d:%s", len(loteoID), loteoID, len(loteID), loteID, len(clienteID), clienteID, len(sellerID), sellerID)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}
