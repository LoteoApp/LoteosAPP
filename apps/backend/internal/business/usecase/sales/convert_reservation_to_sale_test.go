package sales_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/sales"
)

const reservationID = "4f1d5c3a-0000-4000-8000-00000000ab01"

func validConversion() sales.ConvertReservationToSaleInput {
	return sales.ConvertReservationToSaleInput{
		Actor:          adminActor(),
		ReservationID:  " 4F1D5C3A-0000-4000-8000-00000000AB01 ",
		IdempotencyKey: " convert-key ",
	}
}

func TestConvertReservationToSaleBuildsTheCommandFromTheReserva(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.SaleRepository{ConvertResult: domain.Sale{ID: "sale-1"}}
	useCase := sales.NewConvertReservationToSale(repository, users)

	sale, err := useCase.Execute(context.Background(), validConversion())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if sale.ID != "sale-1" || repository.ConvertCalls != 1 {
		t.Fatalf("Execute() = %#v after %d calls, want the repository result", sale, repository.ConvertCalls)
	}
	command := repository.ConvertCommand
	if command.ReservationID != reservationID || command.ActorID != "actor-id" || command.ActorAuthProviderID != "actor-subject" {
		t.Errorf("command identity = %#v", command)
	}
	if command.PaymentMethod != domain.PaymentMethodCash || command.PaymentPlan != nil {
		t.Errorf("command terms = %q/%#v, want contado without plan", command.PaymentMethod, command.PaymentPlan)
	}
	if command.IdempotencyKey != "convert-key" {
		t.Errorf("idempotency key = %q, want the trimmed header", command.IdempotencyKey)
	}
	if want := sha256Hex("reservation-to-sale:v1|36:" + reservationID + "7:contado"); command.IdempotencyPayloadHash != want {
		t.Errorf("payload hash = %q, want %q", command.IdempotencyPayloadHash, want)
	}
}

func TestConvertReservationToSaleHashesTheTermsAndTheReserva(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.SaleRepository{}
	useCase := sales.NewConvertReservationToSale(repository, users)

	financed := validConversion()
	financed.PaymentMethod = " financiado "
	financed.PaymentPlan = &sales.PaymentPlanInput{Installments: 12, InterestRate: 10, Period: " mensual "}
	if _, err := useCase.Execute(context.Background(), financed); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	command := repository.ConvertCommand
	if command.PaymentPlan == nil || command.PaymentPlan.Period != domain.PaymentPeriodMonthly || command.PaymentPlan.Installments != 12 {
		t.Fatalf("normalized plan = %#v", command.PaymentPlan)
	}
	financedHash := command.IdempotencyPayloadHash
	if want := sha256Hex("reservation-to-sale:v1|36:" + reservationID + "10:financiado|12:10:mensual:0"); financedHash != want {
		t.Errorf("financed hash = %q, want %q", financedHash, want)
	}

	otherReservation := financed
	otherReservation.ReservationID = "4f1d5c3a-0000-4000-8000-00000000ab02"
	if _, err := useCase.Execute(context.Background(), otherReservation); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.ConvertCommand.IdempotencyPayloadHash == financedHash {
		t.Error("another reserva must produce another payload hash")
	}
}

