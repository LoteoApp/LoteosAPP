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

func activeAgencyUser() domain.Usuario {
	return domain.Usuario{ID: "agency-user-id", AuthProviderID: "agency-subject", Rol: domain.RolInmobiliaria}
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
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.CollectionRepository{GetDebtStatementResult: domain.DebtStatement{Sale: domain.Sale{ID: "sale-1"}}}
	useCase := collections.NewGetDebtStatement(repository, users, fixedClock{now: now})

	statement, err := useCase.Execute(context.Background(), adminActor(), " sale-1 ")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if statement.Sale.ID != "sale-1" || repository.GetDebtStatementID != "sale-1" {
		t.Errorf("statement = %#v, id = %q", statement, repository.GetDebtStatementID)
	}
	if !repository.GetDebtStatementNow.Equal(now) || repository.GetDebtStatementNow.Location() != time.UTC {
		t.Errorf("now = %s, want %s in UTC", repository.GetDebtStatementNow, now)
	}
	if repository.GetDebtStatementScope.AssigneeAuthProviderID != nil || repository.GetDebtStatementScope.ByAgency {
		t.Errorf("admin scope = %#v, want unrestricted", repository.GetDebtStatementScope)
	}

	users.FindByAuthProviderIDResult = activeAgencyUser()
	if _, err := useCase.Execute(context.Background(), agencyActor(), "sale-1"); err != nil {
		t.Fatalf("agency Execute() error = %v", err)
	}
	scope := repository.GetDebtStatementScope
	if scope.AssigneeAuthProviderID == nil || *scope.AssigneeAuthProviderID != "agency-subject" || !scope.ByAgency {
		t.Errorf("agency scope = %#v", scope)
	}

	users.FindByAuthProviderIDResult = domain.Usuario{ID: "surveyor-id", Rol: domain.RolAgrimensor}
	if _, err := useCase.Execute(context.Background(), surveyorActor(), "sale-1"); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Errorf("surveyor error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	users.FindByAuthProviderIDResult = activeAdmin()
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
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.CollectionRepository{ListDueInstallmentsResult: domain.DueInstallmentPage{Total: 3}}
	useCase := collections.NewListDueInstallments(repository, users, fixedClock{now: now})

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
	users.FindByAuthProviderIDResult = domain.Usuario{ID: "surveyor-id", Rol: domain.RolAgrimensor}
	if _, err := useCase.Execute(context.Background(), collections.ListDueInstallmentsInput{Actor: surveyorActor()}); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Errorf("surveyor error = %v", err)
	}
	users.FindByAuthProviderIDResult = activeAdmin()
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
	if command.Medium != domain.PaymentMediumTransfer || command.Observation != "Comprobante 123" || command.ExpectedAmount != 0 {
		t.Errorf("terms = %#v", command)
	}
	if !command.PaidAt.Equal(paidAt) || command.PaidAt.Location() != time.UTC || !command.Now.Equal(now) {
		t.Errorf("dates = %s / %s", command.PaidAt, command.Now)
	}

	if len(command.Charges) != 0 {
		t.Errorf("charges = %#v, want none", command.Charges)
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

func TestRegisterPaymentNormalizesTheCharges(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.CollectionRepository{RegisterPaymentResult: domain.Payment{ID: "cobro-1"}}
	useCase := collections.NewRegisterPayment(repository, users, fixedClock{now: now})

	if _, err := useCase.Execute(context.Background(), collections.RegisterPaymentInput{
		Actor: adminActor(), SaleID: "sale-1", InstallmentIDs: []string{"c-1"}, Medium: "efectivo",
		Charges: []collections.ChargeInput{
			{Type: " servicios ", Amount: 150000, Currency: " ars ", Detail: "  Agua  "},
			{Type: "gasto_administrativo", Amount: 25.5},
		},
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	charges := repository.RegisterPaymentCommand.Charges
	if len(charges) != 2 {
		t.Fatalf("charges = %#v", charges)
	}
	if charges[0].Type != domain.ChargeTypeServices || charges[0].Amount != 150000 || charges[0].Currency != "ARS" || charges[0].Detail != "Agua" {
		t.Errorf("first charge = %#v", charges[0])
	}
	// A charge with no currency keeps it blank here: only the repository
	// knows the sale's currency to fall back to.
	if charges[1].Type != domain.ChargeTypeAdministrative || charges[1].Amount != 25.5 || charges[1].Currency != "" {
		t.Errorf("second charge = %#v", charges[1])
	}

	invalid := []struct {
		name   string
		charge collections.ChargeInput
		want   error
	}{
		{"unknown type", collections.ChargeInput{Type: "propina", Amount: 10}, domain.ErrChargeInvalidType},
		{"zero amount", collections.ChargeInput{Type: "servicios", Amount: 0}, domain.ErrChargeInvalidAmount},
		{"detail too long", collections.ChargeInput{Type: "servicios", Amount: 10, Detail: strings.Repeat("x", 201)}, domain.ErrChargeDetailTooLong},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			calls := repository.RegisterPaymentCalls
			_, err := useCase.Execute(context.Background(), collections.RegisterPaymentInput{
				Actor: adminActor(), SaleID: "sale-1", InstallmentIDs: []string{"c-1"}, Medium: "efectivo",
				Charges: []collections.ChargeInput{tc.charge},
			})
			if !errors.Is(err, tc.want) {
				t.Errorf("Execute() error = %v, want %v", err, tc.want)
			}
			if repository.RegisterPaymentCalls != calls {
				t.Error("an invalid charge should not reach the repository")
			}
		})
	}
}

func TestSettleSaleCarriesTheCharges(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.CollectionRepository{RegisterPaymentResult: domain.Payment{ID: "cobro-9"}}
	useCase := collections.NewSettleSale(repository, users, fixedClock{now: now})
	settled := 900.0

	if _, err := useCase.Execute(context.Background(), collections.SettleSaleInput{
		Actor: adminActor(), SaleID: "sale-1", ExpectedAmount: &settled, Medium: "efectivo",
		Charges: []collections.ChargeInput{{Type: "honorarios", Amount: 900, Currency: "ars"}},
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	charges := repository.RegisterPaymentCommand.Charges
	if len(charges) != 1 || charges[0].Type != domain.ChargeTypeFees || charges[0].Currency != "ARS" || charges[0].Amount != 900 {
		t.Errorf("charges = %#v", charges)
	}

	calls := repository.RegisterPaymentCalls
	_, err := useCase.Execute(context.Background(), collections.SettleSaleInput{
		Actor: adminActor(), SaleID: "sale-1", ExpectedAmount: &settled, Medium: "efectivo",
		Charges: []collections.ChargeInput{{Type: "servicios", Amount: 1.555}},
	})
	if !errors.Is(err, domain.ErrChargeInvalidAmount) {
		t.Errorf("invalid charge error = %v, want %v", err, domain.ErrChargeInvalidAmount)
	}
	if repository.RegisterPaymentCalls != calls {
		t.Error("an invalid charge should not reach the repository")
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
	users.FindByAuthProviderIDResult = domain.Usuario{ID: "surveyor-id", Rol: domain.RolAgrimensor}
	if _, err := useCase.Execute(context.Background(), valid); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Errorf("surveyor row error = %v, want %v", err, domain.ErrNoAutorizado)
	}
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
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAgencyUser()}
	repository := &gatewayfake.CollectionRepository{RegisterPaymentResult: domain.Payment{ID: "cobro-9", Type: domain.PaymentTypeSettlement}}
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
	if command.SaleID != "sale-1" || command.ActorID != "agency-user-id" || command.Type != domain.PaymentTypeSettlement || command.ExpectedAmount != expected {
		t.Errorf("command = %#v", command)
	}
	if command.Medium != domain.PaymentMediumCheque || command.Observation != "Cheque 44" || !command.PaidAt.Equal(now) || len(command.InstallmentIDs) != 0 || command.IncludeDownPayment {
		t.Errorf("terms = %#v", command)
	}
	scope := repository.RegisterPaymentScope
	if scope.AssigneeAuthProviderID == nil || *scope.AssigneeAuthProviderID != "agency-subject" || !scope.ByAgency {
		t.Errorf("agency scope = %#v", scope)
	}

	users.FindByAuthProviderIDResult = activeAdmin()
	valid := collections.SettleSaleInput{Actor: adminActor(), SaleID: "sale-1", ExpectedAmount: &expected, Medium: "efectivo"}
	rejections := []struct {
		name  string
		input collections.SettleSaleInput
		want  error
	}{
		{"blank sale", collections.SettleSaleInput{Actor: adminActor(), ExpectedAmount: &expected, Medium: "efectivo"}, domain.ErrSaleNotFound},
		{"no expected amount", collections.SettleSaleInput{Actor: adminActor(), SaleID: "sale-1", Medium: "efectivo"}, domain.ErrSettlementExpectedAmountRequired},
		{"blank medium", collections.SettleSaleInput{Actor: adminActor(), SaleID: "sale-1", ExpectedAmount: &expected}, domain.ErrPaymentInvalidMedium},
	}
	for _, tc := range rejections {
		t.Run(tc.name, func(t *testing.T) {
			calls := repository.RegisterPaymentCalls
			if _, err := useCase.Execute(context.Background(), tc.input); !errors.Is(err, tc.want) {
				t.Errorf("Execute() error = %v, want %v", err, tc.want)
			}
			if repository.RegisterPaymentCalls != calls {
				t.Error("a rejected settlement should not reach the repository")
			}
		})
	}

	users.FindByAuthProviderIDResult = domain.Usuario{ID: "actor-id", Rol: domain.RolAgrimensor}
	if _, err := useCase.Execute(context.Background(), valid); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Errorf("role mismatch error = %v", err)
	}
	users.FindByAuthProviderIDResult = activeAdmin()
	users.FindByAuthProviderIDErr = errors.New("db down")
	if _, err := useCase.Execute(context.Background(), valid); !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Errorf("user lookup failure error = %v", err)
	}
	users.FindByAuthProviderIDErr = nil
	repository.RegisterPaymentErr = domain.ErrSettlementAmountMismatch
	if _, err := useCase.Execute(context.Background(), valid); !errors.Is(err, domain.ErrSettlementAmountMismatch) {
		t.Errorf("mismatch error = %v", err)
	}
}

// An administrador demoted to inmobiliaria keeps a token that still says
// administrador until it expires. Every use case scopes by the current row.
func TestCollectionsScopeADemotedUserByTheirCurrentRole(t *testing.T) {
	demoted := activeAgencyUser()
	demoted.AuthProviderID = "actor-subject"
	staleToken := adminActor()
	expected := 1000.0
	wantAgencyScope := func(t *testing.T, scope gateway.SaleScope) {
		t.Helper()
		if scope.AssigneeAuthProviderID == nil || *scope.AssigneeAuthProviderID != "actor-subject" || !scope.ByAgency {
			t.Errorf("scope = %#v, want the agency scope of the current rol", scope)
		}
	}

	t.Run("debt statement", func(t *testing.T) {
		repository := &gatewayfake.CollectionRepository{}
		users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: demoted}
		if _, err := collections.NewGetDebtStatement(repository, users, fixedClock{now: now}).Execute(context.Background(), staleToken, "sale-1"); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		wantAgencyScope(t, repository.GetDebtStatementScope)
	})
	t.Run("due installments", func(t *testing.T) {
		repository := &gatewayfake.CollectionRepository{}
		users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: demoted}
		if _, err := collections.NewListDueInstallments(repository, users, fixedClock{now: now}).Execute(context.Background(), collections.ListDueInstallmentsInput{Actor: staleToken}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		wantAgencyScope(t, repository.ListDueInstallmentsScope)
	})
	t.Run("register payment", func(t *testing.T) {
		repository := &gatewayfake.CollectionRepository{}
		users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: demoted}
		if _, err := collections.NewRegisterPayment(repository, users, fixedClock{now: now}).Execute(context.Background(), collections.RegisterPaymentInput{
			Actor: staleToken, SaleID: "sale-1", InstallmentIDs: []string{"c-1"}, Medium: "efectivo",
		}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		wantAgencyScope(t, repository.RegisterPaymentScope)
	})
	t.Run("settle sale", func(t *testing.T) {
		repository := &gatewayfake.CollectionRepository{}
		users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: demoted}
		if _, err := collections.NewSettleSale(repository, users, fixedClock{now: now}).Execute(context.Background(), collections.SettleSaleInput{
			Actor: staleToken, SaleID: "sale-1", ExpectedAmount: &expected, Medium: "efectivo",
		}); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		wantAgencyScope(t, repository.RegisterPaymentScope)
	})
	t.Run("reads reject an unprovisioned or inactive caller", func(t *testing.T) {
		users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado}
		useCase := collections.NewGetDebtStatement(&gatewayfake.CollectionRepository{}, users, fixedClock{now: now})
		if _, err := useCase.Execute(context.Background(), staleToken, "sale-1"); !errors.Is(err, domain.ErrActorNoAprovisionado) {
			t.Errorf("unprovisioned error = %v, want %v", err, domain.ErrActorNoAprovisionado)
		}
		deactivated := now
		users.FindByAuthProviderIDErr = nil
		users.FindByAuthProviderIDResult = domain.Usuario{ID: "actor-id", Rol: domain.RolAdministrador, FechaBaja: &deactivated}
		if _, err := useCase.Execute(context.Background(), staleToken, "sale-1"); !errors.Is(err, domain.ErrCuentaInactiva) {
			t.Errorf("inactive error = %v, want %v", err, domain.ErrCuentaInactiva)
		}
	})
}

func TestSystemClockAndDefaultScope(t *testing.T) {
	before := time.Now()
	got := collections.SystemClock{}.Now()
	if got.Before(before) || got.After(time.Now()) {
		t.Errorf("SystemClock.Now() = %s, want between %s and now", got, before)
	}
	repository := &gatewayfake.CollectionRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	useCase := collections.NewGetDebtStatement(repository, users, nil)
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
