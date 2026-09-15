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

// PaymentPlanInput is the plan as the caller describes it: TasaInteres is a
// percentage over the financed amount and MontoEntrega is only meaningful
// for entrega_financiada.
type PaymentPlanInput struct {
	CantidadCuotas int
	TasaInteres    float64
	Periodicidad   PaymentPeriod
	MontoEntrega   float64
}

// ValidatePaymentPlan checks the plan against the modalidad before the lote
// price is known: contado takes no plan, the other two require one, and only
// entrega_financiada carries a down payment. The down payment's upper bound
// (the lote price) is checked by BuildPaymentSchedule.
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
	if plan.CantidadCuotas < 1 || plan.CantidadCuotas > MaxSaleInstallments {
		return ErrSaleInvalidInstallments
	}
	if math.IsNaN(plan.TasaInteres) || math.IsInf(plan.TasaInteres, 0) || plan.TasaInteres < 0 || plan.TasaInteres > MaxSaleInterestRate {
		return ErrSaleInvalidInterestRate
	}
	if !plan.Periodicidad.IsValid() {
		return ErrSaleInvalidPeriodicity
	}
	if math.IsNaN(plan.MontoEntrega) || math.IsInf(plan.MontoEntrega, 0) {
		return ErrSaleInvalidDownPayment
	}
	switch method {
	case PaymentMethodFinanced:
		if plan.MontoEntrega != 0 {
			return ErrSaleDownPaymentNotApplicable
		}
	case PaymentMethodDownAndFi:
		if plan.MontoEntrega <= 0 {
			return ErrSaleInvalidDownPayment
		}
	}
	return nil
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
	MontoFinanciado float64
	MontoTotal      float64
	MontoCuota      float64
	Cuotas          []Installment
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
//	financiado = round2(monto - montoEntrega)
//	total      = round2(financiado * (1 + tasaInteres / 100))
//	cuota      = round2(total / cantidadCuotas)
//	última     = round2(total - cuota * (cantidadCuotas - 1))
//
// The last installment absorbs the rounding remainder so the cuotas add up
// to total exactly. Due dates: installment k (1-based) falls k periods after
// the sale date, on the same day of the month, clamped to the last day when
// the target month is shorter (a sale on Jan 31 is due Feb 28/29, Mar 31...).
func BuildPaymentSchedule(monto float64, plan PaymentPlanInput, saleDate time.Time) (PaymentSchedule, error) {
	if plan.MontoEntrega >= monto {
		return PaymentSchedule{}, ErrSaleInvalidDownPayment
	}
	financed := RoundMoney(monto - plan.MontoEntrega)
	total := RoundMoney(financed * (1 + plan.TasaInteres/100))
	amount := RoundMoney(total / float64(plan.CantidadCuotas))
	last := RoundMoney(total - amount*float64(plan.CantidadCuotas-1))
	if amount < installmentAmountMin || last < installmentAmountMin {
		return PaymentSchedule{}, ErrSaleInstallmentTooSmall
	}

	months := plan.Periodicidad.Months()
	installments := make([]Installment, plan.CantidadCuotas)
	for i := range installments {
		number := i + 1
		installment := Installment{
			Numero:           number,
			Monto:            amount,
			Estado:           InstallmentStatePending,
			FechaVencimiento: addMonthsClamped(saleDate, months*number),
		}
		if number == plan.CantidadCuotas {
			installment.Monto = last
		}
		installments[i] = installment
	}
	return PaymentSchedule{
		MontoFinanciado: financed,
		MontoTotal:      total,
		MontoCuota:      amount,
		Cuotas:          installments,
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
