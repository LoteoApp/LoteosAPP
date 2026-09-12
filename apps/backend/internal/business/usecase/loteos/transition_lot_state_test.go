package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func reservationTransition() domain.LotStateTransition {
	reservationID := " reservation-1 "
	return domain.LotStateTransition{
		DevelopmentID: " loteo-1 ",
		LotID:         " lote-1 ",
		ExpectedState: domain.LotStateAvailable,
		NextState:     domain.LotStateReserved,
		Origin:        domain.LotStateOriginReservation,
		Reason:        " initial hold ",
		ReservationID: &reservationID,
	}
}

func provisionedUserRepository() *gatewayfake.UserRepository {
	return &gatewayfake.UserRepository{
		FindByAuthProviderIDResult: domain.Usuario{ID: "user-1", AuthProviderID: "actor-1"},
	}
}

func TestTransitionLotStateExecutesAnAuthorizedCompareAndSet(t *testing.T) {
	t.Parallel()

	states := &gatewayfake.LotStateRepository{
		LotExistsResult:  true,
		TransitionResult: domain.LotStateEvent{ID: "event-1", State: domain.LotStateReserved},
	}
	useCase := loteos.NewTransitionLotState(states, provisionedUserRepository())

	event, err := useCase.Execute(context.Background(), administrador(), reservationTransition())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if event.ID != "event-1" || event.State != domain.LotStateReserved {
		t.Errorf("Execute() = %#v", event)
	}
	if states.ReceivedDevelopmentID != "loteo-1" || states.ReceivedLotID != "lote-1" {
		t.Errorf("ids = %q/%q, want trimmed values", states.ReceivedDevelopmentID, states.ReceivedLotID)
	}
	if states.ReceivedCommand.ActorID != "user-1" || states.ReceivedCommand.Reason != "initial hold" {
		t.Errorf("command = %#v, want resolved actor and trimmed reason", states.ReceivedCommand)
	}
	if states.ReceivedCommand.ReservationID == nil || *states.ReceivedCommand.ReservationID != "reservation-1" {
		t.Errorf("reservation id = %v, want trimmed value", states.ReceivedCommand.ReservationID)
	}
	if states.ReceivedLotScope.AssigneeAuthProviderID != nil {
		t.Error("an administrador should use an unrestricted scope")
	}
}

func TestTransitionLotStateScopesAnAgencyUser(t *testing.T) {
	t.Parallel()

	states := &gatewayfake.LotStateRepository{LotExistsResult: true}
	useCase := loteos.NewTransitionLotState(states, provisionedUserRepository())
	actor := loteos.Actor{AuthProviderID: "actor-1", Roles: []string{domain.RolInmobiliaria}}

	if _, err := useCase.Execute(context.Background(), actor, reservationTransition()); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	scope := states.ReceivedLotScope
	if scope.AssigneeAuthProviderID == nil || *scope.AssigneeAuthProviderID != "actor-1" || !scope.ByAgencyAssignment || scope.ByUserAssignment {
		t.Errorf("scope = %#v, want the actor's agency assignments", scope)
	}
}

func TestTransitionLotStateAllowsAnAdministrativoWithoutAssignment(t *testing.T) {
	t.Parallel()

	states := &gatewayfake.LotStateRepository{LotExistsResult: true}
	useCase := loteos.NewTransitionLotState(states, provisionedUserRepository())
	actor := loteos.Actor{AuthProviderID: "actor-1", Roles: []string{domain.RolAdministrativo}}

	if _, err := useCase.Execute(context.Background(), actor, reservationTransition()); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if states.ReceivedLotScope.AssigneeAuthProviderID != nil {
		t.Error("an administrativo should use an unrestricted scope")
	}
}

func TestTransitionLotStateRejectsUnauthorizedRolesBeforeDependencies(t *testing.T) {
	t.Parallel()

	states := &gatewayfake.LotStateRepository{}
	users := provisionedUserRepository()
	useCase := loteos.NewTransitionLotState(states, users)
	actor := loteos.Actor{AuthProviderID: "actor-1", Roles: []string{domain.RolAgrimensor}}

	_, err := useCase.Execute(context.Background(), actor, reservationTransition())
	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if users.FindByAuthProviderIDCalls != 0 || states.LotExistsCalls != 0 {
		t.Error("Execute() should not reach dependencies for an unauthorized role")
	}
}

