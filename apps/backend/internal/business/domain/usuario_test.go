package domain

import (
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
