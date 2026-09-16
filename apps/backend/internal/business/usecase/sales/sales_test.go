package sales_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/sales"
)

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

func activeAdmin() domain.Usuario {
	return domain.Usuario{ID: "actor-id", AuthProviderID: "actor-subject", Rol: domain.RolAdministrador}
}

func adminActor() sales.Actor {
	return sales.Actor{AuthProviderID: "actor-subject", Roles: []string{domain.RolAdministrador}}
}

func validInput() sales.CreateSaleInput {
	return sales.CreateSaleInput{
		Actor:          adminActor(),
		DevelopmentID:  " loteo-id ",
		LotID:          " lote-id ",
		ClientID:       " cliente-id ",
		SellerID:       " seller-id ",
		IdempotencyKey: " sale-key ",
	}
}

func sha256Hex(payload string) string {
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func TestCreateSaleNormalizesAndDefaultsToContado(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.SaleRepository{CreateResult: domain.Sale{ID: "sale-1"}}
	created := time.Date(2026, 9, 14, 15, 0, 0, 0, time.FixedZone("ART", -3*60*60))
	useCase := sales.NewCreateSale(repository, users, fixedClock{now: created})

	sale, err := useCase.Execute(context.Background(), validInput())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if sale.ID != "sale-1" {
		t.Errorf("Execute() = %#v, want the repository result", sale)
	}
	command := repository.CreateCommand
	if command.DevelopmentID != "loteo-id" || command.LotID != "lote-id" || command.ClientID != "cliente-id" || command.SellerID != "seller-id" {
		t.Errorf("normalized command = %#v", command)
	}
	if command.ActorID != "actor-id" || command.PaymentMethod != domain.PaymentMethodCash {
		t.Errorf("command actor/method = %q/%q", command.ActorID, command.PaymentMethod)
	}
	if !command.CreatedAt.Equal(created.UTC()) {
		t.Errorf("created at = %s, want %s", command.CreatedAt, created.UTC())
	}
	if command.IdempotencyKey != "sale-key" {
		t.Errorf("idempotency key = %q, want the trimmed header", command.IdempotencyKey)
	}
	wantHash := sha256Hex("8:loteo-id7:lote-id10:cliente-id9:seller-id7:contado")
	if command.IdempotencyPayloadHash != wantHash {
		t.Errorf("payload hash = %q, want %q", command.IdempotencyPayloadHash, wantHash)
	}
}

func TestCreateSalePayloadHashChangesWithTheSale(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.SaleRepository{}
	useCase := sales.NewCreateSale(repository, users)

	if _, err := useCase.Execute(context.Background(), validInput()); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	first := repository.CreateCommand.IdempotencyPayloadHash

	other := validInput()
	other.ClientID = "another-client"
	if _, err := useCase.Execute(context.Background(), other); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.CreateCommand.IdempotencyPayloadHash == first {
		t.Error("a different cliente must produce a different payload hash")
	}

	financed := validInput()
	financed.PaymentMethod = string(domain.PaymentMethodFinanced)
	financed.PaymentPlan = &sales.PaymentPlanInput{Installments: 12, InterestRate: 10, Period: "mensual"}
	if _, err := useCase.Execute(context.Background(), financed); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	financedHash := repository.CreateCommand.IdempotencyPayloadHash
	if financedHash == first {
		t.Error("a financed sale must produce a different payload hash than contado")
	}
	if financedHash != sha256Hex("8:loteo-id7:lote-id10:cliente-id9:seller-id10:financiado|12:10:mensual:0") {
		t.Errorf("financed payload hash = %q", financedHash)
	}

	otherPlan := financed
	otherPlan.PaymentPlan = &sales.PaymentPlanInput{Installments: 24, InterestRate: 10, Period: "mensual"}
	if _, err := useCase.Execute(context.Background(), otherPlan); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.CreateCommand.IdempotencyPayloadHash == financedHash {
		t.Error("a different plan must produce a different payload hash")
	}
}

func TestCreateSalePassesTheNormalizedPlanToTheRepository(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.SaleRepository{}
	useCase := sales.NewCreateSale(repository, users, fixedClock{})

	input := validInput()
	input.PaymentMethod = string(domain.PaymentMethodDownAndFi)
	input.PaymentPlan = &sales.PaymentPlanInput{Installments: 24, InterestRate: 15.5, Period: " mensual ", DownPayment: 20000}
	if _, err := useCase.Execute(context.Background(), input); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	command := repository.CreateCommand
	if command.PaymentMethod != domain.PaymentMethodDownAndFi || command.PaymentPlan == nil {
		t.Fatalf("command = %#v, want a plan", command)
	}
	want := domain.PaymentPlanInput{Installments: 24, InterestRate: 15.5, Period: domain.PaymentPeriodMonthly, DownPayment: 20000}
	if *command.PaymentPlan != want {
		t.Errorf("plan = %#v, want %#v", *command.PaymentPlan, want)
	}

	// The repository knows the lote price, so a down payment above it comes
	// back from there as the same domain error.
	repository = &gatewayfake.SaleRepository{CreateErr: domain.ErrSaleInvalidDownPayment}
	useCase = sales.NewCreateSale(repository, users, fixedClock{})
	if _, err := useCase.Execute(context.Background(), input); !errors.Is(err, domain.ErrSaleInvalidDownPayment) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrSaleInvalidDownPayment)
	}
}