func TestTransitionLotStateValidatesTheCommandBeforeDependencies(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mutate func(*domain.LotStateTransition)
		want   error
	}{
		"empty loteo": {
			mutate: func(command *domain.LotStateTransition) { command.DevelopmentID = " " },
			want:   domain.ErrLoteNotFound,
		},
		"empty lote": {
			mutate: func(command *domain.LotStateTransition) { command.LotID = " " },
			want:   domain.ErrLoteNotFound,
		},
		"invalid transition": {
			mutate: func(command *domain.LotStateTransition) { command.NextState = domain.LotStateCompleted },
			want:   domain.ErrInvalidLotStateTransition,
		},
		"system origin": {
			mutate: func(command *domain.LotStateTransition) { command.Origin = domain.LotStateOriginSystem },
			want:   domain.ErrInvalidLotStateOrigin,
		},
		"correction origin": {
			mutate: func(command *domain.LotStateTransition) { command.Origin = domain.LotStateOriginCorrection },
			want:   domain.ErrInvalidLotStateOrigin,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			states := &gatewayfake.LotStateRepository{}
			users := provisionedUserRepository()
			useCase := loteos.NewTransitionLotState(states, users)
			command := reservationTransition()
			test.mutate(&command)

			_, err := useCase.Execute(context.Background(), administrador(), command)
			if !errors.Is(err, test.want) {
				t.Fatalf("Execute() error = %v, want %v", err, test.want)
			}
			if users.FindByAuthProviderIDCalls != 0 || states.LotExistsCalls != 0 {
				t.Error("Execute() should not reach dependencies with an invalid command")
			}
		})
	}
}

func TestTransitionLotStateRequiresAProvisionedActor(t *testing.T) {
	t.Parallel()

	states := &gatewayfake.LotStateRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado}
	useCase := loteos.NewTransitionLotState(states, users)

	_, err := useCase.Execute(context.Background(), administrador(), reservationTransition())
	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
	if states.LotExistsCalls != 0 {
		t.Error("Execute() should stop before checking the lote")
	}
}

func TestTransitionLotStateReturnsNotFoundOutsideTheActorsScope(t *testing.T) {
	t.Parallel()

	states := &gatewayfake.LotStateRepository{LotExistsResult: false}
	useCase := loteos.NewTransitionLotState(states, provisionedUserRepository())

	_, err := useCase.Execute(context.Background(), administrador(), reservationTransition())
	if !errors.Is(err, domain.ErrLoteNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteNotFound)
	}
	if states.TransitionCalls != 0 {
		t.Error("Execute() should not transition a missing lote")
	}
}

func TestTransitionLotStateMapsDependencyFailures(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection reset")
	tests := map[string]struct {
		states *gatewayfake.LotStateRepository
		users  *gatewayfake.UserRepository
	}{
		"actor lookup": {
			states: &gatewayfake.LotStateRepository{},
			users:  &gatewayfake.UserRepository{FindByAuthProviderIDErr: cause},
		},
		"lote lookup": {
			states: &gatewayfake.LotStateRepository{LotExistsErr: cause},
			users:  provisionedUserRepository(),
		},
		"transition": {
			states: &gatewayfake.LotStateRepository{LotExistsResult: true, TransitionErr: cause},
			users:  provisionedUserRepository(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			useCase := loteos.NewTransitionLotState(test.states, test.users)

			_, err := useCase.Execute(context.Background(), administrador(), reservationTransition())
			assertUnavailable(t, err, cause)
		})
	}
}

func TestTransitionLotStateKeepsRepositoryBusinessErrors(t *testing.T) {
	t.Parallel()

	states := &gatewayfake.LotStateRepository{
		LotExistsResult: true,
		TransitionErr:   domain.ErrLotStateConflict,
	}
	useCase := loteos.NewTransitionLotState(states, provisionedUserRepository())

	_, err := useCase.Execute(context.Background(), administrador(), reservationTransition())
	if !errors.Is(err, domain.ErrLotStateConflict) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLotStateConflict)
	}
}
