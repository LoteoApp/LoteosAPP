package domain

import (
	"math"
	"time"
)

type PaymentPeriod string

const (
	PaymentPeriodMonthly    PaymentPeriod = "mensual"
	PaymentPeriodBimonthly  PaymentPeriod = "bimestral"
	PaymentPeriodQuarterly  PaymentPeriod = "trimestral"
	PaymentPeriodSemiannual PaymentPeriod = "semestral"
)

const (
	MaxSaleInstallments  = 360
	MaxSaleInterestRate  = 1000
	installmentAmountMin = 0.01
	interestRateDecimals = 4
	moneyDecimals        = 2
)

type InstallmentState string

const (
	InstallmentStatePending InstallmentState = "pendiente"
	InstallmentStatePaid    InstallmentState = "pagada"
	InstallmentStateOverdue InstallmentState = "vencida"
)

var (
	ErrSalePaymentPlanRequired      = &Error{Kind: KindInvalid, Code: "sale_payment_plan_required", Message: "Una venta financiada necesita un plan de pago"}
	ErrSalePaymentPlanNotApplicable = &Error{Kind: KindInvalid, Code: "sale_payment_plan_not_applicable", Message: "Una venta al contado no lleva plan de pago"}
	ErrSaleInvalidInstallments      = &Error{Kind: KindInvalid, Code: "invalid_sale_installments", Message: "La cantidad de cuotas tiene que ser un entero entre 1 y 360"}
	ErrSaleInvalidInterestRate      = &Error{Kind: KindInvalid, Code: "invalid_sale_interest_rate", Message: "La tasa de interés tiene que ser un porcentaje entre 0 y 1000"}
	ErrSaleInvalidPeriodicity       = &Error{Kind: KindInvalid, Code: "invalid_sale_periodicity", Message: "La periodicidad de las cuotas no es válida"}
	ErrSaleInvalidDownPayment       = &Error{Kind: KindInvalid, Code: "invalid_sale_down_payment", Message: "La entrega tiene que ser mayor a cero y menor al precio del lote"}
	ErrSaleDownPaymentNotApplicable = &Error{Kind: KindInvalid, Code: "sale_down_payment_not_applicable", Message: "Solo entrega + financiación lleva un monto de entrega"}
	ErrSaleInstallmentTooSmall      = &Error{Kind: KindInvalid, Code: "sale_installment_too_small", Message: "El monto financiado no alcanza para esa cantidad de cuotas"}
)

func (period PaymentPeriod) IsValid() bool {
	_, ok := paymentPeriodMonths[period]
	return ok
}

var paymentPeriodMonths = map[PaymentPeriod]int{
	PaymentPeriodMonthly:    1,
	PaymentPeriodBimonthly:  2,
	PaymentPeriodQuarterly:  3,
	PaymentPeriodSemiannual: 6,
}

// Months is how many months separate two consecutive installments.
func (period PaymentPeriod) Months() int {
	return paymentPeriodMonths[period]
}

// PaymentPlanInput is the plan as the caller describes it: InterestRate is a
// percentage over the financed amount and DownPayment is only meaningful for
// entrega_financiada.
type PaymentPlanInput struct {
	Installments int
	InterestRate float64
	Period       PaymentPeriod
	DownPayment  float64
}

