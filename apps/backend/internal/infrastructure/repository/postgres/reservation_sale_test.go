package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

type syncClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *syncClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *syncClock) Set(now time.Time) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = now
}

func conversionPool(t *testing.T) *pgxpool.Pool {
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

type conversionSetup struct {
	adminID, adminAuthID, clientID, loteoID, lotID string
}

func newConversionSetup(t *testing.T, pool *pgxpool.Pool) conversionSetup {
	t.Helper()
	adminID, clientID, loteoID, lotID := reservationFixture(t, pool)
	return conversionSetup{
		adminID: adminID, adminAuthID: authProviderIDOf(t, pool, adminID),
		clientID: clientID, loteoID: loteoID, lotID: lotID,
	}
}

func authProviderIDOf(t *testing.T, pool *pgxpool.Pool, userID string) string {
	t.Helper()
	var authID string
	if err := pool.QueryRow(context.Background(), `SELECT auth_provider_id::text FROM usuarios WHERE id = $1::uuid`, userID).Scan(&authID); err != nil {
		t.Fatalf("read auth provider id: %v", err)
	}
	return authID
}

// reserve registers, as the administrator, a reserva held by sellerID that
// starts at createdAt.
func (setup conversionSetup) reserve(t *testing.T, pool *pgxpool.Pool, lotID, sellerID string, createdAt time.Time) domain.Reservation {
	t.Helper()
	repository := postgres.NewReservationRepository(pool, fixedReservationClock{now: createdAt})
	reservation, err := repository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID: setup.loteoID, LoteID: lotID, ClienteID: setup.clientID, VendedorID: sellerID,
		ActorID: setup.adminID, ActorAuthProviderID: setup.adminAuthID,
		IdempotencyKey: "conversion-reserve-" + newUUID(t), IdempotencyPayloadHash: saleHash(t), CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	return reservation
}

func conversionCommand(t *testing.T, reservationID, actorID, actorAuthID string) gateway.ConvertReservationCommand {
	t.Helper()
	return gateway.ConvertReservationCommand{
		ReservationID: reservationID, ActorID: actorID, ActorAuthProviderID: actorAuthID,
		PaymentMethod:     domain.PaymentMethodCash,
		InstallmentDueDay: domain.DefaultInstallmentDueDay,
		IdempotencyKey:    "conversion-" + newUUID(t), IdempotencyPayloadHash: saleHash(t),
	}
}

func readState(t *testing.T, pool *pgxpool.Pool, table, id string) string {
	t.Helper()
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT estado_actual FROM `+table+` WHERE id = $1::uuid`, id).Scan(&state); err != nil {
		t.Fatalf("read %s state: %v", table, err)
	}
	return state
}

func countSalesOf(t *testing.T, pool *pgxpool.Pool, reservationID string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM ventas WHERE reserva_id = $1::uuid`, reservationID).Scan(&count); err != nil {
		t.Fatalf("count ventas of reserva: %v", err)
	}
	return count
}

func TestConvertReservationContadoLinksAndSettles(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	reservedAt := time.Date(2027, 2, 1, 12, 0, 0, 0, time.UTC)
	convertedAt := reservedAt.Add(30 * time.Minute)
	reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, reservedAt)
	reservations := postgres.NewReservationRepository(pool, fixedReservationClock{now: convertedAt})
	ownScope := gateway.ReservationScope{ActorAuthProviderID: &setup.adminAuthID}

	before, err := reservations.Get(context.Background(), reservation.ID, ownScope)
	if err != nil {
		t.Fatalf("Get() before conversion error = %v", err)
	}
	if !before.PuedeConvertir || before.VentaID != nil {
		t.Fatalf("before conversion = puedeConvertir %v, ventaId %v; want convertible and unlinked", before.PuedeConvertir, before.VentaID)
	}

	repository := postgres.NewSaleRepository(pool, fixedReservationClock{now: convertedAt})
	command := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
	sale, err := repository.ConvertReservation(context.Background(), command)
	if err != nil {
		t.Fatalf("ConvertReservation() error = %v", err)
	}
	if sale.ReservaID == nil || *sale.ReservaID != reservation.ID {
		t.Fatalf("sale reserva = %v, want %s", sale.ReservaID, reservation.ID)
	}
	if sale.LoteID != setup.lotID || sale.Cliente.ID != setup.clientID || sale.Vendedor.ID != setup.adminID || sale.UsuarioAlta.ID != setup.adminID {
		t.Errorf("sale parties = %#v", sale)
	}
	if sale.Monto != 100000 || sale.Moneda != "USD" || !sale.FechaCreacion.Equal(convertedAt) {
		t.Errorf("sale terms = %v %s at %s, want 100000 USD at the conversion instant", sale.Monto, sale.Moneda, sale.FechaCreacion)
	}
	if sale.Estado != domain.SaleStateCompleted || len(sale.Historial) != 2 {
		t.Errorf("contado sale = %q with %d entries, want completada", sale.Estado, len(sale.Historial))
	}
	if got := readState(t, pool, "lotes", setup.lotID); got != string(domain.LotStateCompleted) {
		t.Errorf("lot state = %q, want finalizado", got)
	}
	if got := readState(t, pool, "reservas", reservation.ID); got != string(domain.ReservationStateConverted) {
		t.Errorf("reservation state = %q, want convertida", got)
	}

	var soldEvents int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM lote_estados
		WHERE lote_id = $1::uuid AND estado = 'vendido' AND origen = 'venta'
		  AND reserva_id = $2::uuid AND venta_id = $3::uuid
	`, setup.lotID, reservation.ID, sale.ID).Scan(&soldEvents); err != nil {
		t.Fatalf("read lot events: %v", err)
	}
	if soldEvents != 1 {
		t.Errorf("reservado -> vendido events linking both = %d, want 1", soldEvents)
	}
	var released int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM lote_estados WHERE lote_id = $1::uuid AND estado = 'disponible' AND origen <> 'alta'
	`, setup.lotID).Scan(&released); err != nil {
		t.Fatalf("read releases: %v", err)
	}
	if released != 0 {
		t.Errorf("lot released %d times during the conversion, want never", released)
	}

	after, err := reservations.Get(context.Background(), reservation.ID, ownScope)
	if err != nil {
		t.Fatalf("Get() after conversion error = %v", err)
	}
	if after.PuedeConvertir || after.VentaID == nil || *after.VentaID != sale.ID {
		t.Errorf("after conversion = puedeConvertir %v, ventaId %v; want linked to %s", after.PuedeConvertir, after.VentaID, sale.ID)
	}
	if last := after.Historial[len(after.Historial)-1]; last.Estado != domain.ReservationStateConverted || last.Usuario == nil || last.Usuario.ID != setup.adminID {
		t.Errorf("last reservation event = %#v", last)
	}
	listed, err := reservations.List(context.Background(), domain.ReservationListFilter{LoteoID: setup.loteoID}, ownScope)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].VentaID == nil || listed.Items[0].PuedeConvertir {
		t.Errorf("List() = %#v, want the linked reserva", listed.Items)
	}

	// A price change after the venta doesn't touch its replay.
	if _, err := pool.Exec(context.Background(), `UPDATE lotes SET precio = 250000 WHERE id = $1::uuid`, setup.lotID); err != nil {
		t.Fatalf("change price: %v", err)
	}
	replayed, err := repository.ConvertReservation(context.Background(), command)
	if err != nil {
		t.Fatalf("replayed ConvertReservation() error = %v", err)
	}
	if replayed.ID != sale.ID || replayed.Monto != 100000 || len(replayed.Historial) != 2 {
		t.Errorf("replay = %#v, want the original sale", replayed)
	}

	otherTerms := command
	otherTerms.IdempotencyPayloadHash = saleHash(t)
	if _, err := repository.ConvertReservation(context.Background(), otherTerms); !errors.Is(err, domain.ErrSaleIdempotencyConflict) {
		t.Fatalf("same key, other terms error = %v, want %v", err, domain.ErrSaleIdempotencyConflict)
	}
	otherKey := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
	if _, err := repository.ConvertReservation(context.Background(), otherKey); !errors.Is(err, domain.ErrReservationConverted) {
		t.Fatalf("second conversion error = %v, want %v", err, domain.ErrReservationConverted)
	}
	if countSalesOf(t, pool, reservation.ID) != 1 {
		t.Errorf("ventas of the reserva = %d, want 1", countSalesOf(t, pool, reservation.ID))
	}

	fetched, err := repository.Get(context.Background(), sale.ID, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("Get() sale error = %v", err)
	}
	if fetched.ReservaID == nil || *fetched.ReservaID != reservation.ID {
		t.Errorf("Get() sale reserva = %v", fetched.ReservaID)
	}

	if _, err := reservations.Cancel(context.Background(), gateway.CancelReservationCommand{
		ReservationID: reservation.ID, ActorID: setup.adminID, Reason: "Cliente desistió",
	}, gateway.ReservationScope{}); !errors.Is(err, domain.ErrReservationConverted) {
		t.Fatalf("Cancel() after conversion error = %v, want %v", err, domain.ErrReservationConverted)
	}
}

