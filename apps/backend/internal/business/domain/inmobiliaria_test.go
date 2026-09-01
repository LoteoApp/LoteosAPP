package domain_test

import (
	"testing"

	"loteosapp/backend/internal/business/domain"
)

func TestNormalizarCUIT(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cuit string
		want string
	}{
		{name: "hyphenated", cuit: "30-71234567-8", want: "30712345678"},
		{name: "spaced", cuit: "30 71234567 8", want: "30712345678"},
		{name: "dotted", cuit: "30.71234567.8", want: "30712345678"},
		{name: "already normalized", cuit: "30712345678", want: "30712345678"},
		{name: "keeps non separator characters", cuit: "30x", want: "30x"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := domain.NormalizarCUIT(test.cuit); got != test.want {
				t.Errorf("NormalizarCUIT(%q) = %q, want %q", test.cuit, got, test.want)
			}
		})
	}
}

func TestCUITValido(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cuit string
		want bool
	}{
		{name: "eleven digits", cuit: "30712345678", want: true},
		{name: "too short", cuit: "3071234567", want: false},
		{name: "too long", cuit: "307123456789", want: false},
		{name: "not digits", cuit: "3071234567x", want: false},
		{name: "empty", cuit: "", want: false},
		{name: "still separated", cuit: "30-71234567-8", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := domain.CUITValido(test.cuit); got != test.want {
				t.Errorf("CUITValido(%q) = %v, want %v", test.cuit, got, test.want)
			}
		})
	}
}
