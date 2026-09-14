package postgres_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

func TestReservationRepository(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	repository := postgres.NewReservationRepository(pool, fixedReservationClock{now: now})
	command := gateway.CreateReservationCommand{
		LoteoID:                loteoID,
		LoteID:                 lotID,
		ClienteID:              clientID,
		VendedorID:             actorID,
		ActorID:                actorID,
		ActorAuthProviderID:    newUUID(t),
		IdempotencyKey:         "reservation-create-" + newUUID(t),
		IdempotencyPayloadHash: strings.Repeat("a", 64),
		CreatedAt:              now,
	}

	created, err := repository.Create(context.Background(), command)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Estado != domain.ReservationStateActive || !created.FechaVencimiento.Equal(now.Add(domain.ReservationDuration)) {
		t.Errorf("created reservation = %#v", created)
	}

	retried, err := repository.Create(context.Background(), command)
	if err != nil {
		t.Fatalf("idempotent Create() error = %v", err)
	}
	if retried.ID != created.ID {
		t.Fatalf("idempotent Create() id = %q, want %q", retried.ID, created.ID)
	}

	conflicting := command
	conflicting.IdempotencyPayloadHash = strings.Repeat("b", 64)
	if _, err := repository.Create(context.Background(), conflicting); !errors.Is(err, domain.ErrReservationIdempotencyConflict) {
		t.Fatalf("conflicting Create() error = %v, want %v", err, domain.ErrReservationIdempotencyConflict)
	}
	activeConflict := command
	activeConflict.IdempotencyKey = "reservation-active-conflict-" + newUUID(t)
	activeConflict.IdempotencyPayloadHash = strings.Repeat("9", 64)
	if _, err := repository.Create(context.Background(), activeConflict); !errors.Is(err, domain.ErrReservationActiveConflict) {
		t.Fatalf("active conflict Create() error = %v, want %v", err, domain.ErrReservationActiveConflict)
	}

	page, err := repository.List(context.Background(), domain.ReservationListFilter{}, gateway.ReservationScope{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.Total < 1 || len(page.Items) < 1 {
		t.Fatalf("List() = %#v, want the created reservation", page)
	}

	searchPage, err := repository.List(context.Background(), domain.ReservationListFilter{Search: "Cliente", Limit: 1}, gateway.ReservationScope{})
	if err != nil {
		t.Fatalf("List() search error = %v", err)
	}
	if searchPage.Total < 1 || len(searchPage.Items) != 1 {
		t.Fatalf("List() search = %#v, want one matching reservation", searchPage)
	}

	lotPage, err := repository.List(context.Background(), domain.ReservationListFilter{LoteID: lotID}, gateway.ReservationScope{})
	if err != nil {
		t.Fatalf("List() lot filter error = %v", err)
	}
	if lotPage.Total < 1 || len(lotPage.Items) < 1 || lotPage.Items[0].LoteID != lotID {
		t.Fatalf("List() lot filter = %#v, want the created reservation", lotPage)
	}

	outsidePage, err := repository.List(context.Background(), domain.ReservationListFilter{Page: 99}, gateway.ReservationScope{})
	if err != nil {
		t.Fatalf("List() out-of-range error = %v", err)
	}
	if len(outsidePage.Items) != 0 || outsidePage.Total < 1 || outsidePage.TotalPages < 1 {
		t.Fatalf("List() out-of-range = %#v, want totals preserved", outsidePage)
	}

	sellers, err := repository.ListEligibleSellers(context.Background(), loteoID, gateway.ReservationScope{})
	if err != nil {
		t.Fatalf("ListEligibleSellers() error = %v", err)
	}
	if !containsSeller(sellers, actorID) {
		t.Fatalf("ListEligibleSellers() = %#v, want the fixture administrator", sellers)
	}
	if _, err := repository.ListEligibleSellers(context.Background(), "not-a-uuid", gateway.ReservationScope{}); !errors.Is(err, domain.ErrLoteoNotFound) {
		t.Fatalf("ListEligibleSellers() invalid loteo error = %v", err)
	}

	detailed, err := repository.Get(context.Background(), created.ID, gateway.ReservationScope{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(detailed.Historial) != 1 || detailed.Historial[0].Estado != domain.ReservationStateActive {
		t.Fatalf("initial history = %#v", detailed.Historial)
	}

	canceled, err := repository.Cancel(context.Background(), gateway.CancelReservationCommand{
		ReservationID: created.ID,
		ActorID:       actorID,
		Reason:        "Cliente desistió",
		CancelledAt:   now.Add(time.Hour),
	}, gateway.ReservationScope{})
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if canceled.Estado != domain.ReservationStateCancelled || len(canceled.Historial) != 2 {
		t.Fatalf("canceled reservation = %#v", canceled)
	}

	repeated, err := repository.Cancel(context.Background(), gateway.CancelReservationCommand{
		ReservationID: created.ID,
		ActorID:       actorID,
		Reason:        "Otra razón que no debe reemplazar la original",
		CancelledAt:   now.Add(2 * time.Hour),
	}, gateway.ReservationScope{})
	if err != nil {
		t.Fatalf("repeated Cancel() error = %v", err)
	}
	if repeated.ID != created.ID || len(repeated.Historial) != 2 || repeated.Historial[1].Razon != "Cliente desistió" {
		t.Fatalf("repeated cancellation = %#v", repeated)
	}

	if _, err := pool.Exec(context.Background(), `
		UPDATE lotes SET numero = NULL, precio = NULL WHERE id = $1::uuid
	`, lotID); err != nil {
		t.Fatalf("clear lot commercial data: %v", err)
	}
	_, err = repository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID:                loteoID,
		LoteID:                 lotID,
		ClienteID:              clientID,
		VendedorID:             actorID,
		ActorID:                actorID,
		ActorAuthProviderID:    newUUID(t),
		IdempotencyKey:         "reservation-incomplete-" + newUUID(t),
		IdempotencyPayloadHash: strings.Repeat("d", 64),
		CreatedAt:              now,
	})
	if !errors.Is(err, domain.ErrReservationLotIncomplete) {
		t.Fatalf("Create() incomplete lot error = %v, want %v", err, domain.ErrReservationLotIncomplete)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE lotes SET numero = '1', precio = 0 WHERE id = $1::uuid
	`, lotID); err != nil {
		t.Fatalf("set zero lot price: %v", err)
	}
	zeroPriceCommand := command
	zeroPriceCommand.IdempotencyKey = "reservation-zero-price-" + newUUID(t)
	zeroPriceCommand.IdempotencyPayloadHash = strings.Repeat("e", 64)
	if _, err := repository.Create(context.Background(), zeroPriceCommand); !errors.Is(err, domain.ErrReservationLotIncomplete) {
		t.Fatalf("Create() zero price error = %v, want %v", err, domain.ErrReservationLotIncomplete)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE lotes SET numero = '1', precio = 100000 WHERE id = $1::uuid
	`, lotID); err != nil {
		t.Fatalf("restore lot commercial data: %v", err)
	}

	expiringRepository := postgres.NewReservationRepository(pool, fixedReservationClock{now: now.Add(-domain.ReservationDuration)})
	expiring, err := expiringRepository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID:                loteoID,
		LoteID:                 lotID,
		ClienteID:              clientID,
		VendedorID:             actorID,
		ActorID:                actorID,
		ActorAuthProviderID:    newUUID(t),
		IdempotencyKey:         "reservation-expire-" + newUUID(t),
		IdempotencyPayloadHash: strings.Repeat("c", 64),
		CreatedAt:              now.Add(-domain.ReservationDuration),
	})
	if err != nil {
		t.Fatalf("Create() for expiration error = %v", err)
	}

	report, err := repository.ExpireDue(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("ExpireDue() error = %v", err)
	}
	if report.Processed != 1 || len(report.Failures) != 0 {
		t.Fatalf("expiration report = %#v", report)
	}
	expired, err := repository.Get(context.Background(), expiring.ID, gateway.ReservationScope{})
	if err != nil {
		t.Fatalf("Get() expired error = %v", err)
	}
	if expired.Estado != domain.ReservationStateExpired || len(expired.Historial) != 2 {
		t.Fatalf("expired reservation = %#v", expired)
	}
}

