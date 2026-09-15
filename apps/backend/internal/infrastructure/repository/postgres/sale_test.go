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
		LoteoID:       loteoID,
		LoteID:        lotID,
		ClienteID:     clientID,
		VendedorID:    actorID,
		ActorID:       actorID,
		PaymentMethod: domain.PaymentMethodCash,
		CreatedAt:     now,
	}

	created, err := repository.Create(context.Background(), command)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Estado != domain.SaleStateActive || created.Monto != 100000 || created.Moneda != "USD" {
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
	if len(created.Historial) != 1 || created.Historial[0].Estado != domain.SaleStateActive {
		t.Errorf("initial history = %#v", created.Historial)
	}

	var lotState string
	if err := pool.QueryRow(context.Background(), `SELECT estado_actual FROM lotes WHERE id = $1::uuid`, lotID).Scan(&lotState); err != nil {
		t.Fatalf("read lot state: %v", err)
	}
	if lotState != string(domain.LotStateSold) {
		t.Fatalf("lot state after sale = %q, want vendido", lotState)
	}

	if _, err := repository.Create(context.Background(), command); !errors.Is(err, domain.ErrSaleLotUnavailable) {
		t.Fatalf("second Create() error = %v, want %v", err, domain.ErrSaleLotUnavailable)
	}

	fetched, err := repository.Get(context.Background(), created.ID, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if fetched.ID != created.ID || len(fetched.Historial) != 1 {
		t.Errorf("Get() = %#v", fetched)
	}

	page, err := repository.List(context.Background(), domain.SaleListFilter{LoteoID: loteoID}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != created.ID || page.TotalPages != 1 {
		t.Errorf("List() = %#v", page)
	}

	byClient, err := repository.List(context.Background(), domain.SaleListFilter{LoteoID: loteoID, Search: "Cliente"}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("List() by client error = %v", err)
	}
	if byClient.Total != 1 {
		t.Errorf("List() by client = %#v, want the sale", byClient)
	}
	noMatch, err := repository.List(context.Background(), domain.SaleListFilter{LoteoID: loteoID, Search: "nadie"}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("List() no match error = %v", err)
	}
	if noMatch.Total != 0 || len(noMatch.Items) != 0 {
		t.Errorf("List() no match = %#v", noMatch)
	}
	cancelled, err := repository.List(context.Background(), domain.SaleListFilter{LoteoID: loteoID, States: []domain.SaleState{domain.SaleStateCancelled}}, gateway.SaleScope{})
	if err != nil {
		t.Fatalf("List() cancelled error = %v", err)
	}
	if cancelled.Total != 0 {
		t.Errorf("List() cancelled = %#v, want none", cancelled)
	}
	outsidePage, err := repository.List(context.Background(), domain.SaleListFilter{LoteoID: loteoID, Page: 99}, gateway.SaleScope{})
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
	scoped, err := repository.List(context.Background(), domain.SaleListFilter{LoteoID: loteoID}, agencyScope)
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

	// An agency actor can register the sale for a colleague, but not for a
	// seller of another agency.
	_, err = repository.Create(context.Background(), gateway.CreateSaleCommand{
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: otherSellerID,
		ActorID: sellerID, PaymentMethod: domain.PaymentMethodCash, CreatedAt: now,
	})
	if !errors.Is(err, domain.ErrSaleSellerNotEligible) {
		t.Fatalf("Create() for another agency error = %v, want %v", err, domain.ErrSaleSellerNotEligible)
	}

	created, err := repository.Create(context.Background(), gateway.CreateSaleCommand{
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: peerID,
		ActorID: sellerID, PaymentMethod: domain.PaymentMethodCash, CreatedAt: now,
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
	page, err := repository.List(context.Background(), domain.SaleListFilter{LoteoID: loteoID}, ownScope)
	if err != nil {
		t.Fatalf("List() within agency scope error = %v", err)
	}
	if page.Total != 1 || page.Items[0].ID != created.ID {
		t.Errorf("List() within agency scope = %#v", page)
	}
	if _, err := repository.Get(context.Background(), created.ID, gateway.SaleScope{}); err != nil {
		t.Fatalf("Get() as administrator error = %v", err)
	}
	_ = actorID
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
		LoteoID: loteoID, LoteID: lotID, ClienteID: clientID, VendedorID: actorID,
		ActorID: actorID, PaymentMethod: domain.PaymentMethodCash, CreatedAt: now,
	}

	for name, test := range map[string]struct {
		mutate func(command *gateway.CreateSaleCommand)
		want   error
	}{
		"unknown loteo":  {func(c *gateway.CreateSaleCommand) { c.LoteoID = newUUID(t) }, domain.ErrLoteNotFound},
		"malformed lote": {func(c *gateway.CreateSaleCommand) { c.LoteID = "nope" }, domain.ErrLoteNotFound},
		"unknown actor":  {func(c *gateway.CreateSaleCommand) { c.ActorID = newUUID(t) }, domain.ErrActorNoAprovisionado},
		"unknown seller": {func(c *gateway.CreateSaleCommand) { c.VendedorID = newUUID(t) }, domain.ErrSaleSellerNotEligible},
		"unknown client": {func(c *gateway.CreateSaleCommand) { c.ClienteID = newUUID(t) }, domain.ErrSaleInvalidClient},
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