func TestConvertReservationFinancedStartsAtTheConversionInstant(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	reservedAt := time.Date(2027, 1, 20, 15, 0, 0, 0, time.UTC)
	convertedAt := time.Date(2027, 1, 31, 15, 0, 0, 0, time.UTC)
	reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, reservedAt)
	_, secondLotID := secondLotFixture(t, pool, setup.loteoID)
	downPayment := setup.reserve(t, pool, secondLotID, setup.adminID, reservedAt)
	repository := postgres.NewSaleRepository(pool, fixedReservationClock{now: convertedAt})

	financed := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
	financed.PaymentMethod = domain.PaymentMethodFinanced
	financed.PaymentPlan = &domain.PaymentPlanInput{Installments: 12, InterestRate: 10, Period: domain.PaymentPeriodMonthly}
	sale, err := repository.ConvertReservation(context.Background(), financed)
	if err != nil {
		t.Fatalf("ConvertReservation() financed error = %v", err)
	}
	if sale.Estado != domain.SaleStateActive || sale.PlanPago == nil || len(sale.PlanPago.Cuotas) != 12 {
		t.Fatalf("financed sale = %#v", sale)
	}
	if sale.PlanPago.MontoTotal != 110000.04 || sale.PlanPago.MontoCuota != 9166.67 {
		t.Errorf("plan amounts = %#v", sale.PlanPago)
	}
	if !sale.PlanPago.Cuotas[0].FechaVencimiento.Equal(time.Date(2027, 2, 10, 15, 0, 0, 0, time.UTC)) {
		t.Errorf("first due date = %s, want the 10th of the month after the conversion", sale.PlanPago.Cuotas[0].FechaVencimiento)
	}
	if got := readState(t, pool, "lotes", setup.lotID); got != string(domain.LotStateSold) {
		t.Errorf("financed lot state = %q, want vendido", got)
	}

	down := conversionCommand(t, downPayment.ID, setup.adminID, setup.adminAuthID)
	down.PaymentMethod = domain.PaymentMethodDownAndFi
	down.PaymentPlan = &domain.PaymentPlanInput{Installments: 3, Period: domain.PaymentPeriodMonthly, DownPayment: 90000}
	if _, err := repository.ConvertReservation(context.Background(), down); !errors.Is(err, domain.ErrSaleInvalidDownPayment) {
		t.Fatalf("down payment equal to the price error = %v, want %v", err, domain.ErrSaleInvalidDownPayment)
	}
	if got := readState(t, pool, "reservas", downPayment.ID); got != string(domain.ReservationStateActive) {
		t.Fatalf("reserva after a rejected plan = %q, want activa", got)
	}
	down.PaymentPlan = &domain.PaymentPlanInput{Installments: 3, Period: domain.PaymentPeriodMonthly, DownPayment: 30000}
	downSale, err := repository.ConvertReservation(context.Background(), down)
	if err != nil {
		t.Fatalf("ConvertReservation() down payment error = %v", err)
	}
	if downSale.Monto != 90000 || downSale.PlanPago == nil || downSale.PlanPago.MontoEntrega != 30000 || downSale.PlanPago.MontoFinanciado != 60000 {
		t.Errorf("down payment sale = %#v", downSale.PlanPago)
	}
}

