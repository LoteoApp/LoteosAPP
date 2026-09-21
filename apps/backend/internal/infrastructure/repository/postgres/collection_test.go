package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

func collectionPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// financedSaleFixture registers an entrega + financiación sale of 3 quarterly
// cuotas of 20000 over a 100000 lote with a 40000 entrega, dated so the
// first cuota is already vencida when the tests run.
func financedSaleFixture(t *testing.T, pool *pgxpool.Pool, saleDate time.Time) (sale domain.Sale, actorID string) {
	t.Helper()
	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	sale, err := postgres.NewSaleRepository(pool).Create(context.Background(), gateway.CreateSaleCommand{
		DevelopmentID: loteoID, LotID: lotID, ClientID: clientID, SellerID: actorID, ActorID: actorID,
		IdempotencyKey: newUUID(t), IdempotencyPayloadHash: saleHash(t),
		PaymentMethod: domain.PaymentMethodDownAndFi,
		PaymentPlan:   &domain.PaymentPlanInput{Installments: 3, InterestRate: 0, Period: domain.PaymentPeriodQuarterly, DownPayment: 40000},
		CreatedAt:     saleDate,
	})
	if err != nil {
		t.Fatalf("create financed sale: %v", err)
	}
	return sale, actorID
}

func lotStateOf(t *testing.T, pool *pgxpool.Pool, lotID string) string {
	t.Helper()
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT estado_actual FROM lotes WHERE id = $1::uuid`, lotID).Scan(&state); err != nil {
		t.Fatalf("read lot state: %v", err)
	}
	return state
}

func TestCollectionRepositoryStatementPaymentsAndCompletion(t *testing.T) {
	pool := collectionPool(t)
	saleDate := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	sale, actorID := financedSaleFixture(t, pool, saleDate)
	repository := postgres.NewCollectionRepository(pool)

	statement, err := repository.GetDebtStatement(context.Background(), sale.ID, gateway.SaleScope{}, now)
	if err != nil {
		t.Fatalf("GetDebtStatement() error = %v", err)
	}
	if statement.Venta.ID != sale.ID || statement.Venta.PlanPago == nil || !statement.EmitidoEl.Equal(now) {
		t.Fatalf("statement = %#v", statement)
	}
	if statement.Entrega == nil || statement.Entrega.Monto != 40000 || statement.Entrega.Estado != domain.InstallmentStatePending {
		t.Errorf("entrega = %#v", statement.Entrega)
	}
	if len(statement.Cuotas) != 3 || statement.Cuotas[0].Estado != domain.InstallmentStateOverdue || statement.Cuotas[1].Estado != domain.InstallmentStatePending {
		t.Errorf("cuotas = %#v", statement.Cuotas)
	}
	if statement.Resumen.MontoTotal != 100000 || statement.Resumen.MontoPendiente != 100000 || statement.Resumen.MontoVencido != 20000 || statement.Resumen.CuotasVencidas != 1 {
		t.Errorf("resumen = %#v", statement.Resumen)
	}
	if len(statement.Cobros) != 0 {
		t.Errorf("cobros = %#v, want none yet", statement.Cobros)
	}

	// Paying cuota 2 while 1 is owed is rejected, as is a cuota of another plan.
	scope := gateway.SaleScope{}
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular, InstallmentIDs: []string{statement.Cuotas[1].ID},
		Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, scope); !errors.Is(err, domain.ErrPaymentInstallmentOrder) {
		t.Fatalf("out-of-order payment error = %v, want %v", err, domain.ErrPaymentInstallmentOrder)
	}
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular, InstallmentIDs: []string{newUUID(t)},
		Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, scope); !errors.Is(err, domain.ErrPaymentInstallmentUnknown) {
		t.Fatalf("unknown cuota error = %v, want %v", err, domain.ErrPaymentInstallmentUnknown)
	}

	paidAt := now.Add(-time.Hour)
	first, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular,
		InstallmentIDs: []string{statement.Cuotas[0].ID}, IncludeDownPayment: true,
		Medium: domain.PaymentMediumTransfer, Observation: "Transferencia 123", PaidAt: paidAt, Now: now,
	}, scope)
	if err != nil {
		t.Fatalf("RegisterPayment() error = %v", err)
	}
	if first.ID == "" || first.VentaID != sale.ID || first.Tipo != domain.PaymentTypeRegular || first.Monto != 60000 || first.Moneda != "USD" {
		t.Errorf("first payment = %#v", first)
	}
	if first.MedioPago != domain.PaymentMediumTransfer || first.Observacion != "Transferencia 123" || !first.FechaPago.Equal(paidAt) || first.UsuarioAlta.ID != actorID {
		t.Errorf("first payment terms = %#v", first)
	}
	if !first.IncluyeEntrega || first.MontoEntrega != 40000 || len(first.Cuotas) != 1 || first.Cuotas[0].Numero != 1 || first.Cuotas[0].Estado != domain.InstallmentStatePaid {
		t.Errorf("first payment items = %#v", first)
	}
	if got := lotStateOf(t, pool, sale.LoteID); got != string(domain.LotStateSold) {
		t.Errorf("lot state after a partial payment = %q, want vendido", got)
	}

	afterFirst, err := repository.GetDebtStatement(context.Background(), sale.ID, scope, now)
	if err != nil {
		t.Fatalf("GetDebtStatement() after payment error = %v", err)
	}
	if afterFirst.Entrega.Estado != domain.InstallmentStatePaid || afterFirst.Entrega.CobroID != first.ID || afterFirst.Entrega.FechaPago == nil {
		t.Errorf("entrega after payment = %#v", afterFirst.Entrega)
	}
	if afterFirst.Cuotas[0].Estado != domain.InstallmentStatePaid || afterFirst.Cuotas[0].CobroID != first.ID || afterFirst.Cuotas[0].FechaPago == nil {
		t.Errorf("cuota 1 after payment = %#v", afterFirst.Cuotas[0])
	}
	if afterFirst.Resumen.MontoPagado != 60000 || afterFirst.Resumen.MontoPendiente != 40000 || afterFirst.Resumen.CuotasVencidas != 0 || afterFirst.Resumen.CuotasPagadas != 1 {
		t.Errorf("resumen after payment = %#v", afterFirst.Resumen)
	}
	if len(afterFirst.Cobros) != 1 || afterFirst.Cobros[0].ID != first.ID || len(afterFirst.Cobros[0].Cuotas) != 1 || !afterFirst.Cobros[0].IncluyeEntrega {
		t.Errorf("cobros after payment = %#v", afterFirst.Cobros)
	}

	// Paying the same cuota or the entrega again is a conflict.
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular, InstallmentIDs: []string{statement.Cuotas[0].ID},
		Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, scope); !errors.Is(err, domain.ErrPaymentInstallmentPaid) {
		t.Fatalf("repeated cuota error = %v, want %v", err, domain.ErrPaymentInstallmentPaid)
	}
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular, IncludeDownPayment: true,
		Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, scope); !errors.Is(err, domain.ErrPaymentDownPaymentPaid) {
		t.Fatalf("repeated entrega error = %v, want %v", err, domain.ErrPaymentDownPaymentPaid)
	}

	// A settlement over a stale balance is rejected; the right one closes
	// the venta and finalizes the lote with origin cobranza.
	stale := 60000.0
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeSettlement, ExpectedAmount: &stale,
		Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, scope); !errors.Is(err, domain.ErrSettlementAmountMismatch) {
		t.Fatalf("stale settlement error = %v, want %v", err, domain.ErrSettlementAmountMismatch)
	}
	expected := 40000.0
	settlement, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeSettlement, ExpectedAmount: &expected,
		Medium: domain.PaymentMediumCheque, PaidAt: now, Now: now,
	}, scope)
	if err != nil {
		t.Fatalf("settlement error = %v", err)
	}
	if settlement.Tipo != domain.PaymentTypeSettlement || settlement.Monto != 40000 || settlement.IncluyeEntrega || len(settlement.Cuotas) != 2 {
		t.Errorf("settlement = %#v", settlement)
	}
	if got := lotStateOf(t, pool, sale.LoteID); got != string(domain.LotStateCompleted) {
		t.Errorf("lot state after settlement = %q, want finalizado", got)
	}
	closed, err := postgres.NewSaleRepository(pool).Get(context.Background(), sale.ID, scope)
	if err != nil {
		t.Fatalf("Get() closed sale error = %v", err)
	}
	if closed.Estado != domain.SaleStateCompleted || len(closed.Historial) != 2 || closed.Historial[1].Razon != domain.SaleSettledBySettlementReason {
		t.Errorf("closed sale = %q with history %#v", closed.Estado, closed.Historial)
	}
	var origin string
	if err := pool.QueryRow(context.Background(), `
		SELECT origen FROM lote_estados WHERE lote_id = $1::uuid AND estado = 'finalizado'
	`, sale.LoteID).Scan(&origin); err != nil {
		t.Fatalf("read finalizado event: %v", err)
	}
	if origin != string(domain.LotStateOriginCollection) {
		t.Errorf("finalizado origin = %q, want cobranza", origin)
	}

	final, err := repository.GetDebtStatement(context.Background(), sale.ID, scope, now)
	if err != nil {
		t.Fatalf("GetDebtStatement() final error = %v", err)
	}
	if !final.Saldada() || final.Resumen.MontoPendiente != 0 || len(final.Cobros) != 2 || final.Cobros[0].ID != settlement.ID {
		t.Errorf("final statement = %#v", final)
	}
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeSettlement,
		Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, scope); !errors.Is(err, domain.ErrSaleNotActive) {
		t.Fatalf("payment on a completed sale error = %v, want %v", err, domain.ErrSaleNotActive)
	}
}

func TestCollectionRepositoryCompletesWhenTheLastCuotaIsPaid(t *testing.T) {
	pool := collectionPool(t)
	saleDate := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	sale, err := postgres.NewSaleRepository(pool).Create(context.Background(), gateway.CreateSaleCommand{
		DevelopmentID: loteoID, LotID: lotID, ClientID: clientID, SellerID: actorID, ActorID: actorID,
		IdempotencyKey: newUUID(t), IdempotencyPayloadHash: saleHash(t),
		PaymentMethod: domain.PaymentMethodFinanced,
		PaymentPlan:   &domain.PaymentPlanInput{Installments: 2, InterestRate: 10, Period: domain.PaymentPeriodMonthly},
		CreatedAt:     saleDate,
	})
	if err != nil {
		t.Fatalf("create financed sale: %v", err)
	}
	repository := postgres.NewCollectionRepository(pool)
	statement, err := repository.GetDebtStatement(context.Background(), sale.ID, gateway.SaleScope{}, now)
	if err != nil {
		t.Fatalf("GetDebtStatement() error = %v", err)
	}
	if statement.Entrega != nil || len(statement.Cuotas) != 2 || statement.Resumen.MontoTotal != 110000 {
		t.Fatalf("statement = %#v", statement)
	}
	// Both cuotas in one regular cobro: nothing is left, so the venta closes
	// with the "plan completado" reason rather than the settlement one.
	payment, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular,
		InstallmentIDs: []string{statement.Cuotas[1].ID, statement.Cuotas[0].ID},
		Medium:         domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("RegisterPayment() error = %v", err)
	}
	if payment.Monto != 110000 || len(payment.Cuotas) != 2 || payment.Cuotas[0].Numero != 1 || payment.IncluyeEntrega {
		t.Errorf("payment = %#v", payment)
	}
	closed, err := postgres.NewSaleRepository(pool).Get(context.Background(), sale.ID, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if closed.Estado != domain.SaleStateCompleted || len(closed.Historial) != 2 || closed.Historial[1].Razon != domain.SaleSettledByPaymentReason {
		t.Errorf("closed sale = %q with history %#v", closed.Estado, closed.Historial)
	}
	if got := lotStateOf(t, pool, lotID); got != string(domain.LotStateCompleted) {
		t.Errorf("lot state = %q, want finalizado", got)
	}
}

func TestCollectionRepositoryListsDueInstallmentsWithinScope(t *testing.T) {
	pool := collectionPool(t)
	saleDate := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	sale, actorID := financedSaleFixture(t, pool, saleDate)
	repository := postgres.NewCollectionRepository(pool)
	scope := gateway.SaleScope{}

	// Cuotas fall due on Apr 15, Jul 15 and Oct 15: one vencida, one within
	// the next 30 days, one further out.
	page, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() error = %v", err)
	}
	if page.Total != 3 || len(page.Items) != 3 || page.TotalPages != 1 {
		t.Fatalf("page = %#v", page)
	}
	if page.Items[0].Estado != domain.InstallmentStateOverdue || page.Items[1].Estado != domain.InstallmentStatePending || page.Items[0].Numero != 1 {
		t.Errorf("items = %#v", page.Items)
	}
	item := page.Items[0]
	if item.VentaID != sale.ID || item.CantidadCuotas != 3 || item.Monto != 20000 || item.Moneda != "USD" || item.LoteoNombre == "" || item.LoteNumero != "1" || item.ManzanaNumero != "1" {
		t.Errorf("item = %#v", item)
	}
	if item.Cliente.ID != sale.Cliente.ID || item.Vendedor.ID != actorID || item.Inmobiliaria != nil {
		t.Errorf("item parties = %#v", item)
	}
	if page.Resumen.CuotasVencidas != 1 || page.Resumen.CuotasProximas != 1 {
		t.Errorf("resumen = %#v", page.Resumen)
	}

	overdue, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, States: []domain.InstallmentState{domain.InstallmentStateOverdue}}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() overdue error = %v", err)
	}
	if overdue.Total != 1 || overdue.Items[0].Numero != 1 {
		t.Errorf("overdue = %#v", overdue)
	}
	from, to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	july, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, From: &from, To: &to}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() july error = %v", err)
	}
	if july.Total != 1 || july.Items[0].Numero != 2 {
		t.Errorf("july = %#v", july)
	}
	byClient, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, Search: "Cliente"}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() by client error = %v", err)
	}
	if byClient.Total != 3 {
		t.Errorf("by client = %#v", byClient)
	}
	noMatch, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, Search: "nadie"}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() no match error = %v", err)
	}
	if noMatch.Total != 0 || len(noMatch.Items) != 0 || noMatch.Resumen.CuotasVencidas != 1 {
		t.Errorf("no match = %#v, want empty page with the scope summary", noMatch)
	}
	paid, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, States: []domain.InstallmentState{domain.InstallmentStatePaid}}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() paid error = %v", err)
	}
	if paid.Total != 0 {
		t.Errorf("paid = %#v, want none", paid)
	}
	outside, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, Page: 5}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() out of range error = %v", err)
	}
	if outside.Total != 3 || len(outside.Items) != 0 || outside.TotalPages != 1 {
		t.Errorf("out of range = %#v", outside)
	}

	strangerAuthID := newUUID(t)
	agencyScope := gateway.SaleScope{AssigneeAuthProviderID: &strangerAuthID, ByAgency: true}
	scoped, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID}, agencyScope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() outside scope error = %v", err)
	}
	if scoped.Total != 0 || scoped.Resumen.CuotasVencidas != 0 {
		t.Errorf("outside scope = %#v, want none", scoped)
	}
	if _, err := repository.GetDebtStatement(context.Background(), sale.ID, agencyScope, now); !errors.Is(err, domain.ErrSaleNotFound) {
		t.Errorf("GetDebtStatement() outside scope error = %v, want %v", err, domain.ErrSaleNotFound)
	}
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeSettlement, Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, agencyScope); !errors.Is(err, domain.ErrSaleNotFound) {
		t.Errorf("RegisterPayment() outside scope error = %v, want %v", err, domain.ErrSaleNotFound)
	}
}

func TestCollectionRepositoryRejectsMissingAndCashSales(t *testing.T) {
	pool := collectionPool(t)
	now := time.Now().UTC()
	repository := postgres.NewCollectionRepository(pool)
	scope := gateway.SaleScope{}

	for _, id := range []string{newUUID(t), "not-a-uuid"} {
		if _, err := repository.GetDebtStatement(context.Background(), id, scope, now); !errors.Is(err, domain.ErrSaleNotFound) {
			t.Errorf("GetDebtStatement(%q) error = %v, want %v", id, err, domain.ErrSaleNotFound)
		}
		if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
			SaleID: id, Type: domain.PaymentTypeSettlement, Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
		}, scope); !errors.Is(err, domain.ErrSaleNotFound) {
			t.Errorf("RegisterPayment(%q) error = %v, want %v", id, err, domain.ErrSaleNotFound)
		}
	}

	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	cash, err := postgres.NewSaleRepository(pool).Create(context.Background(), gateway.CreateSaleCommand{
		DevelopmentID: loteoID, LotID: lotID, ClientID: clientID, SellerID: actorID, ActorID: actorID,
		IdempotencyKey: newUUID(t), IdempotencyPayloadHash: saleHash(t),
		PaymentMethod: domain.PaymentMethodCash, CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("create cash sale: %v", err)
	}
	if _, err := repository.GetDebtStatement(context.Background(), cash.ID, scope, now); !errors.Is(err, domain.ErrSaleNotFinanced) {
		t.Errorf("GetDebtStatement() cash error = %v, want %v", err, domain.ErrSaleNotFinanced)
	}
	// A contado sale is completada on registration, so it isn't collectable.
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: cash.ID, ActorID: actorID, Type: domain.PaymentTypeSettlement, Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, scope); !errors.Is(err, domain.ErrSaleNotActive) {
		t.Errorf("RegisterPayment() cash error = %v, want %v", err, domain.ErrSaleNotActive)
	}
	if _, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{Page: -1}, scope, now); !errors.Is(err, domain.ErrDueInstallmentInvalidPage) {
		t.Errorf("ListDueInstallments() invalid filter error = %v", err)
	}
}