func TestConvertReservationToSaleRejectsBeforeTheRepository(t *testing.T) {
	inactive := activeAdmin()
	deleted := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	inactive.FechaBaja = &deleted
	surveyor := activeAdmin()
	surveyor.Rol = domain.RolAgrimensor

	for name, test := range map[string]struct {
		mutate func(*sales.ConvertReservationToSaleInput)
		users  *gatewayfake.UserRepository
		want   error
	}{
		"role without sales": {
			mutate: func(input *sales.ConvertReservationToSaleInput) {
				input.Actor.Roles = []string{domain.RolEscribano}
			},
			want: domain.ErrNoAutorizado,
		},
		"missing key": {
			mutate: func(input *sales.ConvertReservationToSaleInput) { input.IdempotencyKey = " " },
			want:   domain.ErrSaleIdempotencyRequired,
		},
		"key too long": {
			mutate: func(input *sales.ConvertReservationToSaleInput) {
				input.IdempotencyKey = strings.Repeat("k", domain.MaxIdempotencyKeySize+1)
			},
			want: domain.ErrSaleIdempotencyRequired,
		},
		"blank reserva": {
			mutate: func(input *sales.ConvertReservationToSaleInput) { input.ReservationID = "  " },
			want:   domain.ErrReservationNotFound,
		},
		"reserva that is not a uuid": {
			mutate: func(input *sales.ConvertReservationToSaleInput) { input.ReservationID = "reserva-id" },
			want:   domain.ErrReservationNotFound,
		},
		"reserva uuid without hyphens": {
			mutate: func(input *sales.ConvertReservationToSaleInput) {
				input.ReservationID = "4f1d5c3a000040008000000000000ab01"
			},
			want: domain.ErrReservationNotFound,
		},
		"reserva uuid with a misplaced hyphen": {
			mutate: func(input *sales.ConvertReservationToSaleInput) {
				input.ReservationID = "4f1d5c3a0-000-4000-8000-00000000ab01"
			},
			want: domain.ErrReservationNotFound,
		},
		"reserva uuid with a non hex digit": {
			mutate: func(input *sales.ConvertReservationToSaleInput) {
				input.ReservationID = "4f1d5c3a-0000-4000-8000-00000000zb01"
			},
			want: domain.ErrReservationNotFound,
		},
		"unknown method": {
			mutate: func(input *sales.ConvertReservationToSaleInput) { input.PaymentMethod = "trueque" },
			want:   domain.ErrSaleInvalidPaymentMethod,
		},
		"contado with a plan": {
			mutate: func(input *sales.ConvertReservationToSaleInput) {
				input.PaymentPlan = &sales.PaymentPlanInput{Installments: 3, Period: "mensual"}
			},
			want: domain.ErrSalePaymentPlanNotApplicable,
		},
		"financed without a plan": {
			mutate: func(input *sales.ConvertReservationToSaleInput) { input.PaymentMethod = "financiado" },
			want:   domain.ErrSalePaymentPlanRequired,
		},
		"unprovisioned actor": {
			users: &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado},
			want:  domain.ErrActorNoAprovisionado,
		},
		"inactive actor": {
			users: &gatewayfake.UserRepository{FindByAuthProviderIDResult: inactive},
			want:  domain.ErrCuentaInactiva,
		},
		"stored role without sales": {
			users: &gatewayfake.UserRepository{FindByAuthProviderIDResult: surveyor},
			want:  domain.ErrNoAutorizado,
		},
	} {
		t.Run(name, func(t *testing.T) {
			users := test.users
			if users == nil {
				users = &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
			}
			repository := &gatewayfake.SaleRepository{}
			input := validConversion()
			if test.mutate != nil {
				test.mutate(&input)
			}
			_, err := sales.NewConvertReservationToSale(repository, users).Execute(context.Background(), input)
			if !errors.Is(err, test.want) {
				t.Fatalf("Execute() error = %v, want %v", err, test.want)
			}
			if repository.ConvertCalls != 0 {
				t.Errorf("repository called %d times, want none", repository.ConvertCalls)
			}
		})
	}
}

func TestConvertReservationToSaleMapsRepositoryErrors(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}

	business := &gatewayfake.SaleRepository{ConvertErr: domain.ErrReservationConvertForbidden}
	if _, err := sales.NewConvertReservationToSale(business, users).Execute(context.Background(), validConversion()); !errors.Is(err, domain.ErrReservationConvertForbidden) {
		t.Fatalf("Execute() business error = %v, want it unchanged", err)
	}

	cause := errors.New("connection reset")
	failing := &gatewayfake.SaleRepository{ConvertErr: cause}
	_, err := sales.NewConvertReservationToSale(failing, users).Execute(context.Background(), validConversion())
	if !errors.Is(err, domain.ErrDatabaseUnavailable) || !errors.Is(err, cause) {
		t.Fatalf("Execute() unexpected error = %v, want database unavailable wrapping the cause", err)
	}

	lookup := &gatewayfake.UserRepository{FindByAuthProviderIDErr: cause}
	_, err = sales.NewConvertReservationToSale(&gatewayfake.SaleRepository{}, lookup).Execute(context.Background(), validConversion())
	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() user lookup error = %v, want database unavailable", err)
	}
}