func TestConvertReservationRequiresTheSellerWithinScope(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	sellerID, sellerAuthID, peerID, agencyID, _ := agencySellerFixture(t, pool, setup.loteoID)
	otherAgencySellerID, _ := unassignedAgencySellerFixture(t, pool)
	t.Cleanup(func() { deleteLoteo(t, pool, setup.loteoID) })
	now := time.Now().UTC().Truncate(time.Microsecond)
	reservation := setup.reserve(t, pool, setup.lotID, sellerID, now.Add(-time.Hour))
	repository := postgres.NewSaleRepository(pool, fixedReservationClock{now: now})
	agencyScope := func(authID string) gateway.ReservationScope {
		return gateway.ReservationScope{AssigneeAuthProviderID: &authID, ByAgencyAssignment: true, ActorAuthProviderID: &authID}
	}
	peerAuthID := authProviderIDOf(t, pool, peerID)
	otherAuthID := authProviderIDOf(t, pool, otherAgencySellerID)

	for name, test := range map[string]struct {
		command gateway.ConvertReservationCommand
		want    error
	}{
		"colleague of the same agency": {conversionCommand(t, reservation.ID, peerID, peerAuthID), domain.ErrReservationConvertForbidden},
		"seller of another agency":     {conversionCommand(t, reservation.ID, otherAgencySellerID, otherAuthID), domain.ErrReservationNotFound},
		"unknown reserva":              {conversionCommand(t, newUUID(t), sellerID, sellerAuthID), domain.ErrReservationNotFound},
		"malformed reserva":            {conversionCommand(t, "nope", sellerID, sellerAuthID), domain.ErrReservationNotFound},
		"unprovisioned actor":          {conversionCommand(t, reservation.ID, newUUID(t), newUUID(t)), domain.ErrActorNoAprovisionado},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := repository.ConvertReservation(context.Background(), test.command); !errors.Is(err, test.want) {
				t.Fatalf("ConvertReservation() error = %v, want %v", err, test.want)
			}
		})
	}

	peerView, err := postgres.NewReservationRepository(pool, fixedReservationClock{now: now}).Get(context.Background(), reservation.ID, agencyScope(peerAuthID))
	if err != nil {
		t.Fatalf("Get() as colleague error = %v", err)
	}
	if peerView.PuedeConvertir {
		t.Error("a colleague reads puedeConvertir = true, want false")
	}

	if _, err := pool.Exec(context.Background(), `
		UPDATE inmobiliaria_loteos SET fecha_baja = now() WHERE inmobiliaria_id = $1::uuid AND loteo_id = $2::uuid
	`, agencyID, setup.loteoID); err != nil {
		t.Fatalf("unassign agency: %v", err)
	}
	if _, err := repository.ConvertReservation(context.Background(), conversionCommand(t, reservation.ID, sellerID, sellerAuthID)); !errors.Is(err, domain.ErrSaleAgencyNotAssigned) {
		t.Fatalf("unassigned agency error = %v, want %v", err, domain.ErrSaleAgencyNotAssigned)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE inmobiliaria_loteos SET fecha_baja = NULL WHERE inmobiliaria_id = $1::uuid AND loteo_id = $2::uuid
	`, agencyID, setup.loteoID); err != nil {
		t.Fatalf("reassign agency: %v", err)
	}

	sale, err := repository.ConvertReservation(context.Background(), conversionCommand(t, reservation.ID, sellerID, sellerAuthID))
	if err != nil {
		t.Fatalf("ConvertReservation() by the agency seller error = %v", err)
	}
	if sale.Vendedor.ID != sellerID || sale.Inmobiliaria == nil || sale.Inmobiliaria.ID != agencyID {
		t.Errorf("agency sale = %#v", sale)
	}
}

func TestConvertReservationByAnInternalUserForTheSeller(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	sellerID, _, _, agencyID, _ := agencySellerFixture(t, pool, setup.loteoID)
	t.Cleanup(func() { deleteLoteo(t, pool, setup.loteoID) })
	reservedAt := time.Date(2027, 7, 1, 12, 0, 0, 0, time.UTC)
	now := reservedAt.Add(time.Hour)
	reservation := setup.reserve(t, pool, setup.lotID, sellerID, reservedAt)
	reservations := postgres.NewReservationRepository(pool, fixedReservationClock{now: now})
	repository := postgres.NewSaleRepository(pool, fixedReservationClock{now: now})
	adminScope := gateway.ReservationScope{ActorAuthProviderID: &setup.adminAuthID}
	canConvert := func() bool {
		t.Helper()
		read, err := reservations.Get(context.Background(), reservation.ID, adminScope)
		if err != nil {
			t.Fatalf("Get() as administrator error = %v", err)
		}
		return read.PuedeConvertir
	}

	// Like an ordinary sale for someone else, the seller has to stay eligible.
	for name, toggle := range map[string]struct{ statement, id string }{
		"inactive agency": {`UPDATE inmobiliarias SET fecha_baja = %s WHERE id = $1::uuid`, agencyID},
		"inactive seller": {`UPDATE usuarios SET fecha_baja = %s WHERE id = $1::uuid`, sellerID},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := pool.Exec(context.Background(), fmt.Sprintf(toggle.statement, "now()"), toggle.id); err != nil {
				t.Fatalf("deactivate: %v", err)
			}
			defer func() {
				if _, err := pool.Exec(context.Background(), fmt.Sprintf(toggle.statement, "NULL"), toggle.id); err != nil {
					t.Errorf("reactivate: %v", err)
				}
			}()
			if canConvert() {
				t.Error("puedeConvertir = true with an ineligible seller")
			}
			_, err := repository.ConvertReservation(context.Background(), conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID))
			if !errors.Is(err, domain.ErrSaleSellerNotEligible) {
				t.Fatalf("ConvertReservation() error = %v, want %v", err, domain.ErrSaleSellerNotEligible)
			}
		})
	}

	if !canConvert() {
		t.Fatal("an administrator reads puedeConvertir = false for someone else's reserva")
	}
	sale, err := repository.ConvertReservation(context.Background(), conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID))
	if err != nil {
		t.Fatalf("ConvertReservation() by the administrator error = %v", err)
	}
	if sale.Vendedor.ID != sellerID || sale.UsuarioAlta.ID != setup.adminID || sale.Inmobiliaria == nil || sale.Inmobiliaria.ID != agencyID {
		t.Errorf("sale = vendedor %s, alta %s, inmobiliaria %#v; want the reserva's seller registered by the administrator",
			sale.Vendedor.ID, sale.UsuarioAlta.ID, sale.Inmobiliaria)
	}
	if got := readState(t, pool, "reservas", reservation.ID); got != string(domain.ReservationStateConverted) {
		t.Errorf("reserva = %q, want convertida", got)
	}
}

func TestConvertReservationRejectsReservasThatCannotBeSold(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	reservedAt := time.Date(2027, 3, 1, 12, 0, 0, 0, time.UTC)
	due := reservedAt.Add(domain.ReservationDuration)
	reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, reservedAt)
	// Every rejection is checked on both sides: the POST refuses it and the
	// read never offers it, so puedeConvertir can't drift from the guards.
	canConvertAt := func(now time.Time) bool {
		t.Helper()
		read, err := postgres.NewReservationRepository(pool, fixedReservationClock{now: now}).
			Get(context.Background(), reservation.ID, gateway.ReservationScope{ActorAuthProviderID: &setup.adminAuthID})
		if err != nil {
			t.Fatalf("Get() at %s error = %v", now, err)
		}
		return read.PuedeConvertir
	}
	convert := func(now time.Time) error {
		t.Helper()
		if canConvertAt(now) {
			t.Errorf("puedeConvertir = true at %s for a reserva the conversion rejects", now)
		}
		_, err := postgres.NewSaleRepository(pool, fixedReservationClock{now: now}).
			ConvertReservation(context.Background(), conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID))
		return err
	}
	if !canConvertAt(due.Add(-time.Minute)) {
		t.Fatal("puedeConvertir = false for a convertible reserva, so the checks below prove nothing")
	}

	// The due instant itself is already late, and the lote stays reservado
	// for the worker to release.
	for _, now := range []time.Time{due, due.Add(time.Minute)} {
		if err := convert(now); !errors.Is(err, domain.ErrReservationConversionExpired) {
			t.Fatalf("convert at %s error = %v, want %v", now, err, domain.ErrReservationConversionExpired)
		}
	}
	if got := readState(t, pool, "lotes", setup.lotID); got != string(domain.LotStateReserved) {
		t.Fatalf("lot after an expired conversion = %q, want reservado", got)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE lotes SET precio = NULL WHERE id = $1::uuid`, setup.lotID); err != nil {
		t.Fatalf("clear price: %v", err)
	}
	if err := convert(due.Add(-time.Minute)); !errors.Is(err, domain.ErrSaleLotIncomplete) {
		t.Fatalf("lot without price error = %v, want %v", err, domain.ErrSaleLotIncomplete)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE lotes SET precio = 100000 WHERE id = $1::uuid`, setup.lotID); err != nil {
		t.Fatalf("restore price: %v", err)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE clientes SET fecha_baja = now() WHERE id = $1::uuid`, setup.clientID); err != nil {
		t.Fatalf("deactivate client: %v", err)
	}
	if err := convert(due.Add(-time.Minute)); !errors.Is(err, domain.ErrSaleInvalidClient) {
		t.Fatalf("inactive client error = %v, want %v", err, domain.ErrSaleInvalidClient)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE clientes SET fecha_baja = NULL WHERE id = $1::uuid`, setup.clientID); err != nil {
		t.Fatalf("reactivate client: %v", err)
	}

	if _, err := postgres.NewReservationRepository(pool, fixedReservationClock{now: reservedAt.Add(time.Hour)}).Cancel(context.Background(), gateway.CancelReservationCommand{
		ReservationID: reservation.ID, ActorID: setup.adminID, Reason: "Cliente desistió",
	}, gateway.ReservationScope{}); err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if err := convert(reservedAt.Add(2 * time.Hour)); !errors.Is(err, domain.ErrReservationAlreadyCancelled) {
		t.Fatalf("cancelled reserva error = %v, want %v", err, domain.ErrReservationAlreadyCancelled)
	}
	if countSalesOf(t, pool, reservation.ID) != 0 {
		t.Error("a rejected conversion left a venta behind")
	}
}

func TestConvertReservationAfterTheWorkerExpiredIt(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	reservedAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	afterDue := reservedAt.Add(domain.ReservationDuration + time.Minute)
	reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, reservedAt)

	if _, err := postgres.NewReservationRepository(pool, fixedReservationClock{now: afterDue}).ExpireDue(context.Background(), afterDue, 1000); err != nil {
		t.Fatalf("ExpireDue() error = %v", err)
	}
	_, err := postgres.NewSaleRepository(pool, fixedReservationClock{now: afterDue}).
		ConvertReservation(context.Background(), conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID))
	if !errors.Is(err, domain.ErrReservationConversionExpired) {
		t.Fatalf("convert a vencida reserva error = %v, want %v", err, domain.ErrReservationConversionExpired)
	}
	if got := readState(t, pool, "lotes", setup.lotID); got != string(domain.LotStateAvailable) {
		t.Errorf("lot = %q, want disponible as the worker left it", got)
	}
}

func TestConvertReservationKeyOfAnOrdinarySaleConflicts(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, now.Add(-time.Hour))
	_, secondLotID := secondLotFixture(t, pool, setup.loteoID)
	key := "shared-key-" + newUUID(t)
	if _, err := postgres.NewSaleRepository(pool).Create(context.Background(), gateway.CreateSaleCommand{
		DevelopmentID: setup.loteoID, LotID: secondLotID, ClientID: setup.clientID, SellerID: setup.adminID, ActorID: setup.adminID,
		PaymentMethod: domain.PaymentMethodCash, IdempotencyKey: key, IdempotencyPayloadHash: saleHash(t), CreatedAt: now,
	}); err != nil {
		t.Fatalf("ordinary Create() error = %v", err)
	}
	command := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
	command.IdempotencyKey = key
	if _, err := postgres.NewSaleRepository(pool, fixedReservationClock{now: now}).ConvertReservation(context.Background(), command); !errors.Is(err, domain.ErrSaleIdempotencyConflict) {
		t.Fatalf("conversion with an ordinary sale key error = %v, want %v", err, domain.ErrSaleIdempotencyConflict)
	}
	if got := readState(t, pool, "reservas", reservation.ID); got != string(domain.ReservationStateActive) {
		t.Errorf("reserva = %q, want activa", got)
	}
}

func TestConvertReservationRollsBackEveryWrite(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, now.Add(-time.Hour))
	repository := postgres.NewSaleRepository(pool, fixedReservationClock{now: now})
	injected := errors.New("injected failure after the conversion writes")
	var sawSale bool
	postgres.SetAfterConversionWrite(repository, func(ctx context.Context, tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM ventas WHERE reserva_id = $1::uuid)`, reservation.ID).Scan(&sawSale); err != nil {
			return err
		}
		return injected
	})

	command := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
	command.PaymentMethod = domain.PaymentMethodFinanced
	command.PaymentPlan = &domain.PaymentPlanInput{Installments: 6, Period: domain.PaymentPeriodMonthly}
	if _, err := repository.ConvertReservation(context.Background(), command); !errors.Is(err, injected) {
		t.Fatalf("ConvertReservation() error = %v, want the injected failure", err)
	}
	if !sawSale {
		t.Fatal("the hook ran before the venta was written")
	}
	if countSalesOf(t, pool, reservation.ID) != 0 {
		t.Error("the venta survived the rollback")
	}
	var plans, saleEvents, convertedEvents int
	if err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM planes_pago p JOIN ventas v ON v.id = p.venta_id WHERE v.lote_id = $1::uuid),
			(SELECT count(*) FROM lote_estados WHERE lote_id = $1::uuid AND origen = 'venta'),
			(SELECT count(*) FROM reserva_estados WHERE reserva_id = $2::uuid AND estado = 'convertida')
	`, setup.lotID, reservation.ID).Scan(&plans, &saleEvents, &convertedEvents); err != nil {
		t.Fatalf("read leftovers: %v", err)
	}
	if plans != 0 || saleEvents != 0 || convertedEvents != 0 {
		t.Errorf("leftovers = %d plans, %d lot sale events, %d convertida events; want none", plans, saleEvents, convertedEvents)
	}
	if readState(t, pool, "reservas", reservation.ID) != string(domain.ReservationStateActive) || readState(t, pool, "lotes", setup.lotID) != string(domain.LotStateReserved) {
		t.Error("reserva and lote must keep their states after a rollback")
	}

	// A failure reported as ambiguous is reconciled: nothing was committed,
	// so the original failure comes back instead of a made-up venta.
	if _, err := postgres.ReconcileConversion(context.Background(), repository, command, postgres.ErrRetrySaleWrite); !errors.Is(err, postgres.ErrRetrySaleWrite) {
		t.Fatalf("ReconcileConversion() without a venta error = %v, want the original", err)
	}

	postgres.SetAfterConversionWrite(repository, nil)
	sale, err := repository.ConvertReservation(context.Background(), command)
	if err != nil {
		t.Fatalf("retried ConvertReservation() error = %v", err)
	}
	reconciled, err := postgres.ReconcileConversion(context.Background(), repository, command, postgres.ErrRetrySaleWrite)
	if err != nil || reconciled.ID != sale.ID {
		t.Fatalf("ReconcileConversion() after commit = %#v, %v; want the committed venta", reconciled, err)
	}
	conflicting := command
	conflicting.IdempotencyPayloadHash = saleHash(t)
	if _, err := postgres.ReconcileConversion(context.Background(), repository, conflicting, postgres.ErrRetrySaleWrite); !errors.Is(err, domain.ErrSaleIdempotencyConflict) {
		t.Fatalf("ReconcileConversion() other terms error = %v, want %v", err, domain.ErrSaleIdempotencyConflict)
	}
	// The reconciliation reads the actor's role again: once moved to an
	// agency, the reserva (sold without one) is out of their scope.
	_, agencyID := unassignedAgencySellerFixture(t, pool)
	moveActorToAgency(t, pool, setup.adminID, agencyID)
	if _, err := postgres.ReconcileConversion(context.Background(), repository, command, postgres.ErrRetrySaleWrite); !errors.Is(err, domain.ErrReservationNotFound) {
		t.Fatalf("ReconcileConversion() outside the current scope error = %v, want %v", err, domain.ErrReservationNotFound)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE usuarios SET fecha_baja = now() WHERE id = $1::uuid`, setup.adminID); err != nil {
		t.Fatalf("deactivate actor: %v", err)
	}
	if _, err := postgres.ReconcileConversion(context.Background(), repository, command, postgres.ErrRetrySaleWrite); !errors.Is(err, domain.ErrCuentaInactiva) {
		t.Fatalf("ReconcileConversion() by an inactive actor error = %v, want %v", err, domain.ErrCuentaInactiva)
	}
}