func TestCreateSaleLetsAnAgencyUserPickAColleague(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{
		ID: "agency-user", AuthProviderID: "agency-subject", Rol: domain.RolInmobiliaria,
	}}
	repository := &gatewayfake.SaleRepository{}
	useCase := sales.NewCreateSale(repository, users, fixedClock{})

	input := validInput()
	input.Actor = sales.Actor{AuthProviderID: "agency-subject", Roles: []string{domain.RolInmobiliaria}}
	input.SellerID = "colleague"
	if _, err := useCase.Execute(context.Background(), input); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.CreateCommand.SellerID != "colleague" || repository.CreateCommand.ActorID != "agency-user" {
		t.Errorf("seller command = %#v", repository.CreateCommand)
	}
}

func TestCreateSaleValidatesInput(t *testing.T) {
	tests := []struct {
		name  string
		input func() sales.CreateSaleInput
		want  error
	}{
		{"unauthorized role", func() sales.CreateSaleInput {
			input := validInput()
			input.Actor.Roles = []string{domain.RolEscribano}
			return input
		}, domain.ErrNoAutorizado},
		{"missing idempotency key", func() sales.CreateSaleInput {
			input := validInput()
			input.IdempotencyKey = "  "
			return input
		}, domain.ErrSaleIdempotencyRequired},
		{"idempotency key too long", func() sales.CreateSaleInput {
			input := validInput()
			input.IdempotencyKey = strings.Repeat("k", domain.MaxIdempotencyKeySize+1)
			return input
		}, domain.ErrSaleIdempotencyRequired},
		{"missing lote", func() sales.CreateSaleInput {
			input := validInput()
			input.LotID = " "
			return input
		}, domain.ErrLoteNotFound},
		{"missing client", func() sales.CreateSaleInput {
			input := validInput()
			input.ClientID = ""
			return input
		}, domain.ErrSaleInvalidClient},
		{"missing seller", func() sales.CreateSaleInput {
			input := validInput()
			input.SellerID = ""
			return input
		}, domain.ErrSaleSellerRequired},
		{"invalid payment method", func() sales.CreateSaleInput {
			input := validInput()
			input.PaymentMethod = "cripto"
			return input
		}, domain.ErrSaleInvalidPaymentMethod},
		{"financed without plan", func() sales.CreateSaleInput {
			input := validInput()
			input.PaymentMethod = string(domain.PaymentMethodFinanced)
			return input
		}, domain.ErrSalePaymentPlanRequired},
		{"contado with plan", func() sales.CreateSaleInput {
			input := validInput()
			input.PaymentPlan = &sales.PaymentPlanInput{Installments: 12, Period: "mensual"}
			return input
		}, domain.ErrSalePaymentPlanNotApplicable},
		{"invalid installments", func() sales.CreateSaleInput {
			input := validInput()
			input.PaymentMethod = string(domain.PaymentMethodFinanced)
			input.PaymentPlan = &sales.PaymentPlanInput{Installments: 0, Period: "mensual"}
			return input
		}, domain.ErrSaleInvalidInstallments},
		{"invalid rate", func() sales.CreateSaleInput {
			input := validInput()
			input.PaymentMethod = string(domain.PaymentMethodFinanced)
			input.PaymentPlan = &sales.PaymentPlanInput{Installments: 12, InterestRate: -1, Period: "mensual"}
			return input
		}, domain.ErrSaleInvalidInterestRate},
		{"invalid periodicity", func() sales.CreateSaleInput {
			input := validInput()
			input.PaymentMethod = string(domain.PaymentMethodFinanced)
			input.PaymentPlan = &sales.PaymentPlanInput{Installments: 12, Period: "semanal"}
			return input
		}, domain.ErrSaleInvalidPeriodicity},
		{"down payment on financiado", func() sales.CreateSaleInput {
			input := validInput()
			input.PaymentMethod = string(domain.PaymentMethodFinanced)
			input.PaymentPlan = &sales.PaymentPlanInput{Installments: 12, Period: "mensual", DownPayment: 10}
			return input
		}, domain.ErrSaleDownPaymentNotApplicable},
		{"entrega without down payment", func() sales.CreateSaleInput {
			input := validInput()
			input.PaymentMethod = string(domain.PaymentMethodDownAndFi)
			input.PaymentPlan = &sales.PaymentPlanInput{Installments: 12, Period: "mensual"}
			return input
		}, domain.ErrSaleInvalidDownPayment},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
			repository := &gatewayfake.SaleRepository{}
			useCase := sales.NewCreateSale(repository, users)

			_, err := useCase.Execute(context.Background(), test.input())
			if !errors.Is(err, test.want) {
				t.Fatalf("Execute() error = %v, want %v", err, test.want)
			}
			if repository.CreateCalls != 0 {
				t.Error("Execute() should not reach the repository")
			}
		})
	}
}

