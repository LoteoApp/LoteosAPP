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

func TestUsuarioEsAgrimensor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		usuario Usuario
		want    bool
	}{
		{name: "agrimensor", usuario: Usuario{Rol: RolAgrimensor}, want: true},
		{name: "administrador", usuario: Usuario{Rol: RolAdministrador}, want: false},
		{name: "sin rol", usuario: Usuario{}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.usuario.EsAgrimensor(); got != test.want {
				t.Errorf("EsAgrimensor() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestPerfilEstaCompleto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nombre   string
		apellido string
		want     bool
	}{
		{name: "ambos presentes", nombre: "Ana", apellido: "Gómez", want: true},
		{name: "sin apellido", nombre: "Ana", want: false},
		{name: "sin nombre", apellido: "Gómez", want: false},
		{name: "vacío", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := PerfilEstaCompleto(test.nombre, test.apellido); got != test.want {
				t.Errorf("PerfilEstaCompleto(%q, %q) = %v, want %v", test.nombre, test.apellido, got, test.want)
			}
		})
	}
}
