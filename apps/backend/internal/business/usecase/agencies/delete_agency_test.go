package agencies

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestDeleteAgencyRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	roles := []string{domain.RolAdministrativo, domain.RolInmobiliaria, domain.RolAgrimensor, domain.RolEscribano}

	for _, rol := range roles {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.InmobiliariaRepository{}
			users := &gatewayfake.UserRepository{}
			deleteAgency := NewDeleteAgency(repository, users)

			err := deleteAgency.Execute(context.Background(), DeleteAgencyInput{
				ActorRoles: []string{rol}, Subject: "sb-1", ID: "agency-1",
			})

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if repository.SoftDeleteCalls != 0 {
				t.Error("Execute() should not call repository when actor is not administrador")
			}
		})
	}
}

func TestDeleteAgencyHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	deleteAgency := NewDeleteAgency(repository, users)

	err := deleteAgency.Execute(context.Background(), DeleteAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.SoftDeleteCalls != 1 {
		t.Errorf("Execute() repository.SoftDelete calls = %d, want 1", repository.SoftDeleteCalls)
	}
	if repository.SoftDeletedID != "agency-1" {
		t.Errorf("Execute() soft deleted id = %q, want %q", repository.SoftDeletedID, "agency-1")
	}
	if repository.SoftDeletedActor != "user-1" {
		t.Errorf("Execute() soft deleted actor = %q, want %q", repository.SoftDeletedActor, "user-1")
	}
}

func TestDeleteAgencyPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrInmobiliariaNoEncontrada
	repository := &gatewayfake.InmobiliariaRepository{SoftDeleteErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	deleteAgency := NewDeleteAgency(repository, users)

	err := deleteAgency.Execute(context.Background(), DeleteAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1",
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestDeleteAgencyLeavesUnexpectedRepositoryErrorUnclassified(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("syntax error at end of input")
	repository := &gatewayfake.InmobiliariaRepository{SoftDeleteErr: rawErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	deleteAgency := NewDeleteAgency(repository, users)

	err := deleteAgency.Execute(context.Background(), DeleteAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1",
	})

	if !errors.Is(err, rawErr) {
		t.Fatalf("Execute() error = %v, want %v", err, rawErr)
	}
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		t.Errorf("Execute() error = %v, want it unclassified so it surfaces as a 500", err)
	}
}

func TestDeleteAgencyRejectsActorWithoutUsuario(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado}
	deleteAgency := NewDeleteAgency(repository, users)

	err := deleteAgency.Execute(context.Background(), DeleteAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1",
	})

	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}

func TestDeleteAgencyPropagatesActorResolutionError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection refused")
	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: rawErr}
	deleteAgency := NewDeleteAgency(repository, users)

	err := deleteAgency.Execute(context.Background(), DeleteAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1",
	})

	if !errors.Is(err, rawErr) {
		t.Fatalf("Execute() error = %v, want %v", err, rawErr)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}
