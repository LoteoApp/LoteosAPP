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
	if statement.Sale.ID != sale.ID || statement.Sale.PlanPago == nil || !statement.IssuedAt.Equal(now) {
		t.Fatalf("statement = %#v", statement)
	}
	if statement.DownPayment == nil || statement.DownPayment.Amount != 40000 || statement.DownPayment.State != domain.InstallmentStatePending {
		t.Errorf("entrega = %#v", statement.DownPayment)
	}
	if len(statement.Installments) != 3 || statement.Installments[0].Estado != domain.InstallmentStateOverdue || statement.Installments[1].Estado != domain.InstallmentStatePending {
		t.Errorf("cuotas = %#v", statement.Installments)
	}
	if statement.Summary.TotalAmount != 100000 || statement.Summary.PendingAmount != 100000 || statement.Summary.OverdueAmount != 20000 || statement.Summary.OverdueInstallments != 1 {
		t.Errorf("resumen = %#v", statement.Summary)
	}
	if len(statement.Payments) != 0 {
		t.Errorf("cobros = %#v, want none yet", statement.Payments)
	}

	// Paying cuota 2 while 1 is owed is rejected, as is a cuota of another plan.
	scope := gateway.SaleScope{}
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular, InstallmentIDs: []string{statement.Installments[1].ID},
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
		InstallmentIDs: []string{statement.Installments[0].ID}, IncludeDownPayment: true,
		Charges: []domain.PaymentChargeInput{
			{Type: domain.ChargeTypeServices, Amount: 150000, Currency: "ars", Detail: "Agua y luz"},
			{Type: domain.ChargeTypeAdministrative, Amount: 250.5},
		},
		Medium: domain.PaymentMediumTransfer, Observation: "Transferencia 123", PaidAt: paidAt, Now: now,
	}, scope)
	if err != nil {
		t.Fatalf("RegisterPayment() error = %v", err)
	}
	// The cobro's monto is only what went to the plan; the services in ARS
	// travel with it without being converted.
	if len(first.Charges) != 2 {
		t.Fatalf("charges = %#v", first.Charges)
	}
	// Charges read back in the order the totals use: the sale's currency
	// first, so the charge with no currency of its own leads.
	if first.Charges[0].ID == "" || first.Charges[0].Type != domain.ChargeTypeAdministrative || first.Charges[0].Amount != 250.5 || first.Charges[0].Currency != "USD" || first.Charges[0].Detail != "" {
		t.Errorf("first charge = %#v", first.Charges[0])
	}
	if first.Charges[1].Type != domain.ChargeTypeServices || first.Charges[1].Amount != 150000 || first.Charges[1].Currency != "ARS" || first.Charges[1].Detail != "Agua y luz" {
		t.Errorf("second charge = %#v", first.Charges[1])
	}
	wantTotals := []domain.CurrencyTotal{{Currency: "USD", Amount: 60250.5}, {Currency: "ARS", Amount: 150000}}
	if len(first.Totals) != 2 || first.Totals[0] != wantTotals[0] || first.Totals[1] != wantTotals[1] {
		t.Errorf("totals = %#v, want %#v", first.Totals, wantTotals)
	}
	if first.ID == "" || first.SaleID != sale.ID || first.Type != domain.PaymentTypeRegular || first.Amount != 60000 || first.Currency != "USD" {
		t.Errorf("first payment = %#v", first)
	}
	if first.Medium != domain.PaymentMediumTransfer || first.Observation != "Transferencia 123" || !first.PaidAt.Equal(paidAt) || first.CreatedBy.ID != actorID {
		t.Errorf("first payment terms = %#v", first)
	}
	if !first.IncludesDownPayment || first.DownPaymentAmount != 40000 || len(first.Installments) != 1 || first.Installments[0].Numero != 1 || first.Installments[0].Estado != domain.InstallmentStatePaid {
		t.Errorf("first payment items = %#v", first)
	}
	if got := lotStateOf(t, pool, sale.LoteID); got != string(domain.LotStateSold) {
		t.Errorf("lot state after a partial payment = %q, want vendido", got)
	}

	afterFirst, err := repository.GetDebtStatement(context.Background(), sale.ID, scope, now)
	if err != nil {
		t.Fatalf("GetDebtStatement() after payment error = %v", err)
	}
	if afterFirst.DownPayment.State != domain.InstallmentStatePaid || afterFirst.DownPayment.PaymentID != first.ID || afterFirst.DownPayment.PaidAt == nil {
		t.Errorf("entrega after payment = %#v", afterFirst.DownPayment)
	}
	if afterFirst.Installments[0].Estado != domain.InstallmentStatePaid || afterFirst.Installments[0].PaymentID != first.ID || afterFirst.Installments[0].FechaPago == nil {
		t.Errorf("cuota 1 after payment = %#v", afterFirst.Installments[0])
	}
	if afterFirst.Summary.PaidAmount != 60000 || afterFirst.Summary.PendingAmount != 40000 || afterFirst.Summary.OverdueInstallments != 0 || afterFirst.Summary.PaidInstallments != 1 {
		t.Errorf("resumen after payment = %#v", afterFirst.Summary)
	}
	if len(afterFirst.Payments) != 1 || afterFirst.Payments[0].ID != first.ID || len(afterFirst.Payments[0].Installments) != 1 || !afterFirst.Payments[0].IncludesDownPayment {
		t.Errorf("cobros after payment = %#v", afterFirst.Payments)
	}
	if len(afterFirst.Payments[0].Charges) != 2 || len(afterFirst.Payments[0].Totals) != 2 {
		t.Errorf("charges of the cobro = %#v", afterFirst.Payments[0])
	}
	// The statement totals the charges apart from the plan, per currency.
	wantCharges := []domain.CurrencyTotal{{Currency: "ARS", Amount: 150000}, {Currency: "USD", Amount: 250.5}}
	if len(afterFirst.CollectedCharges) != 2 || afterFirst.CollectedCharges[0] != wantCharges[0] || afterFirst.CollectedCharges[1] != wantCharges[1] {
		t.Errorf("charges collected = %#v, want %#v", afterFirst.CollectedCharges, wantCharges)
	}
	if afterFirst.Summary.PaidAmount != 60000 {
		t.Errorf("charges must not count as plan payments: %#v", afterFirst.Summary)
	}

	// An invalid charge rolls the whole cobro back.
	before := afterFirst.Summary.PaidAmount
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular, InstallmentIDs: []string{statement.Installments[1].ID},
		Charges: []domain.PaymentChargeInput{{Type: "propina", Amount: 10}},
		Medium:  domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, scope); !errors.Is(err, domain.ErrChargeInvalidType) {
		t.Fatalf("invalid charge error = %v, want %v", err, domain.ErrChargeInvalidType)
	}
	rolledBack, err := repository.GetDebtStatement(context.Background(), sale.ID, scope, now)
	if err != nil {
		t.Fatalf("GetDebtStatement() after the rejected cobro error = %v", err)
	}
	if rolledBack.Summary.PaidAmount != before || len(rolledBack.Payments) != 1 {
		t.Errorf("a rejected charge left something behind: %#v", rolledBack.Summary)
	}

	// Paying the same cuota or the entrega again is a conflict.
	if _, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular, InstallmentIDs: []string{statement.Installments[0].ID},
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
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeSettlement, ExpectedAmount: stale,
		Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, scope); !errors.Is(err, domain.ErrSettlementAmountMismatch) {
		t.Fatalf("stale settlement error = %v, want %v", err, domain.ErrSettlementAmountMismatch)
	}
	expected := 40000.0
	settlement, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeSettlement, ExpectedAmount: expected,
		Medium: domain.PaymentMediumCheque, PaidAt: now, Now: now,
	}, scope)
	if err != nil {
		t.Fatalf("settlement error = %v", err)
	}
	if settlement.Type != domain.PaymentTypeSettlement || settlement.Amount != 40000 || settlement.IncludesDownPayment || len(settlement.Installments) != 2 {
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
	if !final.Saldada() || final.Summary.PendingAmount != 0 || len(final.Payments) != 2 || final.Payments[0].ID != settlement.ID {
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
	if statement.DownPayment != nil || len(statement.Installments) != 2 || statement.Summary.TotalAmount != 110000 {
		t.Fatalf("statement = %#v", statement)
	}
	// Both cuotas in one regular cobro: nothing is left, so the venta closes
	// with the "plan completado" reason rather than the settlement one.
	payment, err := repository.RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
		SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular,
		InstallmentIDs: []string{statement.Installments[1].ID, statement.Installments[0].ID},
		Medium:         domain.PaymentMediumCash, PaidAt: now, Now: now,
	}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("RegisterPayment() error = %v", err)
	}
	if payment.Amount != 110000 || len(payment.Installments) != 2 || payment.Installments[0].Numero != 1 || payment.IncludesDownPayment {
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
	if page.Items[0].State != domain.InstallmentStateOverdue || page.Items[1].State != domain.InstallmentStatePending || page.Items[0].Number != 1 {
		t.Errorf("items = %#v", page.Items)
	}
	item := page.Items[0]
	if item.SaleID != sale.ID || item.InstallmentCount != 3 || item.Amount != 20000 || item.Currency != "USD" || item.DevelopmentName == "" || item.LotNumber != "1" || item.BlockNumber != "1" {
		t.Errorf("item = %#v", item)
	}
	if item.Client.ID != sale.Cliente.ID || item.Seller.ID != actorID || item.Agency != nil {
		t.Errorf("item parties = %#v", item)
	}
	if page.Summary.OverdueInstallments != 1 || page.Summary.UpcomingInstallments != 1 {
		t.Errorf("resumen = %#v", page.Summary)
	}

	overdue, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, States: []domain.InstallmentState{domain.InstallmentStateOverdue}}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() overdue error = %v", err)
	}
	if overdue.Total != 1 || overdue.Items[0].Number != 1 {
		t.Errorf("overdue = %#v", overdue)
	}
	from, to := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	july, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, From: &from, To: &to}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() july error = %v", err)
	}
	if july.Total != 1 || july.Items[0].Number != 2 {
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
	if noMatch.Total != 0 || len(noMatch.Items) != 0 || noMatch.Summary.OverdueInstallments != 1 {
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
	// A full page needs the count; the last, partial page tells it by itself.
	firstPage, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, Limit: 2}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() first page error = %v", err)
	}
	if firstPage.Total != 3 || firstPage.TotalPages != 2 || len(firstPage.Items) != 2 || firstPage.Items[0].Number != 1 || firstPage.Items[1].Number != 2 {
		t.Errorf("first page = %#v", firstPage)
	}
	lastPage, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, Limit: 2, Page: 2}, scope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() last page error = %v", err)
	}
	if lastPage.Total != 3 || lastPage.TotalPages != 2 || len(lastPage.Items) != 1 || lastPage.Items[0].Number != 3 || lastPage.Items[0].Client.ID != sale.Cliente.ID {
		t.Errorf("last page = %#v", lastPage)
	}

	strangerAuthID := newUUID(t)
	agencyScope := gateway.SaleScope{AssigneeAuthProviderID: &strangerAuthID, ByAgency: true}
	scoped, err := repository.ListDueInstallments(context.Background(), domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID}, agencyScope, now)
	if err != nil {
		t.Fatalf("ListDueInstallments() outside scope error = %v", err)
	}
	if scoped.Total != 0 || scoped.Summary.OverdueInstallments != 0 {
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

func TestCollectionRepositoryKeepsACuotaPendingOnItsDueDay(t *testing.T) {
	pool := collectionPool(t)
	// 12:00 in Argentina: cuota 1 falls due on Apr 15 at that clock.
	saleDate := time.Date(2026, 1, 15, 15, 0, 0, 0, time.UTC)
	sale, _ := financedSaleFixture(t, pool, saleDate)
	repository := postgres.NewCollectionRepository(pool)
	filter := domain.DueInstallmentFilter{DevelopmentID: sale.LoteoID, States: []domain.InstallmentState{domain.InstallmentStateOverdue}}

	cases := []struct {
		name        string
		now         time.Time
		wantOverdue bool
	}{
		{"due day, past the sale's clock", time.Date(2026, 4, 15, 23, 59, 0, 0, domain.BusinessLocation), false},
		{"due day, already the next day in UTC", time.Date(2026, 4, 16, 2, 59, 0, 0, time.UTC), false},
		{"the day after", time.Date(2026, 4, 16, 0, 0, 0, 0, domain.BusinessLocation), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			statement, err := repository.GetDebtStatement(context.Background(), sale.ID, gateway.SaleScope{}, tc.now)
			if err != nil {
				t.Fatalf("GetDebtStatement() error = %v", err)
			}
			if got := statement.Installments[0].Estado == domain.InstallmentStateOverdue; got != tc.wantOverdue {
				t.Errorf("statement cuota 1 = %s, want overdue %v", statement.Installments[0].Estado, tc.wantOverdue)
			}
			page, err := repository.ListDueInstallments(context.Background(), filter, gateway.SaleScope{}, tc.now)
			if err != nil {
				t.Fatalf("ListDueInstallments() error = %v", err)
			}
			wantCount := 0
			if tc.wantOverdue {
				wantCount = 1
			}
			if page.Total != wantCount || page.Summary.OverdueInstallments != wantCount {
				t.Errorf("overdue list total = %d, summary = %d, want %d", page.Total, page.Summary.OverdueInstallments, wantCount)
			}
		})
	}
}

