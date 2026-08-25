package clients

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestListClientsHappyPath(t *testing.T) {
	t.Parallel()

	want := []domain.Cliente{{ID: "client-1", Nombre: "Ana", Apellido: "Perez", DNI: "30111222"}}
	repository := &gatewayfake.ClienteRepository{ListResult: want}
	listClients := NewListClients(repository)

	got, err := listClients.Execute(context.Background(), " perez ")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "client-1" {
		t.Errorf("Execute() = %#v", got)
	}
	if repository.ListSearch != "perez" {
		t.Errorf("Execute() search = %q, want %q", repository.ListSearch, "perez")
	}
}

func TestListClientsPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("connection refused")
	repository := &gatewayfake.ClienteRepository{ListErr: wantErr}
	listClients := NewListClients(repository)

	_, err := listClients.Execute(context.Background(), "")

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
