package domain

import (
	"sort"
	"strings"
)

type ChargeType string

const (
	ChargeTypeMunicipalTax   ChargeType = "impuesto_municipal"
	ChargeTypeProvincialTax  ChargeType = "impuesto_provincial"
	ChargeTypeAdministrative ChargeType = "gasto_administrativo"
	ChargeTypeFees           ChargeType = "honorarios"
	ChargeTypeServices       ChargeType = "servicios"
	ChargeTypeAgency         ChargeType = "cargo_inmobiliaria"
	ChargeTypeOther          ChargeType = "otros"
)

const (
	MaxPaymentCharges      = 20
	MaxCurrencyCodeLength  = 10
	MaxChargeDetailLength  = 200
	chargeAmountMin        = 0.01
	maxChargeAmountDecimal = moneyDecimals
)

var (
	ErrChargeInvalidType     = &Error{Kind: KindInvalid, Code: "invalid_charge_type", Message: "El tipo de cargo adicional no es válido"}
	ErrChargeInvalidAmount   = &Error{Kind: KindInvalid, Code: "invalid_charge_amount", Message: "El monto del cargo tiene que ser mayor a cero, con hasta 2 decimales"}
	ErrChargeInvalidCurrency = &Error{
		Kind: KindInvalid, Code: "invalid_charge_currency", Message: "La moneda del cargo no es válida",
	}
	ErrChargeDetailTooLong = &Error{Kind: KindInvalid, Code: "charge_detail_too_long", Message: "El detalle del cargo no puede superar los 200 caracteres"}
	ErrTooManyCharges      = &Error{Kind: KindInvalid, Code: "too_many_charges", Message: "No se pueden cargar más de 20 cargos adicionales en un cobro"}
)

func (charge ChargeType) IsValid() bool {
	switch charge {
	case ChargeTypeMunicipalTax, ChargeTypeProvincialTax, ChargeTypeAdministrative,
		ChargeTypeFees, ChargeTypeServices, ChargeTypeAgency, ChargeTypeOther:
		return true
	default:
		return false
	}
}

// PaymentChargeInput is a cargo adicional as the caller describes it. Moneda
// is its own: a cuota in USD may be collected together with services in ARS,
// so a charge is never converted, only listed under its currency. An empty
// Moneda means the sale's currency.
type PaymentChargeInput struct {
	Tipo    ChargeType
	Monto   float64
	Moneda  string
	Detalle string
}

// PaymentCharge is a cargo adicional as the API publishes it.
type PaymentCharge struct {
	ID      string     `json:"id"`
	Tipo    ChargeType `json:"tipo"`
	Monto   float64    `json:"monto"`
	Moneda  string     `json:"moneda"`
	Detalle string     `json:"detalle,omitempty"`
}

// CurrencyTotal is what a cobro adds up to in one currency. A cobro that
// mixes a cuota in USD with services in ARS has one of these per currency,
// because the two amounts are never added together.
type CurrencyTotal struct {
	Moneda string  `json:"moneda"`
	Monto  float64 `json:"monto"`
}

// NormalizeCurrency trims and upper-cases a currency code, falling back to
// fallback when blank. Currency is free text in the database (lotes.moneda
// has no CHECK), so this only enforces shape, not an ISO 4217 list.
func NormalizeCurrency(currency, fallback string) string {
	normalized := strings.ToUpper(strings.TrimSpace(currency))
	if normalized == "" {
		return strings.ToUpper(strings.TrimSpace(fallback))
	}
	return normalized
}

// NormalizePaymentCharges validates the charges of a cobro and returns them
// normalized: currency upper-cased and defaulted to fallbackCurrency, detail
// trimmed. A blank fallback is allowed here because the repository, which is
// the first place that knows the sale's currency, normalizes again.
func NormalizePaymentCharges(charges []PaymentChargeInput, fallbackCurrency string) ([]PaymentChargeInput, error) {
	if len(charges) > MaxPaymentCharges {
		return nil, ErrTooManyCharges
	}
	normalized := make([]PaymentChargeInput, 0, len(charges))
	for _, charge := range charges {
		if !charge.Tipo.IsValid() {
			return nil, ErrChargeInvalidType
		}
		if !isFinite(charge.Monto) || charge.Monto < chargeAmountMin || !hasAtMostDecimals(charge.Monto, maxChargeAmountDecimal) {
			return nil, ErrChargeInvalidAmount
		}
		currency := NormalizeCurrency(charge.Moneda, fallbackCurrency)
		if len([]rune(currency)) > MaxCurrencyCodeLength {
			return nil, ErrChargeInvalidCurrency
		}
		detail := strings.TrimSpace(charge.Detalle)
		if len([]rune(detail)) > MaxChargeDetailLength {
			return nil, ErrChargeDetailTooLong
		}
		normalized = append(normalized, PaymentChargeInput{
			Tipo:    charge.Tipo,
			Monto:   RoundMoney(charge.Monto),
			Moneda:  currency,
			Detalle: detail,
		})
	}
	return normalized, nil
}

// PaymentTotals is what a cobro adds up to, per currency: the amount applied
// to the plan (entrega and cuotas, always in the sale's currency) plus the
// charges of that currency. The sale's currency comes first and the rest
// follow alphabetically, so the receipt always reads the same way.
func PaymentTotals(planAmount float64, planCurrency string, charges []PaymentCharge) []CurrencyTotal {
	planCurrency = NormalizeCurrency(planCurrency, "")
	amounts := map[string]float64{}
	if planAmount > 0 {
		amounts[planCurrency] = planAmount
	}
	for _, charge := range charges {
		amounts[NormalizeCurrency(charge.Moneda, planCurrency)] += charge.Monto
	}
	currencies := make([]string, 0, len(amounts))
	for currency := range amounts {
		if currency != planCurrency {
			currencies = append(currencies, currency)
		}
	}
	sort.Strings(currencies)
	if _, ok := amounts[planCurrency]; ok {
		currencies = append([]string{planCurrency}, currencies...)
	}
	totals := make([]CurrencyTotal, 0, len(currencies))
	for _, currency := range currencies {
		totals = append(totals, CurrencyTotal{Moneda: currency, Monto: RoundMoney(amounts[currency])})
	}
	return totals
}

// ChargeTotals adds up charges by currency, for the statement's running
// total of what was collected beyond the plan.
func ChargeTotals(charges []PaymentCharge) []CurrencyTotal {
	return PaymentTotals(0, "", charges)
}