// moveActorToAgency turns an internal user into an agency user for the rest
// of the test and restores them afterwards, before the agency is removed.
func moveActorToAgency(t *testing.T, pool *pgxpool.Pool, userID, agencyID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		UPDATE usuarios SET rol = 'inmobiliaria', inmobiliaria_id = $2::uuid WHERE id = $1::uuid
	`, userID, agencyID); err != nil {
		t.Fatalf("move actor to agency: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			UPDATE usuarios SET rol = 'administrador', inmobiliaria_id = NULL, fecha_baja = NULL WHERE id = $1::uuid
		`, userID); err != nil {
			t.Errorf("restore actor: %v", err)
		}
	})
}

func TestConvertReservationScopesByTheRoleItLocks(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	_, _, _, agencyID, _ := agencySellerFixture(t, pool, setup.loteoID)
	reservedAt := time.Date(2027, 6, 1, 12, 0, 0, 0, time.UTC)
	reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, reservedAt)
	command := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
	now := reservedAt.Add(time.Hour)

	// The role change is written but not committed: the conversion still
	// locates the reserva with the internal scope, then waits for the actor's
	// row. Once the change commits, the locked role puts the actor in an
	// agency assigned to the loteo, and a reserva taken without that agency
	// is out of their scope.
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			UPDATE usuarios SET rol = 'administrador', inmobiliaria_id = NULL WHERE id = $1::uuid
		`, setup.adminID); err != nil {
			t.Errorf("restore actor: %v", err)
		}
	})
	change, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin role change: %v", err)
	}
	defer change.Rollback(context.Background())
	var changePID int
	if err := change.QueryRow(context.Background(), `SELECT pg_backend_pid()`).Scan(&changePID); err != nil {
		t.Fatalf("read role change pid: %v", err)
	}
	if _, err := change.Exec(context.Background(), `
		UPDATE usuarios SET rol = 'inmobiliaria', inmobiliaria_id = $2::uuid WHERE id = $1::uuid
	`, setup.adminID, agencyID); err != nil {
		t.Fatalf("change role: %v", err)
	}

	converted := make(chan error, 1)
	go func() {
		_, err := postgres.NewSaleRepository(pool, fixedReservationClock{now: now}).ConvertReservation(context.Background(), command)
		converted <- err
	}()
	waitUntilBlockedBy(t, pool, changePID)
	if err := change.Commit(context.Background()); err != nil {
		t.Fatalf("commit role change: %v", err)
	}

	if err := <-converted; !errors.Is(err, domain.ErrReservationNotFound) {
		t.Fatalf("ConvertReservation() after the role change error = %v, want %v", err, domain.ErrReservationNotFound)
	}
	if countSalesOf(t, pool, reservation.ID) != 0 {
		t.Error("a venta was registered outside the actor's current scope")
	}

	// A read with a scope wider than the actor's current one (stale claims)
	// must not offer what the conversion would refuse.
	wide := gateway.ReservationScope{ActorAuthProviderID: &setup.adminAuthID}
	read, err := postgres.NewReservationRepository(pool, fixedReservationClock{now: now}).Get(context.Background(), reservation.ID, wide)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if read.PuedeConvertir {
		t.Error("puedeConvertir = true for a seller whose current agency isn't the reserva's")
	}
	listed, err := postgres.NewReservationRepository(pool, fixedReservationClock{now: now}).List(context.Background(), domain.ReservationListFilter{LoteoID: setup.loteoID}, wide)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].PuedeConvertir {
		t.Errorf("List() = %#v, want the reserva without puedeConvertir", listed.Items)
	}
}

func TestConvertReservationReadsTheClockAfterWaitingForTheLote(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	reservedAt := time.Now().UTC().Add(-domain.ReservationDuration + time.Hour).Truncate(time.Microsecond)
	reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, reservedAt)
	clock := &syncClock{now: reservedAt.Add(domain.ReservationDuration - time.Minute)}
	repository := postgres.NewSaleRepository(pool, clock)

	holder, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin holder: %v", err)
	}
	defer holder.Rollback(context.Background())
	var holderPID int
	if err := holder.QueryRow(context.Background(), `SELECT pg_backend_pid()`).Scan(&holderPID); err != nil {
		t.Fatalf("read holder pid: %v", err)
	}
	if _, err := holder.Exec(context.Background(), `SELECT 1 FROM lotes WHERE id = $1::uuid FOR UPDATE`, setup.lotID); err != nil {
		t.Fatalf("lock lote: %v", err)
	}

	command := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
	result := make(chan error, 1)
	go func() {
		_, err := repository.ConvertReservation(context.Background(), command)
		result <- err
	}()
	waitUntilBlockedBy(t, pool, holderPID)

	// The reserva runs out while the conversion waits for the lote.
	clock.Set(reservedAt.Add(domain.ReservationDuration))
	if err := holder.Rollback(context.Background()); err != nil {
		t.Fatalf("release lote: %v", err)
	}
	if err := <-result; !errors.Is(err, domain.ErrReservationConversionExpired) {
		t.Fatalf("ConvertReservation() after the wait error = %v, want %v", err, domain.ErrReservationConversionExpired)
	}
	if countSalesOf(t, pool, reservation.ID) != 0 {
		t.Error("an expired conversion registered a venta")
	}
}

func waitUntilBlockedBy(t *testing.T, pool *pgxpool.Pool, holderPID int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var blocked bool
		if err := pool.QueryRow(context.Background(), `
			SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE $1 = ANY(pg_blocking_pids(pid)))
		`, holderPID).Scan(&blocked); err != nil {
			t.Fatalf("read blocked sessions: %v", err)
		}
		if blocked {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the conversion never waited for the locked lote")
}

func TestConvertReservationRacesResolveToOneOutcome(t *testing.T) {
	pool := conversionPool(t)

	t.Run("two conversions with different keys", func(t *testing.T) {
		setup := newConversionSetup(t, pool)
		now := time.Now().UTC().Truncate(time.Microsecond)
		reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, now.Add(-time.Hour))
		repository := postgres.NewSaleRepository(pool, fixedReservationClock{now: now})
		first := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
		second := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
		errs := runTogether(
			func() error {
				_, err := repository.ConvertReservation(context.Background(), first)
				return err
			},
			func() error {
				_, err := repository.ConvertReservation(context.Background(), second)
				return err
			},
		)
		if successes(errs) != 1 || !anyIs(errs, domain.ErrReservationConverted) {
			t.Fatalf("results = %v, want one venta and one already-converted conflict", errs)
		}
		if countSalesOf(t, pool, reservation.ID) != 1 {
			t.Errorf("ventas of the reserva = %d, want 1", countSalesOf(t, pool, reservation.ID))
		}
	})

	t.Run("two retries with the same key", func(t *testing.T) {
		setup := newConversionSetup(t, pool)
		now := time.Now().UTC().Truncate(time.Microsecond)
		reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, now.Add(-time.Hour))
		repository := postgres.NewSaleRepository(pool, fixedReservationClock{now: now})
		command := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
		convert := func() error {
			_, err := repository.ConvertReservation(context.Background(), command)
			return err
		}
		if errs := runTogether(convert, convert); successes(errs) != 2 {
			t.Fatalf("results = %v, want both to get the same venta", errs)
		}
		if countSalesOf(t, pool, reservation.ID) != 1 {
			t.Errorf("ventas of the reserva = %d, want 1", countSalesOf(t, pool, reservation.ID))
		}
	})

	t.Run("conversion against cancellation", func(t *testing.T) {
		setup := newConversionSetup(t, pool)
		now := time.Now().UTC().Truncate(time.Microsecond)
		reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, now.Add(-time.Hour))
		command := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
		errs := runTogether(
			func() error {
				_, err := postgres.NewSaleRepository(pool, fixedReservationClock{now: now}).ConvertReservation(context.Background(), command)
				return err
			},
			func() error {
				_, err := postgres.NewReservationRepository(pool, fixedReservationClock{now: now}).Cancel(context.Background(), gateway.CancelReservationCommand{
					ReservationID: reservation.ID, ActorID: setup.adminID, Reason: "Cliente desistió",
				}, gateway.ReservationScope{})
				return err
			},
		)
		if successes(errs) != 1 {
			t.Fatalf("results = %v, want exactly one winner", errs)
		}
		state := readState(t, pool, "reservas", reservation.ID)
		lot := readState(t, pool, "lotes", setup.lotID)
		switch {
		case errs[0] == nil && (state != string(domain.ReservationStateConverted) || lot != string(domain.LotStateCompleted) || !errors.Is(errs[1], domain.ErrReservationConverted)):
			t.Errorf("conversion won but reserva=%q lote=%q cancel=%v", state, lot, errs[1])
		case errs[1] == nil && (state != string(domain.ReservationStateCancelled) || lot != string(domain.LotStateAvailable) || !errors.Is(errs[0], domain.ErrReservationAlreadyCancelled) || countSalesOf(t, pool, reservation.ID) != 0):
			t.Errorf("cancellation won but reserva=%q lote=%q convert=%v", state, lot, errs[0])
		}
	})

	t.Run("conversion against the expiry worker", func(t *testing.T) {
		setup := newConversionSetup(t, pool)
		reservedAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
		beforeDue := reservedAt.Add(domain.ReservationDuration - time.Minute)
		afterDue := reservedAt.Add(domain.ReservationDuration + time.Minute)
		reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, reservedAt)
		command := conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID)
		var report domain.ExpirationReport
		errs := runTogether(
			func() error {
				_, err := postgres.NewSaleRepository(pool, fixedReservationClock{now: beforeDue}).ConvertReservation(context.Background(), command)
				return err
			},
			func() error {
				var err error
				report, err = postgres.NewReservationRepository(pool, fixedReservationClock{now: afterDue}).ExpireDue(context.Background(), afterDue, 1000)
				return err
			},
		)
		if errs[1] != nil || len(report.Failures) != 0 {
			t.Fatalf("worker = %v / %#v", errs[1], report.Failures)
		}
		state := readState(t, pool, "reservas", reservation.ID)
		lot := readState(t, pool, "lotes", setup.lotID)
		if errs[0] == nil {
			if state != string(domain.ReservationStateConverted) || lot != string(domain.LotStateCompleted) {
				t.Errorf("conversion won but reserva=%q lote=%q", state, lot)
			}
		} else if !errors.Is(errs[0], domain.ErrReservationConversionExpired) || state != string(domain.ReservationStateExpired) || lot != string(domain.LotStateAvailable) {
			t.Errorf("worker won but convert=%v reserva=%q lote=%q", errs[0], state, lot)
		}
	})
}

func runTogether(operations ...func() error) []error {
	errs := make([]error, len(operations))
	start := make(chan struct{})
	var group sync.WaitGroup
	for i, operation := range operations {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			errs[i] = operation()
		}()
	}
	close(start)
	group.Wait()
	return errs
}

func successes(errs []error) int {
	count := 0
	for _, err := range errs {
		if err == nil {
			count++
		}
	}
	return count
}

func anyIs(errs []error, target error) bool {
	for _, err := range errs {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func TestSaleReservaLinkConstraints(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, now.Add(-time.Hour))
	sale, err := postgres.NewSaleRepository(pool, fixedReservationClock{now: now}).
		ConvertReservation(context.Background(), conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID))
	if err != nil {
		t.Fatalf("ConvertReservation() error = %v", err)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE ventas SET reserva_id = NULL WHERE id = $1::uuid`, sale.ID); err == nil || !strings.Contains(err.Error(), "reserva_id is immutable") {
		t.Errorf("clearing the link error = %v, want the immutability guard", err)
	}

	insert := func(lotID, reservationID string) error {
		_, err := pool.Exec(context.Background(), `
			INSERT INTO ventas (lote_id, cliente_id, modalidad_pago, monto, moneda, vendedor_id, usuario_alta, reserva_id)
			VALUES ($1::uuid, $2::uuid, 'contado', 1, 'USD', $3::uuid, $3::uuid, $4::uuid)
		`, lotID, setup.clientID, setup.adminID, reservationID)
		return err
	}
	manzanaID, otherLotID := secondLotFixture(t, pool, setup.loteoID)
	otherReservation := setup.reserve(t, pool, otherLotID, setup.adminID, now.Add(-time.Hour))
	var thirdLotID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO lotes (manzana_id, loteo_id, numero, precio, moneda) VALUES ($1::uuid, $2::uuid, '3', 80000, 'USD') RETURNING id::text
	`, manzanaID, setup.loteoID).Scan(&thirdLotID); err != nil {
		t.Fatalf("create third lot: %v", err)
	}
	if err := insert(thirdLotID, otherReservation.ID); err == nil || !strings.Contains(err.Error(), "ventas_reserva_lote_fk") {
		t.Errorf("link to a reserva of another lote error = %v, want the composite foreign key", err)
	}

	// Cancelling the venta frees the lote's active-sale slot, yet the reserva
	// still can't originate a second one.
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO venta_estados (venta_id, estado, razon, usuario_modificacion) VALUES ($1::uuid, 'cancelada', 'Prueba', $2::uuid)
	`, sale.ID, setup.adminID); err != nil {
		t.Fatalf("cancel venta: %v", err)
	}
	if err := insert(setup.lotID, reservation.ID); err == nil || !strings.Contains(err.Error(), "ventas_reserva_id_idx") {
		t.Errorf("second venta of the reserva error = %v, want the unique link", err)
	}
}