func TestCreateSaleResolvesTheActor(t *testing.T) {
	t.Run("unknown actor", func(t *testing.T) {
		users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado}
		useCase := sales.NewCreateSale(&gatewayfake.SaleRepository{}, users)
		if _, err := useCase.Execute(context.Background(), validInput()); !errors.Is(err, domain.ErrActorNoAprovisionado) {
			t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
		}
	})
	t.Run("inactive actor", func(t *testing.T) {
		inactive := activeAdmin()
		now := time.Now()
		inactive.FechaBaja = &now
		users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: inactive}
		useCase := sales.NewCreateSale(&gatewayfake.SaleRepository{}, users)
		if _, err := useCase.Execute(context.Background(), validInput()); !errors.Is(err, domain.ErrCuentaInactiva) {
			t.Fatalf("Execute() error = %v, want %v", err, domain.ErrCuentaInactiva)
		}
	})
	t.Run("token role that the local profile does not have", func(t *testing.T) {
		users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "u", Rol: domain.RolEscribano}}
		useCase := sales.NewCreateSale(&gatewayfake.SaleRepository{}, users)
		if _, err := useCase.Execute(context.Background(), validInput()); !errors.Is(err, domain.ErrNoAutorizado) {
			t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
		}
	})
	t.Run("lookup failure", func(t *testing.T) {
		cause := errors.New("connection reset")
		users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: cause}
		useCase := sales.NewCreateSale(&gatewayfake.SaleRepository{}, users)
		_, err := useCase.Execute(context.Background(), validInput())
		if !errors.Is(err, domain.ErrDatabaseUnavailable) || !errors.Is(err, cause) {
			t.Fatalf("Execute() error = %v, want unavailable wrapping %v", err, cause)
		}
	})
}