// ValidatePaymentPlan checks the plan against the modalidad before the lote
// price is known: contado takes no plan, the other two require one, and only
// entrega_financiada carries a down payment. The down payment's upper bound
// (the lote price) is checked by BuildPaymentSchedule.
//
// InterestRate is persisted as NUMERIC(8,4) and DownPayment as NUMERIC(14,2),
// so values with more decimals than the columns keep are rejected here rather
// than silently rounded after the schedule was computed on the exact value.
func ValidatePaymentPlan(method PaymentMethod, plan *PaymentPlanInput) error {
	if method == PaymentMethodCash {
		if plan != nil {
			return ErrSalePaymentPlanNotApplicable
		}
		return nil
	}
	if plan == nil {
		return ErrSalePaymentPlanRequired
	}
	if plan.Installments < 1 || plan.Installments > MaxSaleInstallments {
		return ErrSaleInvalidInstallments
	}
	if !isFinite(plan.InterestRate) || plan.InterestRate < 0 || plan.InterestRate > MaxSaleInterestRate || !hasAtMostDecimals(plan.InterestRate, interestRateDecimals) {
		return ErrSaleInvalidInterestRate
	}
	if !plan.Period.IsValid() {
		return ErrSaleInvalidPeriodicity
	}
	if !isFinite(plan.DownPayment) || !hasAtMostDecimals(plan.DownPayment, moneyDecimals) {
		return ErrSaleInvalidDownPayment
	}
	switch method {
	case PaymentMethodFinanced:
		if plan.DownPayment != 0 {
			return ErrSaleDownPaymentNotApplicable
		}
	case PaymentMethodDownAndFi:
		if plan.DownPayment <= 0 {
			return ErrSaleInvalidDownPayment
		}
	}
	return nil
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// hasAtMostDecimals tolerates the representation error of a decimal typed by
// the user (0.1 is not exact in binary) but not a real extra digit.
func hasAtMostDecimals(value float64, decimals int) bool {
	scaled := value * math.Pow10(decimals)
	return math.Abs(scaled-math.Round(scaled)) < 1e-6
}

// Installment is a cuota as the API publishes it.
type Installment struct {
	ID               string           `json:"id,omitempty"`
	Numero           int              `json:"numero"`
	Monto            float64          `json:"monto"`
	Estado           InstallmentState `json:"estado"`
	FechaVencimiento time.Time        `json:"fechaVencimiento"`
	FechaPago        *time.Time       `json:"fechaPago,omitempty"`
}

// PaymentPlan is a plan de pago as the API publishes it. MontoFinanciado,
// MontoCuota and MontoTotal are derived from the sale amount and the plan
// with the same formula the schedule uses, so the client never recomputes
// them; MontoCuota is the regular installment and the last one may differ by
// the rounding remainder.
type PaymentPlan struct {
	ID              string        `json:"id,omitempty"`
	MontoEntrega    float64       `json:"montoEntrega"`
	CantidadCuotas  int           `json:"cantidadCuotas"`
	TasaInteres     float64       `json:"tasaInteres"`
	Periodicidad    PaymentPeriod `json:"periodicidad"`
	Moneda          string        `json:"moneda"`
	MontoFinanciado float64       `json:"montoFinanciado"`
	MontoCuota      float64       `json:"montoCuota"`
	MontoTotal      float64       `json:"montoTotal"`
	Cuotas          []Installment `json:"cuotas,omitempty"`
}

// PaymentSchedule is the deterministic outcome of a plan over a sale amount.
type PaymentSchedule struct {
	FinancedAmount    float64
	TotalAmount       float64
	InstallmentAmount float64
	Installments      []Installment
}

// RoundMoney rounds to 2 decimals, half away from zero. It is written as
// math.Round(value*100)/100 on purpose: the frontend applies
// Math.round(value*100)/100 to the same IEEE 754 doubles, and both agree on
// every positive amount, so the preview in the form matches what is persisted.
func RoundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

// BuildPaymentSchedule computes the cuotas of a plan over the sale amount.
//
// Formula (simple interest on the financed amount; keep in sync with
// buildPaymentSchedule in apps/frontend/src/features/sales/types.ts):
//
//	financed    = round2(amount - downPayment)
//	total       = round2(financed * (1 + interestRate / 100))
//	installment = round2(total / installments)
//	last        = round2(total - installment * (installments - 1))
//
// The last installment absorbs the rounding remainder so the cuotas add up
// to total exactly. Due dates: installment k (1-based) falls k periods after
// the sale date, on the same day of the month, clamped to the last day when
// the target month is shorter (a sale on Jan 31 is due Feb 28/29, Mar 31...).
func BuildPaymentSchedule(amount float64, plan PaymentPlanInput, saleDate time.Time) (PaymentSchedule, error) {
	if plan.DownPayment >= amount {
		return PaymentSchedule{}, ErrSaleInvalidDownPayment
	}
	financed := RoundMoney(amount - plan.DownPayment)
	total := RoundMoney(financed * (1 + plan.InterestRate/100))
	regular := RoundMoney(total / float64(plan.Installments))
	last := RoundMoney(total - regular*float64(plan.Installments-1))
	if regular < installmentAmountMin || last < installmentAmountMin {
		return PaymentSchedule{}, ErrSaleInstallmentTooSmall
	}

	months := plan.Period.Months()
	installments := make([]Installment, plan.Installments)
	for i := range installments {
		number := i + 1
		installment := Installment{
			Numero:           number,
			Monto:            regular,
			Estado:           InstallmentStatePending,
			FechaVencimiento: addMonthsClamped(saleDate, months*number),
		}
		if number == plan.Installments {
			installment.Monto = last
		}
		installments[i] = installment
	}
	return PaymentSchedule{
		FinancedAmount:    financed,
		TotalAmount:       total,
		InstallmentAmount: regular,
		Installments:      installments,
	}, nil
}

func addMonthsClamped(date time.Time, months int) time.Time {
	year, month, day := date.Date()
	first := time.Date(year, month+time.Month(months), 1, 0, 0, 0, 0, date.Location())
	lastDay := first.AddDate(0, 1, -1).Day()
	if day > lastDay {
		day = lastDay
	}
	hour, minute, second := date.Clock()
	return time.Date(first.Year(), first.Month(), day, hour, minute, second, date.Nanosecond(), date.Location())
}
