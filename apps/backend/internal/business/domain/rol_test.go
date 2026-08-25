package domain_test

import (
	"testing"

	"loteosapp/backend/internal/business/domain"
)

func TestRolValido(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rol  domain.Rol
		want bool
	}{
		{name: "administrador", rol: domain.RolAdministrador, want: true},
		{name: "administrativo", rol: domain.RolAdministrativo, want: true},
		{name: "agrimensor", rol: domain.RolAgrimensor, want: true},
		{name: "escribano", rol: domain.RolEscribano, want: true},
		{name: "inmobiliaria", rol: domain.RolInmobiliaria, want: true},
		{name: "desconocido", rol: domain.Rol("superadmin"), want: false},
		{name: "vacío", rol: domain.Rol(""), want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.rol.Valido(); got != test.want {
				t.Errorf("Rol(%q).Valido() = %v, want %v", test.rol, got, test.want)
			}
		})
	}
}
