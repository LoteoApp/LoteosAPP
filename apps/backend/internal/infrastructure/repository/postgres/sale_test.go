package postgres_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

// TestSaleRepository is an integration test: it needs a real PostgreSQL
// instance with migrations applied and is skipped when DATABASE_URL is not
// set.
func TestSaleRepository(t *testing.T) {
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
	repository := postgres.NewSaleRepository(pool)
	command := gateway.CreateSaleCommand{
		DevelopmentID:          loteoID,
		LotID:                  lotID,
		ClientID:               clientID,
		SellerID:               actorID,
		ActorID:                actorID,
		PaymentMethod:          domain.PaymentMethodCash,
		IdempotencyKey:         newUUID(t),
		IdempotencyPayloadHash: saleHash(t),
		CreatedAt:              now,
	}

	created, err := repository.Create(context.Background(), command)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Estado != domain.SaleStateCompleted || created.Monto != 100000 || created.Moneda != "USD" {
		t.Errorf("created sale = %#v", created)
	}
	if created.LoteoID != loteoID || created.LoteID != lotID || created.LoteNumero != "1" || created.ManzanaNumero != "1" {
		t.Errorf("created sale lote = %#v", created)
	}
	if created.Cliente.ID != clientID || created.Vendedor.ID != actorID || created.UsuarioAlta.ID != actorID {
		t.Errorf("created sale parties = %#v", created)
	}
	if created.Inmobiliaria != nil {
		t.Errorf("created sale agency = %#v, want none for an internal seller", created.Inmobiliaria)
	}
	if created.ModalidadPago != domain.PaymentMethodCash || !created.FechaCreacion.Equal(now) {
		t.Errorf("created sale payment = %q at %s", created.ModalidadPago, created.FechaCreacion)
	}
	// A contado sale is paid in full on registration: activa, then completada.
	if len(created.Historial) != 2 || created.Historial[0].Estado != domain.SaleStateActive || created.Historial[1].Estado != domain.SaleStateCompleted || created.Historial[1].Razon != "Pago al contado" {
		t.Errorf("history = %#v", created.Historial)
	}
	if created.Historial[1].Usuario == nil || created.Historial[1].Usuario.ID != actorID {
		t.Errorf("completada entry actor = %#v, want %s", created.Historial[1].Usuario, actorID)
	}

	var lotState string
	if err := pool.QueryRow(context.Background(), `SELECT estado_actual FROM lotes WHERE id = $1::uuid`, lotID).Scan(&lotState); err != nil {
		t.Fatalf("read lot state: %v", err)
	}
	if lotState != string(domain.LotStateCompleted) {
		t.Fatalf("lot state after a contado sale = %q, want finalizado", lotState)
	}

	var lotEvents int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM lote_estados WHERE lote_id = $1::uuid AND venta_id = $2::uuid AND origen = $3`, lotID, created.ID, string(domain.LotStateOriginSale)).Scan(&lotEvents); err != nil {
		t.Fatalf("read lot events: %v", err)
	}
	if lotEvents != 2 {
		t.Errorf("lot events for the sale = %d, want vendido and finalizado", lotEvents)
	}
	// A retry with the same key and payload is the lost response of the
	// first request, so it gets the same venta back.
	retried, err := repository.Create(context.Background(), command)
	if err != nil {
		t.Fatalf("retried Create() error = %v", err)
	}
	if retried.ID != created.ID || len(retried.Historial) != 2 {
		t.Errorf("retried Create() = %#v, want the original sale %q", retried, created.ID)
	}

	reused := command
	reused.IdempotencyPayloadHash = saleHash(t)
	if _, err := repository.Create(context.Background(), reused); !errors.Is(err, domain.ErrSaleIdempotencyConflict) {
		t.Fatalf("Create() reusing the key with other data error = %v, want %v", err, domain.ErrSaleIdempotencyConflict)
	}

	fresh := command
	fresh.IdempotencyKey = newUUID(t)
	if _, err := repository.Create(context.Background(), fresh); !errors.Is(err, domain.ErrSaleLotUnavailable) {
		t.Fatalf("second Create() error = %v, want %v", err, domain.ErrSaleLotUnavailable)
	}

	fetched, err := repository.Get(context.Background(), created.ID, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.ID != created.ID || len(fetched.Historial) != 2 {
		t.Errorf("Get() = %#v", fetched)
	}

	page, err := repository.List(context.Background(), domain.SaleListFilter{DevelopmentID: loteoID}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != created.ID || page.TotalPages != 1 {
		t.Errorf("List() = %#v", page)
	}

	byClient, err := repository.List(context.Background(), domain.SaleListFilter{DevelopmentID: loteoID, Search: "Cliente"}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("List() by client error = %v", err)
	}
	if byClient.Total != 1 {
		t.Errorf("List() by client = %#v, want the sale", byClient)
	}
	noMatch, err := repository.List(context.Background(), domain.SaleListFilter{DevelopmentID: loteoID, Search: "nadie"}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("List() no match error = %v", err)
	}
	if noMatch.Total != 0 || len(noMatch.Items) != 0 {
		t.Errorf("List() no match = %#v", noMatch)
	}
	cancelled, err := repository.List(context.Background(), domain.SaleListFilter{DevelopmentID: loteoID, States: []domain.SaleState{domain.SaleStateCancelled}}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("List() cancelled error = %v", err)
	}
	if cancelled.Total != 0 {
		t.Errorf("List() cancelled = %#v, want none", cancelled)
	}
	outsidePage, err := repository.List(context.Background(), domain.SaleListFilter{DevelopmentID: loteoID, Page: 99}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("List() out-of-range error = %v", err)
	}
	if len(outsidePage.Items) != 0 || outsidePage.Total != 1 || outsidePage.TotalPages != 1 {
		t.Errorf("List() out-of-range = %#v, want totals preserved", outsidePage)
	}

	strangerAuthID := newUUID(t)
	agencyScope := gateway.SaleScope{AssigneeAuthProviderID: &strangerAuthID, ByAgency: true}
	if _, err := repository.Get(context.Background(), created.ID, agencyScope); !errors.Is(err, domain.ErrSaleNotFound) {
		t.Fatalf("Get() outside scope error = %v, want %v", err, domain.ErrSaleNotFound)
	}
	scoped, err := repository.List(context.Background(), domain.SaleListFilter{DevelopmentID: loteoID}, agencyScope)
	if err != nil {
		t.Fatalf("List() outside scope error = %v", err)
	}
	if scoped.Total != 0 {
		t.Errorf("List() outside scope = %#v, want none", scoped)
	}
}

func TestSaleRepositoryAgencySellerAndScope(t *testing.T) {
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
	sellerID, sellerAuthID, peerID, agencyID, agencyName := agencySellerFixture(t, pool, loteoID)
	otherSellerID, _ := unassignedAgencySellerFixture(t, pool)
	// The sale references the agency sellers, so the loteo (and its ventas)
	// must go before the seller fixtures clean themselves up.
	t.Cleanup(func() { deleteLoteo(t, pool, loteoID) })
	repository := postgres.NewSaleRepository(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)

	// An agency actor sells only on a loteo their agency is assigned to; the
	// administrator is free to pick an unassigned agency's seller.
	_, err = repository.Create(context.Background(), gateway.CreateSaleCommand{
		DevelopmentID: loteoID, LotID: lotID, ClientID: clientID, SellerID: otherSellerID,
		ActorID: otherSellerID, PaymentMethod: domain.PaymentMethodCash, CreatedAt: now,
		IdempotencyKey: newUUID(t), IdempotencyPayloadHash: saleHash(t),
	})
	if !errors.Is(err, domain.ErrSaleAgencyNotAssigned) {
		t.Fatalf("Create() by an unassigned agency error = %v, want %v", err, domain.ErrSaleAgencyNotAssigned)
	}

	// An agency actor can register the sale for a colleague, but not for a
	// seller of another agency.
	_, err = repository.Create(context.Background(), gateway.CreateSaleCommand{
		DevelopmentID: loteoID, LotID: lotID, ClientID: clientID, SellerID: otherSellerID,
		ActorID: sellerID, PaymentMethod: domain.PaymentMethodCash, CreatedAt: now,
		IdempotencyKey: newUUID(t), IdempotencyPayloadHash: saleHash(t),
	})
	if !errors.Is(err, domain.ErrSaleSellerNotEligible) {
		t.Fatalf("Create() for another agency error = %v, want %v", err, domain.ErrSaleSellerNotEligible)
	}

	created, err := repository.Create(context.Background(), gateway.CreateSaleCommand{
		DevelopmentID: loteoID, LotID: lotID, ClientID: clientID, SellerID: peerID,
		ActorID: sellerID, PaymentMethod: domain.PaymentMethodCash, CreatedAt: now,
		IdempotencyKey: newUUID(t), IdempotencyPayloadHash: saleHash(t),
	})
	if err != nil {
		t.Fatalf("Create() for a colleague error = %v", err)
	}
	if created.Vendedor.ID != peerID || created.UsuarioAlta.ID != sellerID {
		t.Errorf("created sale parties = %#v", created)
	}
	if created.Inmobiliaria == nil || created.Inmobiliaria.ID != agencyID || created.Inmobiliaria.BusinessName != agencyName {
		t.Errorf("created sale agency = %#v, want %q", created.Inmobiliaria, agencyID)
	}

	ownScope := gateway.SaleScope{AssigneeAuthProviderID: &sellerAuthID, ByAgency: true}
	fetched, err := repository.Get(context.Background(), created.ID, ownScope)
	if err != nil {
		t.Fatalf("Get() within agency scope error = %v", err)
	}
	if fetched.ID != created.ID {
		t.Errorf("Get() within agency scope = %#v", fetched)
	}
	page, err := repository.List(context.Background(), domain.SaleListFilter{DevelopmentID: loteoID}, ownScope)
	if err != nil {
		t.Fatalf("List() within agency scope error = %v", err)
	}
	if page.Total != 1 || page.Items[0].ID != created.ID {
		t.Errorf("List() within agency scope = %#v", page)
	}
	if _, err := repository.Get(context.Background(), created.ID, gateway.SaleScope{}); err != nil {
		t.Fatalf("Get() as administrator error = %v", err)
	}

	_, otherLotID := secondLotFixture(t, pool, loteoID)
	byAdmin, err := repository.Create(context.Background(), gateway.CreateSaleCommand{
		DevelopmentID: loteoID, LotID: otherLotID, ClientID: clientID, SellerID: otherSellerID,
		ActorID: actorID, PaymentMethod: domain.PaymentMethodCash, CreatedAt: now,
		IdempotencyKey: newUUID(t), IdempotencyPayloadHash: saleHash(t),
	})
	if err != nil {
		t.Fatalf("Create() by the administrator with an unassigned agency seller error = %v", err)
	}
	if byAdmin.Vendedor.ID != otherSellerID {
		t.Errorf("administrator sale seller = %#v, want %q", byAdmin.Vendedor, otherSellerID)
	}
}

func secondLotFixture(t *testing.T, pool *pgxpool.Pool, loteoID string) (manzanaID, lotID string) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), `
		SELECT id::text FROM manzanas WHERE loteo_id = $1::uuid LIMIT 1
	`, loteoID).Scan(&manzanaID); err != nil {
		t.Fatalf("find fixture manzana: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO lotes (manzana_id, loteo_id, numero, precio, moneda) VALUES ($1::uuid, $2::uuid, '2', 90000, 'USD') RETURNING id::text
	`, manzanaID, loteoID).Scan(&lotID); err != nil {
		t.Fatalf("create second lot: %v", err)
	}
	return manzanaID, lotID
}

