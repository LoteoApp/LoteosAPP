package domain

import (
	"strings"
	"testing"
	"time"
)

func TestUsuarioActivo(t *testing.T) {
	t.Parallel()

	baja := time.Now()
	tests := []struct {
		name    string
		usuario Usuario
		want    bool
	}{
		{name: "sin fecha de baja", usuario: Usuario{}, want: true},
		{name: "con fecha de baja", usuario: Usuario{FechaBaja: &baja}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.usuario.Activo(); got != test.want {
				t.Errorf("Activo() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestEmailValido(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{name: "empty", email: "", want: false},
		{name: "no at sign", email: "ana.example.com", want: false},
		{name: "valid", email: "ana@example.com", want: true},
		{name: "at max length", email: strings.Repeat("a", maxEmailLength-len("@example.com")) + "@example.com", want: true},
		{name: "over max length", email: strings.Repeat("a", maxEmailLength-len("@example.com")+1) + "@example.com", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := EmailValido(test.email); got != test.want {
				t.Errorf("EmailValido(%q) = %v, want %v", test.email, got, test.want)
			}
		})
	}
}