func TestConvertReservationRequiresTheReservaThatHoldsTheLote(t *testing.T) {
	pool := conversionPool(t)
	setup := newConversionSetup(t, pool)
	reservedAt := time.Date(2027, 4, 1, 12, 0, 0, 0, time.UTC)
	holder := setup.reserve(t, pool, setup.lotID, setup.adminID, reservedAt)

	// The holder ends without releasing the lote, and a second reserva is
	// written straight to the table: the lote stays reservado by the first.
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO reserva_estados (reserva_id, estado, razon) VALUES ($1::uuid, 'vencida', 'Prueba')
	`, holder.ID); err != nil {
		t.Fatalf("end holder reserva: %v", err)
	}
	var orphanID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO reservas (lote_id, cliente_id, vendedor_id, usuario_alta, fecha_vencimiento, fecha_creacion, fecha_modificacion)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $3::uuid, $4, $5, $5)
		RETURNING id::text
	`, setup.lotID, setup.clientID, setup.adminID, reservedAt.Add(domain.ReservationDuration), reservedAt).Scan(&orphanID); err != nil {
		t.Fatalf("insert orphan reserva: %v", err)
	}

	now := reservedAt.Add(time.Hour)
	ownScope := gateway.ReservationScope{ActorAuthProviderID: &setup.adminAuthID}
	orphan, err := postgres.NewReservationRepository(pool, fixedReservationClock{now: now}).Get(context.Background(), orphanID, ownScope)
	if err != nil {
		t.Fatalf("Get() orphan error = %v", err)
	}
	if orphan.PuedeConvertir {
		t.Error("a reserva that doesn't hold the lote reads puedeConvertir = true")
	}
	_, err = postgres.NewSaleRepository(pool, fixedReservationClock{now: now}).
		ConvertReservation(context.Background(), conversionCommand(t, orphanID, setup.adminID, setup.adminAuthID))
	if !errors.Is(err, domain.ErrSaleLotUnavailable) {
		t.Fatalf("convert a reserva that doesn't hold the lote error = %v, want %v", err, domain.ErrSaleLotUnavailable)
	}
	if countSalesOf(t, pool, orphanID) != 0 {
		t.Error("a venta was registered for a reserva that doesn't hold the lote")
	}
}