func saleHash(t *testing.T) string {
	t.Helper()
	sum := sha256.Sum256([]byte(newUUID(t)))
	return hex.EncodeToString(sum[:])
}

func TestSaleRepositoryRejectsInvalidReferences(t *testing.T) {
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
	repository := postgres.NewSaleRepository(pool)
	now := time.Now().UTC()
	valid := gateway.CreateSaleCommand{
		DevelopmentID: loteoID, LotID: lotID, ClientID: clientID, SellerID: actorID,
		ActorID: actorID, PaymentMethod: domain.PaymentMethodCash, CreatedAt: now,
		IdempotencyKey: newUUID(t), IdempotencyPayloadHash: saleHash(t),
	}

	for name, test := range map[string]struct {
		mutate func(command *gateway.CreateSaleCommand)
		want   error
	}{
		"unknown loteo":  {func(c *gateway.CreateSaleCommand) { c.DevelopmentID = newUUID(t) }, domain.ErrLoteNotFound},
		"malformed lote": {func(c *gateway.CreateSaleCommand) { c.LotID = "nope" }, domain.ErrLoteNotFound},
		"unknown actor":  {func(c *gateway.CreateSaleCommand) { c.ActorID = newUUID(t) }, domain.ErrActorNoAprovisionado},
		"unknown seller": {func(c *gateway.CreateSaleCommand) { c.SellerID = newUUID(t) }, domain.ErrSaleSellerNotEligible},
		"unknown client": {func(c *gateway.CreateSaleCommand) { c.ClientID = newUUID(t) }, domain.ErrSaleInvalidClient},
	} {
		t.Run(name, func(t *testing.T) {
			command := valid
			test.mutate(&command)
			if _, err := repository.Create(context.Background(), command); !errors.Is(err, test.want) {
				t.Fatalf("Create() error = %v, want %v", err, test.want)
			}
		})
	}

	if _, err := pool.Exec(context.Background(), `UPDATE lotes SET precio = NULL WHERE id = $1::uuid`, lotID); err != nil {
		t.Fatalf("clear lot price: %v", err)
	}
	if _, err := repository.Create(context.Background(), valid); !errors.Is(err, domain.ErrSaleLotIncomplete) {
		t.Fatalf("Create() without price error = %v, want %v", err, domain.ErrSaleLotIncomplete)
	}

	if _, err := repository.Get(context.Background(), "nope", gateway.SaleScope{}); !errors.Is(err, domain.ErrSaleNotFound) {
		t.Fatalf("Get() malformed id error = %v, want %v", err, domain.ErrSaleNotFound)
	}
	if _, err := repository.Get(context.Background(), newUUID(t), gateway.SaleScope{}); !errors.Is(err, domain.ErrSaleNotFound) {
		t.Fatalf("Get() unknown id error = %v, want %v", err, domain.ErrSaleNotFound)
	}
	if _, err := repository.List(context.Background(), domain.SaleListFilter{Page: -1}, gateway.SaleScope{}); !errors.Is(err, domain.ErrSaleInvalidPage) {
		t.Fatalf("List() invalid page error = %v, want %v", err, domain.ErrSaleInvalidPage)
	}
}
