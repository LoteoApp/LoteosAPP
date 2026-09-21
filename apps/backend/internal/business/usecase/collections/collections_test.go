package collections_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/collections"
)

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

var now = time.Date(2026, 9, 21, 15, 0, 0, 0, time.FixedZone("ART", -3*60*60))

func activeAdmin() domain.Usuario {
	return domain.Usuario{ID: "actor-id", AuthProviderID: "actor-subject", Rol: domain.RolAdministrador}
}

func adminActor() collections.Actor {
	return collections.Actor{AuthProviderID: "actor-subject", Roles: []string{domain.RolAdministrador}}
}

func agencyActor() collections.Actor {
	return collections.Actor{AuthProviderID: "agency-subject", Roles: []string{domain.RolInmobiliaria}}
}

func surveyorActor() collections.Actor {
	return collections.Actor{AuthProviderID: "surveyor-subject", Roles: []string{domain.RolAgrimensor}}
}

func TestGetDebtStatement(t *testing.T) {
	repository := &gatewayfake.CollectionRepository{GetDebtStatementResult: domain.DebtStatement{Venta: domain.Sale{ID: "sale-1"}}}
	useCase := collections.NewGetDebtStatement(repository, fixedClock{now: now})

	statement, err := useCase.Execute(context.Background(), adminActor(), " sale-1 ")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if statement.Venta.ID != "sale-1" || repository.GetDebtStatementID != "sale-1" {
		t.Errorf("statement = %#v, id = %q", statement, repository.GetDebtStatementID)
	}
	if !repository.GetDebtStatementNow.Equal(now) || repository.GetDebtStatementNow.Location() != time.UTC {
		t.Errorf("now = %s, want %s in UTC", repository.GetDebtStatementNow, now)
	}
	if repository.GetDebtStatementScope.AssigneeAuthProviderID != nil || repository.GetDebtStatementScope.ByAgency {
		t.Errorf("admin scope = %#v, want unrestricted", repository.GetDebtStatementScope)
	}

	if _, err := useCase.Execute(context.Background(), agencyActor(), "sale-1"); err != nil {
		t.Fatalf("agency Execute() error = %v", err)
	}
	scope := repository.GetDebtStatementScope
	if scope.AssigneeAuthProviderID == nil || *scope.AssigneeAuthProviderID != "agency-subject" || !scope.ByAgency {
		t.Errorf("agency scope = %#v", scope)
	}

	if _, err := useCase.Execute(context.Background(), surveyorActor(), "sale-1"); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Errorf("surveyor error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if _, err := useCase.Execute(context.Background(), adminActor(), "  "); !errors.Is(err, domain.ErrSaleNotFound) {
		t.Errorf("blank id error = %v, want %v", err, domain.ErrSaleNotFound)
	}

	repository.GetDebtStatementErr = domain.ErrSaleNotFinanced
	if _, err := useCase.Execute(context.Background(), adminActor(), "sale-1"); !errors.Is(err, domain.ErrSaleNotFinanced) {
		t.Errorf("domain error = %v, want it passed through", err)
	}
	repository.GetDebtStatementErr = errors.New("boom")
	_, err = useCase.Execute(context.Background(), adminActor(), "sale-1")
	if !errors.Is(err, domain.ErrDatabaseUnavailable) || !strings.Contains(errors.Unwrap(err).Error(), "boom") {
		t.Errorf("unexpected error = %v, want database unavailable wrapping boom", err)
	}
}

func TestListDueInstallments(t *testing.T) {
	repository := &gatewayfake.CollectionRepository{ListDueInstallmentsResult: domain.DueInstallmentPage{Total: 3}}
	useCase := collections.NewListDueInstallments(repository, fixedClock{now: now})

	page, err := useCase.Execute(context.Background(), collections.ListDueInstallmentsInput{
		Actor:  adminActor(),
		Filter: domain.DueInstallmentFilter{Search: " ana ", Page: 2},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if page.Total != 3 {
		t.Errorf("page = %#v", page)
	}
	filter := repository.ListDueInstallmentsFilter
	if filter.Search != "ana" || filter.Page != 2 || filter.Limit != domain.DefaultDueInstallmentLimit || len(filter.States) != 2 {
		t.Errorf("normalized filter = %#v", filter)
	}
	if !repository.ListDueInstallmentsNow.Equal(now) {
		t.Errorf("now = %s", repository.ListDueInstallmentsNow)
	}

	if _, err := useCase.Execute(context.Background(), collections.ListDueInstallmentsInput{Actor: adminActor(), Filter: domain.DueInstallmentFilter{Page: -1}}); !errors.Is(err, domain.ErrDueInstallmentInvalidPage) {
		t.Errorf("invalid page error = %v", err)
	}
	if _, err := useCase.Execute(context.Background(), collections.ListDueInstallmentsInput{Actor: surveyorActor()}); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Errorf("surveyor error = %v", err)
	}
	repository.ListDueInstallmentsErr = errors.New("down")
	if _, err := useCase.Execute(context.Background(), collections.ListDueInstallmentsInput{Actor: adminActor()}); !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Errorf("repository failure error = %v", err)
	}
}