func TestCreateSalePropagatesRepositoryErrors(t *testing.T) {
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: activeAdmin()}
	repository := &gatewayfake.SaleRepository{CreateErr: domain.ErrSaleLotUnavailable}
	useCase := sales.NewCreateSale(repository, users)
	if _, err := useCase.Execute(context.Background(), validInput()); !errors.Is(err, domain.ErrSaleLotUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrSaleLotUnavailable)
	}

	cause := errors.New("insert failed")
	repository = &gatewayfake.SaleRepository{CreateErr: cause}
	useCase = sales.NewCreateSale(repository, users)
	_, err := useCase.Execute(context.Background(), validInput())
	if !errors.Is(err, domain.ErrDatabaseUnavailable) || !errors.Is(err, cause) {
		t.Fatalf("Execute() error = %v, want unavailable wrapping %v", err, cause)
	}
}

func TestListSalesAppliesScopeAndNormalizesTheFilter(t *testing.T) {
	repository := &gatewayfake.SaleRepository{ListResult: domain.SalePage{Total: 1}}
	useCase := sales.NewListSales(repository)

	page, err := useCase.Execute(context.Background(), sales.ListSalesInput{
		Actor:  sales.Actor{AuthProviderID: "agency-subject", Roles: []string{domain.RolInmobiliaria}},
		Filter: domain.SaleListFilter{Search: " ana "},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if page.Total != 1 {
		t.Errorf("Execute() = %#v, want the repository page", page)
	}
	if repository.ListFilter.Search != "ana" || repository.ListFilter.Page != domain.DefaultSalePage || repository.ListFilter.Limit != domain.DefaultSaleLimit {
		t.Errorf("filter = %#v", repository.ListFilter)
	}
	if !repository.ListScope.ByAgency || repository.ListScope.AssigneeAuthProviderID == nil || *repository.ListScope.AssigneeAuthProviderID != "agency-subject" {
		t.Errorf("agency scope = %#v", repository.ListScope)
	}

	if _, err := useCase.Execute(context.Background(), sales.ListSalesInput{Actor: adminActor()}); err != nil {
		t.Fatalf("Execute() admin error = %v", err)
	}
	if repository.ListScope.ByAgency || repository.ListScope.AssigneeAuthProviderID != nil {
		t.Errorf("admin scope = %#v, want unrestricted", repository.ListScope)
	}

	if _, err := useCase.Execute(context.Background(), sales.ListSalesInput{
		Actor: sales.Actor{AuthProviderID: "s", Roles: []string{domain.RolAgrimensor}},
	}); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if _, err := useCase.Execute(context.Background(), sales.ListSalesInput{
		Actor: adminActor(), Filter: domain.SaleListFilter{Page: -1},
	}); !errors.Is(err, domain.ErrSaleInvalidPage) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrSaleInvalidPage)
	}

	cause := errors.New("query failed")
	repository.ListErr = cause
	_, err = useCase.Execute(context.Background(), sales.ListSalesInput{Actor: adminActor()})
	if !errors.Is(err, domain.ErrDatabaseUnavailable) || !errors.Is(err, cause) {
		t.Fatalf("Execute() error = %v, want unavailable wrapping %v", err, cause)
	}
}

func TestGetSaleAppliesScope(t *testing.T) {
	repository := &gatewayfake.SaleRepository{GetResult: domain.Sale{ID: "sale-1"}}
	useCase := sales.NewGetSale(repository)

	sale, err := useCase.Execute(context.Background(), adminActor(), " sale-1 ")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if sale.ID != "sale-1" || repository.GetID != "sale-1" {
		t.Errorf("Execute() = %#v, id passed = %q", sale, repository.GetID)
	}

	if _, err := useCase.Execute(context.Background(), adminActor(), "  "); !errors.Is(err, domain.ErrSaleNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrSaleNotFound)
	}
	if _, err := useCase.Execute(context.Background(), sales.Actor{Roles: []string{domain.RolEscribano}}, "sale-1"); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}

	repository.GetErr = domain.ErrSaleNotFound
	if _, err := useCase.Execute(context.Background(), adminActor(), "missing"); !errors.Is(err, domain.ErrSaleNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrSaleNotFound)
	}
}