type fixedReservationClock struct{ now time.Time }

func (clock fixedReservationClock) Now() time.Time { return clock.now }

type mutableReservationClock struct{ now time.Time }

func (clock *mutableReservationClock) Now() time.Time { return clock.now }

func containsSeller(sellers []domain.SellerOption, id string) bool {
	for _, seller := range sellers {
		if seller.ID == id {
			return true
		}
	}
	return false
}

func TestReservationCancelUsesThePostLockClock(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := &mutableReservationClock{now: now.Add(-domain.ReservationDuration)}
	repository := postgres.NewReservationRepository(pool, clock)
	reservation, err := repository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID:                loteoID,
		LoteID:                 lotID,
		ClienteID:              clientID,
		VendedorID:             actorID,
		ActorID:                actorID,
		ActorAuthProviderID:    newUUID(t),
		IdempotencyKey:         "reservation-cancel-expired-" + newUUID(t),
		IdempotencyPayloadHash: strings.Repeat("e", 64),
		CreatedAt:              clock.Now(),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	clock.now = now
	if _, err := repository.Cancel(context.Background(), gateway.CancelReservationCommand{
		ReservationID: reservation.ID,
		ActorID:       actorID,
		Reason:        "Cliente desistió",
		CancelledAt:   now.Add(-domain.ReservationDuration),
	}, gateway.ReservationScope{}); !errors.Is(err, domain.ErrReservationExpired) {
		t.Fatalf("Cancel() expired error = %v, want %v", err, domain.ErrReservationExpired)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT estado_actual FROM reservas WHERE id = $1::uuid`, reservation.ID).Scan(&state); err != nil {
		t.Fatalf("read expired reservation: %v", err)
	}
	if state != string(domain.ReservationStateExpired) {
		t.Fatalf("expired state = %q", state)
	}
}

func TestReservationExpiryRegularizesDeletedLot(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := &mutableReservationClock{now: now.Add(-domain.ReservationDuration)}
	repository := postgres.NewReservationRepository(pool, clock)
	reservation, err := repository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID:                loteoID,
		LoteID:                 lotID,
		ClienteID:              clientID,
		VendedorID:             actorID,
		ActorID:                actorID,
		ActorAuthProviderID:    newUUID(t),
		IdempotencyKey:         "reservation-detached-" + newUUID(t),
		IdempotencyPayloadHash: strings.Repeat("f", 64),
		CreatedAt:              clock.Now(),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE lotes SET fecha_baja = now() WHERE id = $1::uuid`, lotID); err != nil {
		t.Fatalf("soft-delete lot: %v", err)
	}
	clock.now = now
	report, err := repository.ExpireDue(context.Background(), now, 1)
	if err != nil {
		t.Fatalf("ExpireDue() error = %v", err)
	}
	if report.Processed != 1 || len(report.Failures) != 0 {
		t.Fatalf("expiration report = %#v", report)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT estado_actual FROM reservas WHERE id = $1::uuid`, reservation.ID).Scan(&state); err != nil {
		t.Fatalf("read detached reservation: %v", err)
	}
	if state != string(domain.ReservationStateExpired) {
		t.Fatalf("detached state = %q", state)
	}
}

func TestReservationExpiryRegularizesLotOutsideReservedState(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := &mutableReservationClock{now: now.Add(-domain.ReservationDuration)}
	repository := postgres.NewReservationRepository(pool, clock)
	reservation, err := repository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: actorID, ActorID: actorID, ActorAuthProviderID: newUUID(t),
		IdempotencyKey: "reservation-state-drift-" + newUUID(t), IdempotencyPayloadHash: strings.Repeat("7", 64),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO lote_estados (lote_id, estado, origen, razon)
		VALUES ($1::uuid, 'disponible', 'sistema', 'Simulación de desincronización')
	`, lotID); err != nil {
		t.Fatalf("move lot outside reserved state: %v", err)
	}

	clock.now = now
	report, err := repository.ExpireDue(context.Background(), now, 1)
	if err != nil {
		t.Fatalf("ExpireDue() error = %v", err)
	}
	if report.Processed != 1 || report.Skipped != 0 || len(report.Failures) != 0 {
		t.Fatalf("expiration report = %#v", report)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT estado_actual FROM reservas WHERE id = $1::uuid`, reservation.ID).Scan(&state); err != nil {
		t.Fatalf("read regularized reservation: %v", err)
	}
	if state != string(domain.ReservationStateExpired) {
		t.Fatalf("regularized reservation state = %q", state)
	}
}

func TestReservationAgencyMustRemainAssignedDuringCreate(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	_, clientID, loteoID, lotID := reservationFixture(t, pool)
	var agencyID string
	if err := pool.QueryRow(context.Background(), `INSERT INTO inmobiliarias (razon_social) VALUES ($1) RETURNING id::text`, "Reservation Agency "+newUUID(t)).Scan(&agencyID); err != nil {
		t.Fatalf("create reservation agency: %v", err)
	}
	authProviderID := newUUID(t)
	var actorID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO usuarios (auth_provider_id, email, rol, nombre, apellido, inmobiliaria_id, perfil_completo)
		VALUES ($1::uuid, $2, 'inmobiliaria', 'Agencia', 'Reserva', $3::uuid, true)
		RETURNING id::text
	`, authProviderID, newEmail(t), agencyID).Scan(&actorID); err != nil {
		t.Fatalf("create reservation agency actor: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO inmobiliaria_loteos (inmobiliaria_id, loteo_id) VALUES ($1::uuid, $2::uuid)`, agencyID, loteoID); err != nil {
		t.Fatalf("assign reservation agency: %v", err)
	}
	sameAgencyAuthProviderID := newUUID(t)
	var sameAgencyUserID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO usuarios (auth_provider_id, email, rol, nombre, apellido, inmobiliaria_id, perfil_completo)
		VALUES ($1::uuid, $2, 'inmobiliaria', 'Otro', 'Agente', $3::uuid, true)
		RETURNING id::text
	`, sameAgencyAuthProviderID, newEmail(t), agencyID).Scan(&sameAgencyUserID); err != nil {
		t.Fatalf("create second reservation agency actor: %v", err)
	}
	t.Cleanup(func() { deleteUsuario(t, pool, sameAgencyAuthProviderID) })
	t.Cleanup(func() {
		tx, err := pool.Begin(context.Background())
		if err != nil {
			t.Errorf("begin reservation agency cleanup: %v", err)
			return
		}
		defer tx.Rollback(context.Background())
		if _, err := tx.Exec(context.Background(), `
			ALTER TABLE lote_estados DISABLE TRIGGER lote_estados_reject_mutation;
			ALTER TABLE reserva_estados DISABLE TRIGGER reserva_estados_reject_mutation;
		`); err != nil {
			t.Errorf("cleanup reservation agency: %v", err)
			return
		}
		cleanupStatements := []struct {
			query string
			args  []any
		}{
			{`DELETE FROM lote_estados WHERE reserva_id IN (SELECT id FROM reservas WHERE vendedor_id = $1::uuid)`, []any{actorID}},
			{`DELETE FROM reserva_estados WHERE reserva_id IN (SELECT id FROM reservas WHERE vendedor_id = $1::uuid)`, []any{actorID}},
			{`DELETE FROM reservas WHERE vendedor_id = $1::uuid`, []any{actorID}},
			{`UPDATE lotes SET usuario_modificacion = NULL WHERE usuario_modificacion = $1::uuid`, []any{sameAgencyUserID}},
			{`DELETE FROM inmobiliaria_loteos WHERE inmobiliaria_id = $1::uuid AND loteo_id = $2::uuid`, []any{agencyID, loteoID}},
			{`UPDATE usuarios SET fecha_baja = now() WHERE id = $1::uuid`, []any{actorID}},
			{`UPDATE inmobiliarias SET fecha_baja = now() WHERE id = $1::uuid`, []any{agencyID}},
		}
		for _, statement := range cleanupStatements {
			if _, err := tx.Exec(context.Background(), statement.query, statement.args...); err != nil {
				t.Errorf("cleanup reservation agency: %v", err)
				return
			}
		}
		if _, err := tx.Exec(context.Background(), `
			ALTER TABLE lote_estados ENABLE TRIGGER lote_estados_reject_mutation;
			ALTER TABLE reserva_estados ENABLE TRIGGER reserva_estados_reject_mutation;
		`); err != nil {
			t.Errorf("cleanup reservation agency: %v", err)
			return
		}
		if err := tx.Commit(context.Background()); err != nil {
			t.Errorf("commit reservation agency cleanup: %v", err)
		}
	})

	now := time.Now().UTC().Truncate(time.Microsecond)
	repository := postgres.NewReservationRepository(pool, fixedReservationClock{now: now})
	created, err := repository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID:                loteoID,
		LoteID:                 lotID,
		ClienteID:              clientID,
		VendedorID:             actorID,
		ActorID:                actorID,
		ActorAuthProviderID:    authProviderID,
		SellerIsActor:          true,
		IdempotencyKey:         "reservation-agency-" + newUUID(t),
		IdempotencyPayloadHash: strings.Repeat("1", 64),
		CreatedAt:              now,
	})
	if err != nil {
		t.Fatalf("agency Create() error = %v", err)
	}
	if created.Vendedor.ID != actorID || created.Estado != domain.ReservationStateActive {
		t.Fatalf("agency reservation = %#v", created)
	}
	if created.Inmobiliaria == nil || created.Inmobiliaria.ID != agencyID || !created.PuedeCancelar {
		t.Fatalf("reservation agency and actor permission = %#v, want agency %q and permission true", created, agencyID)
	}

	if _, err := pool.Exec(context.Background(), `
		UPDATE inmobiliaria_loteos SET fecha_baja = now()
		WHERE inmobiliaria_id = $1::uuid AND loteo_id = $2::uuid
	`, agencyID, loteoID); err != nil {
		t.Fatalf("remove current agency assignment: %v", err)
	}
	scope := gateway.ReservationScope{
		AssigneeAuthProviderID: &sameAgencyAuthProviderID,
		ByAgencyAssignment:     true,
		ActorAuthProviderID:    &sameAgencyAuthProviderID,
	}
	page, err := repository.List(context.Background(), domain.ReservationListFilter{}, scope)
	if err != nil {
		t.Fatalf("list reservation by persisted agency: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != created.ID || !page.Items[0].PuedeCancelar {
		t.Fatalf("agency reservation page = %#v, want created reservation and cancellation permission", page)
	}
	detailed, err := repository.Get(context.Background(), created.ID, scope)
	if err != nil || !detailed.PuedeCancelar || detailed.Inmobiliaria == nil || detailed.Inmobiliaria.ID != agencyID {
		t.Fatalf("agency reservation detail = %#v, %v, want persistent agency and cancellation permission", detailed, err)
	}

	if _, err := repository.Cancel(context.Background(), gateway.CancelReservationCommand{
		ReservationID: created.ID,
		ActorID:       sameAgencyUserID,
		Reason:        "Cliente desistió",
	}, scope); err != nil {
		t.Fatalf("cancel reservation for persisted agency: %v", err)
	}
}

func TestReservationCreateRejectsInvalidReferences(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	repository := postgres.NewReservationRepository(pool, fixedReservationClock{now: now})
	base := gateway.CreateReservationCommand{
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: actorID, ActorID: actorID, ActorAuthProviderID: newUUID(t),
		IdempotencyPayloadHash: strings.Repeat("2", 64), CreatedAt: now,
	}
	attempt := func(name string, command gateway.CreateReservationCommand, want error) {
		t.Helper()
		command.IdempotencyKey = "reservation-invalid-" + name + "-" + newUUID(t)
		if _, err := repository.Create(context.Background(), command); !errors.Is(err, want) {
			t.Errorf("Create(%s) error = %v, want %v", name, err, want)
		}
	}

	invalidLoteo := base
	invalidLoteo.LoteoID = ""
	attempt("empty-loteo", invalidLoteo, domain.ErrLoteNotFound)
	missingLoteo := base
	missingLoteo.LoteoID = newUUID(t)
	attempt("missing-loteo", missingLoteo, domain.ErrLoteNotFound)
	missingLot := base
	missingLot.LoteID = newUUID(t)
	attempt("missing-lot", missingLot, domain.ErrLoteNotFound)
	missingActor := base
	missingActor.ActorID = newUUID(t)
	attempt("missing-actor", missingActor, domain.ErrActorNoAprovisionado)
	missingClient := base
	missingClient.ClienteID = newUUID(t)
	attempt("missing-client", missingClient, domain.ErrReservationInvalidClient)
	missingSeller := base
	missingSeller.VendedorID = newUUID(t)
	attempt("missing-seller", missingSeller, domain.ErrReservationSellerNotEligible)
	actorSeller := base
	actorSeller.SellerIsActor = true
	attempt("administrative-actor-seller", actorSeller, domain.ErrReservationSellerNotEligible)

	if _, err := pool.Exec(context.Background(), `UPDATE usuarios SET fecha_baja = now() WHERE id = $1::uuid`, actorID); err != nil {
		t.Fatalf("deactivate reservation actor: %v", err)
	}
	inactiveActor := base
	attempt("inactive-actor", inactiveActor, domain.ErrCuentaInactiva)
	if _, err := pool.Exec(context.Background(), `UPDATE usuarios SET fecha_baja = NULL WHERE id = $1::uuid`, actorID); err != nil {
		t.Fatalf("restore reservation actor: %v", err)
	}

	sellerAuthID := newUUID(t)
	var inactiveSellerID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO usuarios (auth_provider_id, email, rol, nombre, apellido, fecha_baja)
		VALUES ($1::uuid, $2, 'administrador', 'Inactivo', 'Reserva', now()) RETURNING id::text
	`, sellerAuthID, newEmail(t)).Scan(&inactiveSellerID); err != nil {
		t.Fatalf("create inactive reservation seller: %v", err)
	}
	t.Cleanup(func() { deleteUsuario(t, pool, sellerAuthID) })
	inactiveSeller := base
	inactiveSeller.VendedorID = inactiveSellerID
	attempt("inactive-seller", inactiveSeller, domain.ErrReservationSellerNotEligible)

	roleSellerAuthID := newUUID(t)
	var roleSellerID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO usuarios (auth_provider_id, email, rol, nombre, apellido)
		VALUES ($1::uuid, $2, 'agrimensor', 'Agrimensor', 'Reserva') RETURNING id::text
	`, roleSellerAuthID, newEmail(t)).Scan(&roleSellerID); err != nil {
		t.Fatalf("create ineligible reservation seller: %v", err)
	}
	t.Cleanup(func() { deleteUsuario(t, pool, roleSellerAuthID) })
	ineligibleSeller := base
	ineligibleSeller.VendedorID = roleSellerID
	attempt("ineligible-seller", ineligibleSeller, domain.ErrReservationSellerNotEligible)
}

func TestReservationCreateRegularizesExpiredActiveReservation(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	clock := &mutableReservationClock{now: now.Add(-domain.ReservationDuration)}
	repository := postgres.NewReservationRepository(pool, clock)
	expired, err := repository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: actorID, ActorID: actorID, ActorAuthProviderID: newUUID(t),
		IdempotencyKey: "reservation-expired-before-" + newUUID(t), IdempotencyPayloadHash: strings.Repeat("3", 64), CreatedAt: clock.Now(),
	})
	if err != nil {
		t.Fatalf("initial Create() error = %v", err)
	}
	clock.now = now
	replacement, err := repository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: actorID, ActorID: actorID, ActorAuthProviderID: newUUID(t),
		IdempotencyKey: "reservation-replacement-" + newUUID(t), IdempotencyPayloadHash: strings.Repeat("4", 64), CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("replacement Create() error = %v", err)
	}
	if replacement.Estado != domain.ReservationStateActive || replacement.ID == expired.ID {
		t.Fatalf("replacement reservation = %#v", replacement)
	}
	var state string
	if err := pool.QueryRow(context.Background(), `SELECT estado_actual FROM reservas WHERE id = $1::uuid`, expired.ID).Scan(&state); err != nil {
		t.Fatalf("read regularized reservation: %v", err)
	}
	if state != string(domain.ReservationStateExpired) {
		t.Fatalf("regularized state = %q", state)
	}
}

func TestReservationRepositoryHandlesMissingRecordsAndInvalidFilters(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	repository := postgres.NewReservationRepository(pool)
	ctx := context.Background()
	if _, err := repository.Get(ctx, newUUID(t), gateway.ReservationScope{}); !errors.Is(err, domain.ErrReservationNotFound) {
		t.Fatalf("Get() missing error = %v", err)
	}
	if _, err := repository.Get(ctx, "not-a-uuid", gateway.ReservationScope{}); !errors.Is(err, domain.ErrReservationNotFound) {
		t.Fatalf("Get() malformed error = %v", err)
	}
	if _, err := repository.Cancel(ctx, gateway.CancelReservationCommand{
		ReservationID: newUUID(t), ActorID: newUUID(t), Reason: "Cliente desistió",
	}, gateway.ReservationScope{}); !errors.Is(err, domain.ErrReservationNotFound) {
		t.Fatalf("Cancel() missing error = %v", err)
	}
	if _, err := repository.List(ctx, domain.ReservationListFilter{Page: -1}, gateway.ReservationScope{}); !errors.Is(err, domain.ErrReservationInvalidPage) {
		t.Fatalf("List() invalid page error = %v", err)
	}
	report, err := repository.ExpireDue(ctx, time.Now().UTC().Add(-365*24*time.Hour), 1)
	if err != nil {
		t.Fatalf("ExpireDue() empty error = %v", err)
	}
	if report.Candidates != 0 || report.Processed != 0 || report.Skipped != 0 {
		t.Fatalf("ExpireDue() empty report = %#v", report)
	}
}

func TestReservationCancelRejectsUnauthorizedAndExpiredReservations(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	repository := postgres.NewReservationRepository(pool, fixedReservationClock{now: now})
	reservation, err := repository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: actorID, ActorID: actorID, ActorAuthProviderID: newUUID(t),
		IdempotencyKey: "reservation-unauthorized-" + newUUID(t), IdempotencyPayloadHash: strings.Repeat("5", 64), CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	authProviderID := newUUID(t)
	var unauthorizedID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO usuarios (auth_provider_id, email, rol, nombre, apellido)
		VALUES ($1::uuid, $2, 'agrimensor', 'Agrimensor', 'Reserva') RETURNING id::text
	`, authProviderID, newEmail(t)).Scan(&unauthorizedID); err != nil {
		t.Fatalf("create unauthorized actor: %v", err)
	}
	t.Cleanup(func() { deleteUsuario(t, pool, authProviderID) })
	if _, err := repository.Cancel(context.Background(), gateway.CancelReservationCommand{
		ReservationID: reservation.ID, ActorID: unauthorizedID, Reason: "No corresponde",
	}, gateway.ReservationScope{}); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("unauthorized Cancel() error = %v", err)
	}
	if _, err := repository.Cancel(context.Background(), gateway.CancelReservationCommand{
		ReservationID: reservation.ID, ActorID: actorID, Reason: "Cliente desistió",
	}, gateway.ReservationScope{}); err != nil {
		t.Fatalf("cleanup cancellation error = %v", err)
	}

	clock := &mutableReservationClock{now: now.Add(-domain.ReservationDuration)}
	expiringRepository := postgres.NewReservationRepository(pool, clock)
	expiring, err := expiringRepository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: actorID, ActorID: actorID, ActorAuthProviderID: newUUID(t),
		IdempotencyKey: "reservation-cancel-expired-" + newUUID(t), IdempotencyPayloadHash: strings.Repeat("6", 64), CreatedAt: clock.Now(),
	})
	if err != nil {
		t.Fatalf("expired Create() error = %v", err)
	}
	clock.now = now
	if _, err := expiringRepository.Cancel(context.Background(), gateway.CancelReservationCommand{
		ReservationID: expiring.ID, ActorID: actorID, Reason: "Cliente desistió",
		CancelledAt: now.Add(-domain.ReservationDuration),
	}, gateway.ReservationScope{}); !errors.Is(err, domain.ErrReservationExpired) {
		t.Fatalf("expired Cancel() error = %v", err)
	}
}

