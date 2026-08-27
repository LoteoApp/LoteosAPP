package clients

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestListClientsRejectsUnauthorizedRole(t *testing.T) {
	t.Parallel()

	tests := []string{domain.RolAgrimensor, domain.RolEscribano}

	for _, rol := range tests {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.ClienteRepository{}
			listClients := NewListClients(repository)

			_, err := listClients.Execute(context.Background(), []string{rol}, "perez")

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if repository.ListCalls != 0 {
				t.Error("Execute() should not call repository when actor is not authorized")
			}
		})
	}
}

func TestListClientsHappyPath(t *testing.T) {
	t.Parallel()

	tests := []string{domain.RolAdministrador, domain.RolAdministrativo, domain.RolInmobiliaria}

	for _, rol := range tests {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			want := []domain.Cliente{{ID: "client-1", Nombre: "Ana", Apellido: "Perez", DNI: "30111222"}}
			repository := &gatewayfake.ClienteRepository{ListResult: want}
			listClients := NewListClients(repository)

			got, err := listClients.Execute(context.Background(), []string{rol}, " perez ")
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(got) != 1 || got[0].ID != "client-1" {
				t.Errorf("Execute() = %#v", got)
			}
			if repository.ListSearch != "perez" {
				t.Errorf("Execute() search = %q, want %q", repository.ListSearch, "perez")
			}
		})
	}
}

func TestListClientsPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrClienteNoEncontrado
	repository := &gatewayfake.ClienteRepository{ListErr: wantErr}
	listClients := NewListClients(repository)

	_, err := listClients.Execute(context.Background(), []string{domain.RolAdministrador}, "")

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestListClientsWrapsUnexpectedRepositoryError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection refused")
	repository := &gatewayfake.ClienteRepository{ListErr: rawErr}
	listClients := NewListClients(repository)

	_, err := listClients.Execute(context.Background(), []string{domain.RolAdministrador}, "")

	if !errors.Is(err, rawErr) {
		t.Fatalf("Execute() error = %v, want it to wrap %v", err, rawErr)
	}
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("Execute() error = %v, want a *domain.Error", err)
	}
	if domainErr.Kind != domain.KindUnavailable {
		t.Errorf("Execute() error kind = %q, want %q", domainErr.Kind, domain.KindUnavailable)
	}
}