func TestCollectionRepositoryReadsTheStatementInOneSnapshot(t *testing.T) {
	pool := collectionPool(t)
	saleDate := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	sale, actorID := financedSaleFixture(t, pool, saleDate)
	repository := postgres.NewCollectionRepository(pool)
	before, err := repository.GetDebtStatement(context.Background(), sale.ID, gateway.SaleScope{}, now)
	if err != nil {
		t.Fatalf("GetDebtStatement() error = %v", err)
	}

	// A cobro commits after the cuotas were read and before the cobros are.
	var paymentErr error
	postgres.SetAfterStatementInstallments(repository, func() {
		_, paymentErr = postgres.NewCollectionRepository(pool).RegisterPayment(context.Background(), gateway.RegisterPaymentCommand{
			SaleID: sale.ID, ActorID: actorID, Type: domain.PaymentTypeRegular, InstallmentIDs: []string{before.Installments[0].ID},
			Medium: domain.PaymentMediumCash, PaidAt: now, Now: now,
		}, gateway.SaleScope{})
	})
	during, err := repository.GetDebtStatement(context.Background(), sale.ID, gateway.SaleScope{}, now)
	if err != nil {
		t.Fatalf("GetDebtStatement() during a cobro error = %v", err)
	}
	if paymentErr != nil {
		t.Fatalf("RegisterPayment() mid-read error = %v", paymentErr)
	}
	if during.Installments[0].Estado == domain.InstallmentStatePaid || len(during.Payments) != 0 {
		t.Errorf("mid-read statement = cuota 1 %s with %d cobros, want the state before the cobro", during.Installments[0].Estado, len(during.Payments))
	}

	postgres.SetAfterStatementInstallments(repository, nil)
	after, err := repository.GetDebtStatement(context.Background(), sale.ID, gateway.SaleScope{}, now)
	if err != nil {
		t.Fatalf("GetDebtStatement() after the cobro error = %v", err)
	}
	if after.Installments[0].Estado != domain.InstallmentStatePaid || len(after.Payments) != 1 {
		t.Errorf("statement after the cobro = cuota 1 %s with %d cobros", after.Installments[0].Estado, len(after.Payments))
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
