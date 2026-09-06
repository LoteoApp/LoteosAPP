package loteos

import (
	"context"
	"errors"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type TransitionLotState interface {
	Execute(ctx context.Context, actor Actor, command domain.LotStateTransition) (domain.LotStateEvent, error)
}

type transitionLotStateUseCase struct {
	states gateway.LotStateRepository
	users  gateway.UserRepository
}

func NewTransitionLotState(states gateway.LotStateRepository, users gateway.UserRepository) TransitionLotState {
	return &transitionLotStateUseCase{states: states, users: users}
}

func (useCase *transitionLotStateUseCase) Execute(
	ctx context.Context,
	actor Actor,
	command domain.LotStateTransition,
) (domain.LotStateEvent, error) {
	scope, err := lotStateScope(actor)
	if err != nil {
		return domain.LotStateEvent{}, err
	}

	command.DevelopmentID = strings.TrimSpace(command.DevelopmentID)
	command.LotID = strings.TrimSpace(command.LotID)
	command.Reason = strings.TrimSpace(command.Reason)
	command.ReservationID = trimReference(command.ReservationID)
	command.SaleID = trimReference(command.SaleID)
	if command.DevelopmentID == "" || command.LotID == "" {
		return domain.LotStateEvent{}, domain.ErrLoteNotFound
	}
	if command.Origin == domain.LotStateOriginSystem || command.Origin == domain.LotStateOriginCorrection {
		return domain.LotStateEvent{}, domain.ErrInvalidLotStateOrigin
	}
	if err := command.Validate(); err != nil {
		return domain.LotStateEvent{}, err
	}

	user, err := useCase.users.FindByAuthProviderID(ctx, actor.AuthProviderID)
	if err != nil {
		if errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			return domain.LotStateEvent{}, domain.ErrActorNoAprovisionado
		}
		return domain.LotStateEvent{}, fromRepository(err)
	}
	command.ActorID = user.ID

	exists, err := useCase.states.LotExists(ctx, command.DevelopmentID, command.LotID, scope)
	if err != nil {
		return domain.LotStateEvent{}, fromRepository(err)
	}
	if !exists {
		return domain.LotStateEvent{}, domain.ErrLoteNotFound
	}

	event, err := useCase.states.Transition(ctx, command)
	if err != nil {
		return domain.LotStateEvent{}, fromRepository(err)
	}

	return event, nil
}

func lotStateScope(actor Actor) (gateway.LoteoScope, error) {
	if domain.HasRole(actor.Roles, domain.RolAdministrador) ||
		domain.HasRole(actor.Roles, domain.RolAdministrativo) {
		return gateway.LoteoScope{}, nil
	}
	if !domain.HasRole(actor.Roles, domain.RolInmobiliaria) {
		return gateway.LoteoScope{}, domain.ErrNoAutorizado
	}

	authProviderID := actor.AuthProviderID
	return gateway.LoteoScope{
		AssigneeAuthProviderID: &authProviderID,
		ByAgencyAssignment:     true,
	}, nil
}

func trimReference(reference *string) *string {
	if reference == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*reference)
	return &trimmed
}