func TestReservationCancelRejectsConvertedReservation(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	actorID, clientID, loteoID, lotID := reservationFixture(t, pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	repository := postgres.NewReservationRepository(pool, fixedReservationClock{now: now})
	reservation, err := repository.Create(context.Background(), gateway.CreateReservationCommand{
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: actorID, ActorID: actorID, ActorAuthProviderID: newUUID(t),
		IdempotencyKey: "reservation-converted-" + newUUID(t), IdempotencyPayloadHash: strings.Repeat("7", 64), CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO reserva_estados (reserva_id, estado, razon, usuario_modificacion)
		VALUES ($1::uuid, 'convertida', 'Venta confirmada', $2::uuid)
	`, reservation.ID, actorID); err != nil {
		t.Fatalf("convert reservation: %v", err)
	}
	if _, err := repository.Cancel(context.Background(), gateway.CancelReservationCommand{
		ReservationID: reservation.ID, ActorID: actorID, Reason: "Cliente desistió",
	}, gateway.ReservationScope{}); !errors.Is(err, domain.ErrReservationConverted) {
		t.Fatalf("converted Cancel() error = %v, want %v", err, domain.ErrReservationConverted)
	}
}

func TestReservationAgencyCannotCreateOutsideAssignedLoteo(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	adminID, clientID, loteoID, lotID := reservationFixture(t, pool)
	var agencyID string
	if err := pool.QueryRow(context.Background(), `INSERT INTO inmobiliarias (razon_social) VALUES ($1) RETURNING id::text`, "Unassigned Reservation Agency "+newUUID(t)).Scan(&agencyID); err != nil {
		t.Fatalf("create reservation agency: %v", err)
	}
	authProviderID := newUUID(t)
	var agencyUserID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO usuarios (auth_provider_id, email, rol, nombre, apellido, inmobiliaria_id, perfil_completo)
		VALUES ($1::uuid, $2, 'inmobiliaria', 'Agencia', 'Sin Asignación', $3::uuid, true)
		RETURNING id::text
	`, authProviderID, newEmail(t), agencyID).Scan(&agencyUserID); err != nil {
		t.Fatalf("create unassigned reservation actor: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM inmobiliarias WHERE id = $1::uuid`, agencyID); err != nil {
			t.Errorf("cleanup unassigned reservation agency: %v", err)
		}
	})
	t.Cleanup(func() { deleteUsuario(t, pool, authProviderID) })

	now := time.Now().UTC().Truncate(time.Microsecond)
	repository := postgres.NewReservationRepository(pool, fixedReservationClock{now: now})
	agencyCommand := gateway.CreateReservationCommand{
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: agencyUserID, ActorID: agencyUserID,
		ActorAuthProviderID: authProviderID, SellerIsActor: true, IdempotencyKey: "reservation-unassigned-actor-" + newUUID(t),
		IdempotencyPayloadHash: strings.Repeat("8", 64), CreatedAt: now,
	}
	if _, err := repository.Create(context.Background(), agencyCommand); !errors.Is(err, domain.ErrLoteNotFound) {
		t.Fatalf("unassigned agency Create() error = %v, want %v", err, domain.ErrLoteNotFound)
	}
	adminCommand := agencyCommand
	adminCommand.ActorID = adminID
	adminCommand.VendedorID = agencyUserID
	adminCommand.ActorAuthProviderID = newUUID(t)
	adminCommand.SellerIsActor = false
	adminCommand.IdempotencyKey = "reservation-unassigned-seller-" + newUUID(t)
	adminCommand.IdempotencyPayloadHash = strings.Repeat("a", 64)
	if _, err := repository.Create(context.Background(), adminCommand); !errors.Is(err, domain.ErrReservationSellerNotEligible) {
		t.Fatalf("unassigned seller Create() error = %v, want %v", err, domain.ErrReservationSellerNotEligible)
	}
}

func reservationFixture(t *testing.T, pool *pgxpool.Pool) (actorID, clientID, loteoID, lotID string) {
	t.Helper()
	authProviderID := newUUID(t)
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO usuarios (auth_provider_id, email, rol, nombre, apellido, perfil_completo)
		VALUES ($1::uuid, $2, 'administrador', 'Reserva', 'Test', true)
		RETURNING id::text
	`, authProviderID, newEmail(t)).Scan(&actorID); err != nil {
		t.Fatalf("create reservation actor: %v", err)
	}
	t.Cleanup(func() { deleteUsuario(t, pool, authProviderID) })

	if err := pool.QueryRow(context.Background(), `
		INSERT INTO loteos (nombre) VALUES ($1) RETURNING id::text
	`, "Reservation Test "+newUUID(t)).Scan(&loteoID); err != nil {
		t.Fatalf("create reservation loteo: %v", err)
	}
	var manzanaID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO manzanas (loteo_id, numero) VALUES ($1::uuid, '1') RETURNING id::text
	`, loteoID).Scan(&manzanaID); err != nil {
		t.Fatalf("create reservation manzana: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO lotes (manzana_id, loteo_id, numero, precio, moneda) VALUES ($1::uuid, $2::uuid, '1', 100000, 'USD') RETURNING id::text
	`, manzanaID, loteoID).Scan(&lotID); err != nil {
		t.Fatalf("create reservation lot: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO clientes (nombre, apellido, dni) VALUES ('Reserva', 'Cliente', $1) RETURNING id::text
	`, newUUID(t)).Scan(&clientID); err != nil {
		t.Fatalf("create reservation client: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM clientes WHERE id = $1::uuid`, clientID); err != nil {
			t.Errorf("cleanup reservation client: %v", err)
		}
	})
	t.Cleanup(func() { deleteLoteo(t, pool, loteoID) })

	return actorID, clientID, loteoID, lotID
}
