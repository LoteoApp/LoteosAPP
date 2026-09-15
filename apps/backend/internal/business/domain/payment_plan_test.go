package domain_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
)

func monthlyPlan(installments int, rate float64) domain.PaymentPlanInput {
	return domain.PaymentPlanInput{CantidadCuotas: installments, TasaInteres: rate, Periodicidad: domain.PaymentPeriodMonthly}
}

func sumInstallments(installments []domain.Installment) float64 {
	total := 0.0
	for _, installment := range installments {
		total += installment.Monto
	}
	return domain.RoundMoney(total)
}

func TestPaymentPeriodValidityAndMonths(t *testing.T) {
	for period, months := range map[domain.PaymentPeriod]int{
		domain.PaymentPeriodMonthly: 1, domain.PaymentPeriodBimonthly: 2,
		domain.PaymentPeriodQuarterly: 3, domain.PaymentPeriodSemiannual: 6,
	} {
		if !period.IsValid() || period.Months() != months {
			t.Errorf("%q: valid=%v months=%d, want %d", period, period.IsValid(), period.Months(), months)
		}
	}
	if domain.PaymentPeriod("semanal").IsValid() || domain.PaymentPeriod("").IsValid() {
		t.Error("semanal and empty are not periods")
	}
}

func TestValidatePaymentPlan(t *testing.T) {
	valid := monthlyPlan(12, 10)
	withDown := valid
	withDown.MontoEntrega = 1000

	tests := []struct {
		name   string
		method domain.PaymentMethod
		plan   *domain.PaymentPlanInput
		want   error
	}{
		{"contado without plan", domain.PaymentMethodCash, nil, nil},
		{"contado with plan", domain.PaymentMethodCash, &valid, domain.ErrSalePaymentPlanNotApplicable},
		{"financiado without plan", domain.PaymentMethodFinanced, nil, domain.ErrSalePaymentPlanRequired},
		{"financiado valid", domain.PaymentMethodFinanced, &valid, nil},
		{"financiado with down payment", domain.PaymentMethodFinanced, &withDown, domain.ErrSaleDownPaymentNotApplicable},
		{"entrega valid", domain.PaymentMethodDownAndFi, &withDown, nil},
		{"entrega without down payment", domain.PaymentMethodDownAndFi, &valid, domain.ErrSaleInvalidDownPayment},
		{"entrega negative down payment", domain.PaymentMethodDownAndFi, func() *domain.PaymentPlanInput {
			plan := withDown
			plan.MontoEntrega = -1
			return &plan
		}(), domain.ErrSaleInvalidDownPayment},
		{"entrega NaN down payment", domain.PaymentMethodDownAndFi, func() *domain.PaymentPlanInput {
			plan := withDown
			plan.MontoEntrega = math.NaN()
			return &plan
		}(), domain.ErrSaleInvalidDownPayment},
		{"zero installments", domain.PaymentMethodFinanced, func() *domain.PaymentPlanInput {
			plan := valid
			plan.CantidadCuotas = 0
			return &plan
		}(), domain.ErrSaleInvalidInstallments},
		{"too many installments", domain.PaymentMethodFinanced, func() *domain.PaymentPlanInput {
			plan := valid
			plan.CantidadCuotas = domain.MaxSaleInstallments + 1
			return &plan
		}(), domain.ErrSaleInvalidInstallments},
		{"negative rate", domain.PaymentMethodFinanced, func() *domain.PaymentPlanInput {
			plan := valid
			plan.TasaInteres = -0.5
			return &plan
		}(), domain.ErrSaleInvalidInterestRate},
		{"rate above max", domain.PaymentMethodFinanced, func() *domain.PaymentPlanInput {
			plan := valid
			plan.TasaInteres = domain.MaxSaleInterestRate + 1
			return &plan
		}(), domain.ErrSaleInvalidInterestRate},
		{"infinite rate", domain.PaymentMethodFinanced, func() *domain.PaymentPlanInput {
			plan := valid
			plan.TasaInteres = math.Inf(1)
			return &plan
		}(), domain.ErrSaleInvalidInterestRate},
		{"unknown periodicity", domain.PaymentMethodFinanced, func() *domain.PaymentPlanInput {
			plan := valid
			plan.Periodicidad = "semanal"
			return &plan
		}(), domain.ErrSaleInvalidPeriodicity},
		{"single installment without interest", domain.PaymentMethodFinanced, func() *domain.PaymentPlanInput {
			plan := monthlyPlan(1, 0)
			return &plan
		}(), nil},
		{"max installments and rate", domain.PaymentMethodFinanced, func() *domain.PaymentPlanInput {
			plan := monthlyPlan(domain.MaxSaleInstallments, domain.MaxSaleInterestRate)
			return &plan
		}(), nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := domain.ValidatePaymentPlan(test.method, test.plan)
			if !errors.Is(err, test.want) {
				t.Fatalf("ValidatePaymentPlan() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestRoundMoney(t *testing.T) {
	for value, want := range map[float64]float64{
		1.005:    1.0, // 1.005 is 1.00499999... in binary, exactly as in JavaScript
		2.675:    2.68,
		0.125:    0.13,
		100:      100,
		33.33333: 33.33,
		0.004:    0,
	} {
		if got := domain.RoundMoney(value); got != want {
			t.Errorf("RoundMoney(%v) = %v, want %v", value, got, want)
		}
	}
}

func TestBuildPaymentScheduleWithoutInterest(t *testing.T) {
	saleDate := time.Date(2026, 9, 15, 12, 30, 0, 0, time.UTC)
	schedule, err := domain.BuildPaymentSchedule(100000, monthlyPlan(12, 0), saleDate)
	if err != nil {
		t.Fatalf("BuildPaymentSchedule() error = %v", err)
	}
	if schedule.MontoFinanciado != 100000 || schedule.MontoTotal != 100000 || schedule.MontoCuota != 8333.33 {
		t.Errorf("schedule amounts = %#v", schedule)
	}
	if len(schedule.Cuotas) != 12 {
		t.Fatalf("installments = %d, want 12", len(schedule.Cuotas))
	}
	for i, installment := range schedule.Cuotas[:11] {
		if installment.Numero != i+1 || installment.Monto != 8333.33 || installment.Estado != domain.InstallmentStatePending {
			t.Errorf("installment %d = %#v", i+1, installment)
		}
	}
	// 100000 - 11 * 8333.33 = 8333.37: the last one absorbs the remainder.
	if last := schedule.Cuotas[11]; last.Numero != 12 || last.Monto != 8333.37 {
		t.Errorf("last installment = %#v, want 8333.37", last)
	}
	if got := sumInstallments(schedule.Cuotas); got != 100000 {
		t.Errorf("sum of installments = %v, want 100000", got)
	}
	for i, installment := range schedule.Cuotas {
		want := time.Date(2026, time.Month(10+i), 15, 12, 30, 0, 0, time.UTC)
		if !installment.FechaVencimiento.Equal(want) {
			t.Errorf("installment %d due = %s, want %s", i+1, installment.FechaVencimiento, want)
		}
	}
}

func TestBuildPaymentScheduleWithInterestAndDownPayment(t *testing.T) {
	saleDate := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	plan := domain.PaymentPlanInput{CantidadCuotas: 3, TasaInteres: 10, Periodicidad: domain.PaymentPeriodMonthly, MontoEntrega: 1000}
	schedule, err := domain.BuildPaymentSchedule(1100, plan, saleDate)
	if err != nil {
		t.Fatalf("BuildPaymentSchedule() error = %v", err)
	}
	// financed 100, total 110, cuota 36.67, last 110 - 73.34 = 36.66.
	if schedule.MontoFinanciado != 100 || schedule.MontoTotal != 110 || schedule.MontoCuota != 36.67 {
		t.Errorf("schedule amounts = %#v", schedule)
	}
	if schedule.Cuotas[2].Monto != 36.66 || sumInstallments(schedule.Cuotas) != 110 {
		t.Errorf("installments = %#v", schedule.Cuotas)
	}
	// Due dates clamp to the end of shorter months instead of spilling over.
	wantDue := []time.Time{
		time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
	}
	for i, installment := range schedule.Cuotas {
		if !installment.FechaVencimiento.Equal(wantDue[i]) {
			t.Errorf("installment %d due = %s, want %s", i+1, installment.FechaVencimiento, wantDue[i])
		}
	}
}

func TestBuildPaymentScheduleSpacesInstallmentsByPeriod(t *testing.T) {
	saleDate := time.Date(2026, 11, 30, 9, 0, 0, 0, time.UTC)
	plan := domain.PaymentPlanInput{CantidadCuotas: 2, TasaInteres: 0, Periodicidad: domain.PaymentPeriodQuarterly}
	schedule, err := domain.BuildPaymentSchedule(50, plan, saleDate)
	if err != nil {
		t.Fatalf("BuildPaymentSchedule() error = %v", err)
	}
	wantDue := []time.Time{
		time.Date(2027, 2, 28, 9, 0, 0, 0, time.UTC),
		time.Date(2027, 5, 30, 9, 0, 0, 0, time.UTC),
	}
	for i, installment := range schedule.Cuotas {
		if !installment.FechaVencimiento.Equal(wantDue[i]) || installment.Monto != 25 {
			t.Errorf("installment %d = %#v, want due %s", i+1, installment, wantDue[i])
		}
	}
}

func TestBuildPaymentScheduleSingleInstallmentAndRounding(t *testing.T) {
	saleDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	schedule, err := domain.BuildPaymentSchedule(999.99, monthlyPlan(1, 12.5), saleDate)
	if err != nil {
		t.Fatalf("BuildPaymentSchedule() error = %v", err)
	}
	// 999.99 * 1.125 = 1124.98875 -> 1124.99 in a single cuota.
	if schedule.MontoTotal != 1124.99 || schedule.MontoCuota != 1124.99 || len(schedule.Cuotas) != 1 || schedule.Cuotas[0].Monto != 1124.99 {
		t.Errorf("schedule = %#v", schedule)
	}

	// Every schedule adds up to its total regardless of the count.
	for _, count := range []int{2, 7, 13, 97, domain.MaxSaleInstallments} {
		schedule, err := domain.BuildPaymentSchedule(123456.78, monthlyPlan(count, 33.3), saleDate)
		if err != nil {
			t.Fatalf("BuildPaymentSchedule(%d) error = %v", count, err)
		}
		if got := sumInstallments(schedule.Cuotas); got != schedule.MontoTotal {
			t.Errorf("%d installments sum to %v, want %v", count, got, schedule.MontoTotal)
		}
	}
}

func TestBuildPaymentScheduleRejectsImpossiblePlans(t *testing.T) {
	saleDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	plan := domain.PaymentPlanInput{CantidadCuotas: 2, Periodicidad: domain.PaymentPeriodMonthly, MontoEntrega: 100}
	if _, err := domain.BuildPaymentSchedule(100, plan, saleDate); !errors.Is(err, domain.ErrSaleInvalidDownPayment) {
		t.Errorf("down payment equal to the price: error = %v, want %v", err, domain.ErrSaleInvalidDownPayment)
	}
	plan.MontoEntrega = 150
	if _, err := domain.BuildPaymentSchedule(100, plan, saleDate); !errors.Is(err, domain.ErrSaleInvalidDownPayment) {
		t.Errorf("down payment above the price: error = %v, want %v", err, domain.ErrSaleInvalidDownPayment)
	}
	if _, err := domain.BuildPaymentSchedule(0.05, monthlyPlan(10, 0), saleDate); !errors.Is(err, domain.ErrSaleInstallmentTooSmall) {
		t.Errorf("installments below a cent: error = %v, want %v", err, domain.ErrSaleInstallmentTooSmall)
	}
	// 0.03 in 2 cuotas leaves 0.02 + 0.01: still valid.
	if _, err := domain.BuildPaymentSchedule(0.03, monthlyPlan(2, 0), saleDate); err != nil {
		t.Errorf("two cents in two installments: error = %v", err)
	}
	// 0.04 in 3 cuotas rounds to 0.01 each and 0.02 for the last one.
	schedule, err := domain.BuildPaymentSchedule(0.04, monthlyPlan(3, 0), saleDate)
	if err != nil || schedule.Cuotas[2].Monto != 0.02 {
		t.Errorf("four cents in three installments = %#v, error = %v", schedule, err)
	}
}