func TestRegisterPayment(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.CollectionRepository{RegisterPaymentResult: domain.Payment{ID: "cobro-1"}}
	useCase := collections.NewRegisterPayment(repository, users, fixedClock{now: now})
	paidAt := now.Add(-24 * time.Hour)

	payment, err := useCase.Execute(context.Background(), collections.RegisterPaymentInput{
		Actor:              adminActor(),
		SaleID:             " sale-1 ",
		InstallmentIDs:     []string{" c-1 ", "", "c-2"},
		IncludeDownPayment: true,
		Medium:             " transferencia ",
		PaidAt:             &paidAt,
		Observation:        "  Comprobante 123  ",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if payment.ID != "cobro-1" {
		t.Errorf("payment = %#v", payment)
	}
	command := repository.RegisterPaymentCommand
	if command.SaleID != "sale-1" || command.ActorID != "actor-id" || command.Type != domain.PaymentTypeRegular {
		t.Errorf("command = %#v", command)
	}
	if len(command.InstallmentIDs) != 2 || command.InstallmentIDs[0] != "c-1" || command.InstallmentIDs[1] != "c-2" || !command.IncludeDownPayment {
		t.Errorf("selection = %#v / %v", command.InstallmentIDs, command.IncludeDownPayment)
	}
	if command.Medium != domain.PaymentMediumTransfer || command.Observation != "Comprobante 123" || command.ExpectedAmount != nil {
		t.Errorf("terms = %#v", command)
	}
	if !command.PaidAt.Equal(paidAt) || command.PaidAt.Location() != time.UTC || !command.Now.Equal(now) {
		t.Errorf("dates = %s / %s", command.PaidAt, command.Now)
	}

	withoutDate, err := useCase.Execute(context.Background(), collections.RegisterPaymentInput{
		Actor: adminActor(), SaleID: "sale-1", InstallmentIDs: []string{"c-1"}, Medium: "efectivo",
	})
	if err != nil || withoutDate.ID != "cobro-1" {
		t.Fatalf("Execute() without date = %#v, %v", withoutDate, err)
	}
	if !repository.RegisterPaymentCommand.PaidAt.Equal(now) {
		t.Errorf("default paid at = %s, want now", repository.RegisterPaymentCommand.PaidAt)
	}
}

func TestRegisterPaymentRejections(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.CollectionRepository{}
	useCase := collections.NewRegisterPayment(repository, users, fixedClock{now: now})
	future := now.Add(time.Hour)
	valid := collections.RegisterPaymentInput{Actor: adminActor(), SaleID: "sale-1", InstallmentIDs: []string{"c-1"}, Medium: "efectivo"}

	cases := []struct {
		name  string
		input collections.RegisterPaymentInput
		want  error
	}{
		{"surveyor", collections.RegisterPaymentInput{Actor: surveyorActor(), SaleID: "sale-1", InstallmentIDs: []string{"c-1"}, Medium: "efectivo"}, domain.ErrNoAutorizado},
		{"blank sale", collections.RegisterPaymentInput{Actor: adminActor(), InstallmentIDs: []string{"c-1"}, Medium: "efectivo"}, domain.ErrSaleNotFound},
		{"nothing selected", collections.RegisterPaymentInput{Actor: adminActor(), SaleID: "sale-1", InstallmentIDs: []string{" "}, Medium: "efectivo"}, domain.ErrPaymentNothingSelected},
		{"invalid medium", collections.RegisterPaymentInput{Actor: adminActor(), SaleID: "sale-1", InstallmentIDs: []string{"c-1"}, Medium: "canje"}, domain.ErrPaymentInvalidMedium},
		{"future date", collections.RegisterPaymentInput{Actor: adminActor(), SaleID: "sale-1", InstallmentIDs: []string{"c-1"}, Medium: "efectivo", PaidAt: &future}, domain.ErrPaymentDateInFuture},
		{"observation too long", collections.RegisterPaymentInput{Actor: adminActor(), SaleID: "sale-1", InstallmentIDs: []string{"c-1"}, Medium: "efectivo", Observation: strings.Repeat("x", 501)}, domain.ErrPaymentObservationTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := useCase.Execute(context.Background(), tc.input); !errors.Is(err, tc.want) {
				t.Errorf("Execute() error = %v, want %v", err, tc.want)
			}
			if repository.RegisterPaymentCalls != 0 {
				t.Errorf("repository called %d times, want none", repository.RegisterPaymentCalls)
			}
		})
	}

	// The token says administrador but the row says otherwise: the row wins.
	users.FindByAuthProviderIDResult = domain.Usuario{ID: "actor-id", Rol: domain.RolEscribano}
	if _, err := useCase.Execute(context.Background(), valid); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Errorf("role mismatch error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	users.FindByAuthProviderIDResult = activeAdmin()
	users.FindByAuthProviderIDErr = domain.ErrUsuarioNoEncontrado
	if _, err := useCase.Execute(context.Background(), valid); !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Errorf("unprovisioned actor error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
	users.FindByAuthProviderIDErr = nil
	deactivated := now
	users.FindByAuthProviderIDResult = domain.Usuario{ID: "actor-id", Rol: domain.RolAdministrador, FechaBaja: &deactivated}
	if _, err := useCase.Execute(context.Background(), valid); !errors.Is(err, domain.ErrCuentaInactiva) {
		t.Errorf("inactive actor error = %v, want %v", err, domain.ErrCuentaInactiva)
	}
	users.FindByAuthProviderIDResult = activeAdmin()

	repository.RegisterPaymentErr = domain.ErrPaymentInstallmentOrder
	if _, err := useCase.Execute(context.Background(), valid); !errors.Is(err, domain.ErrPaymentInstallmentOrder) {
		t.Errorf("domain error = %v, want it passed through", err)
	}
	repository.RegisterPaymentErr = errors.New("boom")
	if _, err := useCase.Execute(context.Background(), valid); !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Errorf("unexpected error = %v, want database unavailable", err)
	}
}

func TestSettleSale(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.CollectionRepository{RegisterPaymentResult: domain.Payment{ID: "cobro-9", Tipo: domain.PaymentTypeSettlement}}
	useCase := collections.NewSettleSale(repository, users, fixedClock{now: now})
	expected := 12345.67

	payment, err := useCase.Execute(context.Background(), collections.SettleSaleInput{
		Actor: agencyActor(), SaleID: " sale-1 ", ExpectedAmount: &expected, Medium: "cheque", Observation: " Cheque 44 ",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if payment.ID != "cobro-9" {
		t.Errorf("payment = %#v", payment)
	}
	command := repository.RegisterPaymentCommand
	if command.SaleID != "sale-1" || command.Type != domain.PaymentTypeSettlement || command.ExpectedAmount == nil || *command.ExpectedAmount != expected {
		t.Errorf("command = %#v", command)
	}
	if command.Medium != domain.PaymentMediumCheque || command.Observation != "Cheque 44" || !command.PaidAt.Equal(now) || len(command.InstallmentIDs) != 0 || command.IncludeDownPayment {
		t.Errorf("terms = %#v", command)
	}
	scope := repository.RegisterPaymentScope
	if scope.AssigneeAuthProviderID == nil || *scope.AssigneeAuthProviderID != "agency-subject" || !scope.ByAgency {
		t.Errorf("agency scope = %#v", scope)
	}

	if _, err := useCase.Execute(context.Background(), collections.SettleSaleInput{Actor: surveyorActor(), SaleID: "sale-1", Medium: "efectivo"}); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Errorf("surveyor error = %v", err)
	}
	if _, err := useCase.Execute(context.Background(), collections.SettleSaleInput{Actor: adminActor(), Medium: "efectivo"}); !errors.Is(err, domain.ErrSaleNotFound) {
		t.Errorf("blank sale error = %v", err)
	}
	if _, err := useCase.Execute(context.Background(), collections.SettleSaleInput{Actor: adminActor(), SaleID: "sale-1", Medium: ""}); !errors.Is(err, domain.ErrPaymentInvalidMedium) {
		t.Errorf("blank medium error = %v", err)
	}
	users.FindByAuthProviderIDResult = domain.Usuario{ID: "actor-id", Rol: domain.RolAgrimensor}
	if _, err := useCase.Execute(context.Background(), collections.SettleSaleInput{Actor: adminActor(), SaleID: "sale-1", Medium: "efectivo"}); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Errorf("role mismatch error = %v", err)
	}
	users.FindByAuthProviderIDResult = activeAdmin()
	users.FindByAuthProviderIDErr = errors.New("db down")
	if _, err := useCase.Execute(context.Background(), collections.SettleSaleInput{Actor: adminActor(), SaleID: "sale-1", Medium: "efectivo"}); !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Errorf("user lookup failure error = %v", err)
	}
	users.FindByAuthProviderIDErr = nil
	repository.RegisterPaymentErr = domain.ErrSettlementAmountMismatch
	if _, err := useCase.Execute(context.Background(), collections.SettleSaleInput{Actor: adminActor(), SaleID: "sale-1", Medium: "efectivo"}); !errors.Is(err, domain.ErrSettlementAmountMismatch) {
		t.Errorf("mismatch error = %v", err)
	}
}

func TestSystemClockAndDefaultScope(t *testing.T) {
	before := time.Now()
	got := collections.SystemClock{}.Now()
	if got.Before(before) || got.After(time.Now()) {
		t.Errorf("SystemClock.Now() = %s, want between %s and now", got, before)
	}
	repository := &gatewayfake.CollectionRepository{}
	useCase := collections.NewGetDebtStatement(repository, nil)
	if _, err := useCase.Execute(context.Background(), adminActor(), "sale-1"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.GetDebtStatementNow.IsZero() {
		t.Error("nil clock should fall back to the system clock")
	}
	if repository.GetDebtStatementScope != (gateway.SaleScope{}) {
		t.Errorf("scope = %#v", repository.GetDebtStatementScope)
	}
}