// pausedConversion runs a conversion that stops right before its commit,
// holding every lock, until release is closed. ready reports the backend pid
// of its transaction once it is paused.
func pausedConversion(t *testing.T, pool *pgxpool.Pool, now time.Time, command gateway.ConvertReservationCommand) (ready <-chan int, release chan<- struct{}, result <-chan error) {
	t.Helper()
	readyCh := make(chan int, 1)
	releaseCh := make(chan struct{})
	resultCh := make(chan error, 1)
	repository := postgres.NewSaleRepository(pool, fixedReservationClock{now: now})
	postgres.SetAfterConversionWrite(repository, func(ctx context.Context, tx pgx.Tx) error {
		var pid int
		if err := tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
			return err
		}
		readyCh <- pid
		<-releaseCh
		return nil
	})
	go func() {
		_, err := repository.ConvertReservation(context.Background(), command)
		resultCh <- err
	}()
	return readyCh, releaseCh, resultCh
}

func TestConvertReservationHoldsCancellationAndWorkerUntilItCommits(t *testing.T) {
	pool := conversionPool(t)

	t.Run("cancellation waits and then sees the conversion", func(t *testing.T) {
		setup := newConversionSetup(t, pool)
		reservedAt := time.Date(2027, 5, 1, 12, 0, 0, 0, time.UTC)
		now := reservedAt.Add(time.Hour)
		reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, reservedAt)
		ready, release, converted := pausedConversion(t, pool, now,
			conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID))
		pid := <-ready

		cancelled := make(chan error, 1)
		go func() {
			_, err := postgres.NewReservationRepository(pool, fixedReservationClock{now: now}).Cancel(context.Background(), gateway.CancelReservationCommand{
				ReservationID: reservation.ID, ActorID: setup.adminID, Reason: "Cliente desistió",
			}, gateway.ReservationScope{})
			cancelled <- err
		}()
		waitUntilBlockedBy(t, pool, pid)
		close(release)

		if err := <-converted; err != nil {
			t.Fatalf("ConvertReservation() error = %v", err)
		}
		if err := <-cancelled; !errors.Is(err, domain.ErrReservationConverted) {
			t.Fatalf("Cancel() after waiting error = %v, want %v", err, domain.ErrReservationConverted)
		}
		if got := readState(t, pool, "lotes", setup.lotID); got != string(domain.LotStateCompleted) {
			t.Errorf("lot = %q, want finalizado", got)
		}
	})

	t.Run("a worker that found the reserva first skips it after waiting", func(t *testing.T) {
		setup := newConversionSetup(t, pool)
		reservedAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		beforeDue := reservedAt.Add(domain.ReservationDuration - time.Minute)
		afterDue := reservedAt.Add(domain.ReservationDuration + time.Minute)
		reservation := setup.reserve(t, pool, setup.lotID, setup.adminID, reservedAt)
		ready, release, converted := pausedConversion(t, pool, beforeDue,
			conversionCommand(t, reservation.ID, setup.adminID, setup.adminAuthID))
		pid := <-ready

		// The reserva is still activa for every other reader, so the worker
		// lists it as a candidate and then waits for the lote.
		expired := make(chan error, 1)
		go func() {
			_, err := postgres.NewReservationRepository(pool, fixedReservationClock{now: afterDue}).ExpireDue(context.Background(), afterDue, 1000)
			expired <- err
		}()
		waitUntilBlockedBy(t, pool, pid)
		close(release)

		if err := <-converted; err != nil {
			t.Fatalf("ConvertReservation() error = %v", err)
		}
		if err := <-expired; err != nil {
			t.Fatalf("ExpireDue() error = %v", err)
		}
		if state := readState(t, pool, "reservas", reservation.ID); state != string(domain.ReservationStateConverted) {
			t.Errorf("reserva = %q, want convertida", state)
		}
		if got := readState(t, pool, "lotes", setup.lotID); got != string(domain.LotStateCompleted) {
			t.Errorf("lot = %q, want finalizado: the worker must not release a converted reserva", got)
		}
	})
}
