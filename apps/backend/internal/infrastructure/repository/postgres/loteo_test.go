package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

func squareAt(offset float64) domain.Polygon {
	return domain.Polygon{
		{X: offset, Y: offset},
		{X: offset + 10, Y: offset},
		{X: offset + 10, Y: offset + 10},
		{X: offset, Y: offset + 10},
	}
}

func testPlan() *domain.DxfPlan {
	return &domain.DxfPlan{
		Loteo: domain.DxfEntity{Handle: "1A", Polygon: squareAt(0)},
		Manzanas: []domain.DxfEntity{
			{Handle: "2A", Polygon: squareAt(100)},
			{Handle: "2B", Polygon: squareAt(200)},
		},
		Lotes: []domain.DxfLote{
			{DxfEntity: domain.DxfEntity{Handle: "3A", Polygon: squareAt(300)}, ManzanaIndex: 1},
			{DxfEntity: domain.DxfEntity{Handle: "3B", Polygon: squareAt(400)}, ManzanaIndex: 0},
		},
		Calles: []domain.DxfEntity{
			{Handle: "4A", Polygon: squareAt(500)},
		},
	}
}

// TestLoteoRepositoryWithoutAReachableDatabase checks that a connectivity
// failure surfaces as an error on every operation instead of being swallowed
// into an empty result. It needs no database: the pool points at a port
// nothing listens on.
func TestLoteoRepositoryWithoutAReachableDatabase(t *testing.T) {
	t.Parallel()

	pool, err := pgxpool.New(context.Background(), "postgres://loteos:loteos@127.0.0.1:1/loteos")
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	repository := postgres.NewLoteoRepository(pool)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	actor := newUUID(t)

	t.Run("create", func(t *testing.T) {
		if _, err := repository.Create(ctx, actor, domain.NewLoteo{Name: "Loteo Norte", Plan: testPlan()}); err == nil {
			t.Error("Create() should fail when the database is unreachable")
		}
	})

	t.Run("update lote", func(t *testing.T) {
		if _, err := repository.UpdateLote(ctx, actor, newUUID(t), newUUID(t), domain.LoteData{Number: "12"}); err == nil {
			t.Error("UpdateLote() should fail when the database is unreachable")
		}
	})

	t.Run("assignment lookup", func(t *testing.T) {
		// This one must not answer "not assigned" on a failure: that would
		// turn an outage into a silent permission decision.
		assigned, err := repository.IsAssignedToLoteo(ctx, actor, newUUID(t))
		if err == nil {
			t.Error("IsAssignedToLoteo() should fail when the database is unreachable")
		}
		if assigned {
			t.Error("IsAssignedToLoteo() should not report an assignment it could not read")
		}
	})
}

// TestLoteoRepository is an integration test: it needs a real PostgreSQL
// instance with PostGIS and the migrations applied (see docs/database.md) and
// is skipped when DATABASE_URL is not set.
func TestLoteoRepository(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	repository := postgres.NewLoteoRepository(pool)
	actor := createUsuario(t, pool)

	t.Run("create without a plan", func(t *testing.T) {
		loteo, err := repository.Create(context.Background(), actor, domain.NewLoteo{
			Name: "Loteo Sin Plano", Location: "Córdoba",
		})
		t.Cleanup(func() { deleteLoteo(t, pool, loteo.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if loteo.ID == "" {
			t.Error("Create() should assign an id")
		}
		if loteo.CreatedAt.IsZero() {
			t.Error("Create() should set fecha_creacion")
		}
		if loteo.Name != "Loteo Sin Plano" || loteo.Location != "Córdoba" {
			t.Errorf("Create() = %#v", loteo)
		}
		if len(loteo.Manzanas) != 0 || len(loteo.Lotes) != 0 || len(loteo.Calles) != 0 {
			t.Error("Create() should not invent geometry for a loteo without a plan")
		}
	})

	t.Run("create leaves an empty description null", func(t *testing.T) {
		loteo, err := repository.Create(context.Background(), actor, domain.NewLoteo{Name: "Loteo Mínimo"})
		t.Cleanup(func() { deleteLoteo(t, pool, loteo.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if loteo.Location != "" || loteo.Description != "" {
			t.Errorf("Create() = %#v, want empty text for the fields that weren't sent", loteo)
		}
	})

	t.Run("create with a plan", func(t *testing.T) {
		plan := testPlan()

		loteo, err := repository.Create(context.Background(), actor, domain.NewLoteo{
			Name: "Loteo Con Plano", Plan: plan,
		})
		t.Cleanup(func() { deleteLoteo(t, pool, loteo.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if len(loteo.Manzanas) != 2 || len(loteo.Lotes) != 2 || len(loteo.Calles) != 1 {
			t.Fatalf("Create() = %d manzanas, %d lotes, %d calles", len(loteo.Manzanas), len(loteo.Lotes), len(loteo.Calles))
		}

		// The first lote belongs to the second manzana and the second lote to
		// the first, so a result that merely follows the input order would
		// pass this only by accident.
		if loteo.Lotes[0].ManzanaID != loteo.Manzanas[1].ID {
			t.Errorf("lote[0].ManzanaID = %q, want manzana[1] (%q)", loteo.Lotes[0].ManzanaID, loteo.Manzanas[1].ID)
		}
		if loteo.Lotes[1].ManzanaID != loteo.Manzanas[0].ID {
			t.Errorf("lote[1].ManzanaID = %q, want manzana[0] (%q)", loteo.Lotes[1].ManzanaID, loteo.Manzanas[0].ID)
		}

		assertDxfEntities(t, pool, loteo.ID)
	})

	t.Run("update a lote", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)
		price := 150000.0
		area := 300.5

		lote, err := repository.UpdateLote(context.Background(), actor, loteo.ID, loteo.Lotes[0].ID, domain.LoteData{
			Number: "12", Price: &price, Currency: "ARS",
			Area: &area, Features: "esquina",
		})
		if err != nil {
			t.Fatalf("UpdateLote() error = %v", err)
		}

		if lote.Number != "12" || lote.Currency != "ARS" || lote.Features != "esquina" {
			t.Errorf("UpdateLote() = %#v", lote)
		}
		if lote.Price == nil || *lote.Price != price {
			t.Errorf("price = %v, want %v", lote.Price, price)
		}
		if lote.Area == nil || *lote.Area != area {
			t.Errorf("area = %v, want %v", lote.Area, area)
		}
		if lote.ManzanaID != loteo.Manzanas[1].ID {
			t.Errorf("UpdateLote() should keep the lote's manzana, got %q", lote.ManzanaID)
		}
	})

	t.Run("update a lote clears the values that were not sent", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)
		price := 1000.0

		loteID := loteo.Lotes[0].ID
		if _, err := repository.UpdateLote(context.Background(), actor, loteo.ID, loteID, domain.LoteData{
			Number: "12", Price: &price, Currency: "ARS",
		}); err != nil {
			t.Fatalf("UpdateLote() error = %v", err)
		}

		lote, err := repository.UpdateLote(context.Background(), actor, loteo.ID, loteID, domain.LoteData{Number: "12"})
		if err != nil {
			t.Fatalf("UpdateLote() error = %v", err)
		}
		if lote.Price != nil || lote.Currency != "" {
			t.Errorf("UpdateLote() = %#v, want the price cleared", lote)
		}
	})

	t.Run("update rejects a lote of another loteo", func(t *testing.T) {
		propio := createLoteoWithPlan(t, pool, repository, actor)
		ajeno := createLoteoWithPlan(t, pool, repository, actor)

		// The lote exists, but not under the loteo the caller was authorized
		// on, so it has to read as missing.
		_, err := repository.UpdateLote(context.Background(), actor, propio.ID, ajeno.Lotes[0].ID, domain.LoteData{Number: "12"})

		if !errors.Is(err, domain.ErrLoteNotFound) {
			t.Fatalf("UpdateLote() error = %v, want %v", err, domain.ErrLoteNotFound)
		}
	})

	t.Run("update rejects ids that cannot name anything", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)

		for name, loteID := range map[string]string{
			"unknown uuid": newUUID(t),
			"not a uuid":   "'; DROP TABLE lotes; --",
			"empty":        "",
		} {
			t.Run(name, func(t *testing.T) {
				_, err := repository.UpdateLote(context.Background(), actor, loteo.ID, loteID, domain.LoteData{Number: "12"})
				if !errors.Is(err, domain.ErrLoteNotFound) {
					t.Fatalf("UpdateLote() error = %v, want %v", err, domain.ErrLoteNotFound)
				}
			})
		}
	})

	t.Run("update rejects a number already used in the loteo", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)

		if _, err := repository.UpdateLote(context.Background(), actor, loteo.ID, loteo.Lotes[0].ID, domain.LoteData{Number: "7"}); err != nil {
			t.Fatalf("UpdateLote() error = %v", err)
		}

		_, err := repository.UpdateLote(context.Background(), actor, loteo.ID, loteo.Lotes[1].ID, domain.LoteData{Number: "7"})
		if !errors.Is(err, domain.ErrLoteNumberInUse) {
			t.Fatalf("UpdateLote() error = %v, want %v", err, domain.ErrLoteNumberInUse)
		}
	})

	t.Run("the same number is free in another loteo", func(t *testing.T) {
		primero := createLoteoWithPlan(t, pool, repository, actor)
		segundo := createLoteoWithPlan(t, pool, repository, actor)

		if _, err := repository.UpdateLote(context.Background(), actor, primero.ID, primero.Lotes[0].ID, domain.LoteData{Number: "1"}); err != nil {
			t.Fatalf("UpdateLote() error = %v", err)
		}
		if _, err := repository.UpdateLote(context.Background(), actor, segundo.ID, segundo.Lotes[0].ID, domain.LoteData{Number: "1"}); err != nil {
			t.Fatalf("UpdateLote() error = %v, want the number to be free in another loteo", err)
		}
	})

	t.Run("assignment lookup", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)

		assigned, err := repository.IsAssignedToLoteo(context.Background(), actor, loteo.ID)
		if err != nil {
			t.Fatalf("IsAssignedToLoteo() error = %v", err)
		}
		if assigned {
			t.Error("IsAssignedToLoteo() should be false before the loteo is assigned")
		}

		assignLoteo(t, pool, actor, loteo.ID)

		assigned, err = repository.IsAssignedToLoteo(context.Background(), actor, loteo.ID)
		if err != nil {
			t.Fatalf("IsAssignedToLoteo() error = %v", err)
		}
		if !assigned {
			t.Error("IsAssignedToLoteo() should be true once the loteo is assigned")
		}
	})

	t.Run("assignment lookup treats an unusable id as not assigned", func(t *testing.T) {
		assigned, err := repository.IsAssignedToLoteo(context.Background(), actor, "not-a-uuid")
		if err != nil {
			t.Fatalf("IsAssignedToLoteo() error = %v, want a clean negative answer", err)
		}
		if assigned {
			t.Error("IsAssignedToLoteo() should be false for an id that can't name a loteo")
		}
	})
}

// assertDxfEntities checks what only a real database can show: that the rings
// reached a PostGIS polygon column intact, closed, and on the right layer.
func assertDxfEntities(t *testing.T, pool *pgxpool.Pool, loteoID string) {
	t.Helper()

	var layers map[string]int
	rows, err := pool.Query(context.Background(), `
		SELECT capa, count(*) FROM dxf_entidades WHERE loteo_id = $1::uuid GROUP BY capa
	`, loteoID)
	if err != nil {
		t.Fatalf("query dxf_entidades: %v", err)
	}
	defer rows.Close()

	layers = map[string]int{}
	for rows.Next() {
		var layer string
		var total int
		if err := rows.Scan(&layer, &total); err != nil {
			t.Fatalf("scan dxf_entidades: %v", err)
		}
		layers[layer] = total
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read dxf_entidades: %v", err)
	}

	want := map[string]int{"LOTEO": 1, "MANZANA": 2, "LOTES": 2, "CALLE": 1}
	for layer, total := range want {
		if layers[layer] != total {
			t.Errorf("capa %s = %d entidades, want %d", layer, layers[layer], total)
		}
	}

	var wkt string
	err = pool.QueryRow(context.Background(), `
		SELECT ST_AsText(geom) FROM dxf_entidades WHERE loteo_id = $1::uuid AND capa = 'LOTEO'
	`, loteoID).Scan(&wkt)
	if err != nil {
		t.Fatalf("read the loteo geometry: %v", err)
	}
	if wkt != "POLYGON((0 0,10 0,10 10,0 10,0 0))" {
		t.Errorf("stored geometry = %s", wkt)
	}

	var dxfEntityID *string
	if err := pool.QueryRow(context.Background(), `SELECT dxf_entidad_id::text FROM loteos WHERE id = $1::uuid`, loteoID).Scan(&dxfEntityID); err != nil {
		t.Fatalf("read loteos.dxf_entidad_id: %v", err)
	}
	if dxfEntityID == nil {
		t.Error("Create() should point the loteo at its LOTEO entity")
	}
}

func createLoteoWithPlan(t *testing.T, pool *pgxpool.Pool, repository *postgres.LoteoRepository, actor string) domain.Loteo {
	t.Helper()

	loteo, err := repository.Create(context.Background(), actor, domain.NewLoteo{
		Name: "Loteo " + newUUID(t), Plan: testPlan(),
	})
	t.Cleanup(func() { deleteLoteo(t, pool, loteo.ID) })
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	return loteo
}

func createUsuario(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	authProviderID := newUUID(t)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO usuarios (auth_provider_id, email, rol) VALUES ($1::uuid, $2, $3)
	`, authProviderID, newEmail(t), domain.RolAdministrador)
	if err != nil {
		t.Fatalf("create usuario: %v", err)
	}
	t.Cleanup(func() { deleteUsuario(t, pool, authProviderID) })

	return authProviderID
}

func assignLoteo(t *testing.T, pool *pgxpool.Pool, authProviderID, loteoID string) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `
		INSERT INTO usuario_loteos (usuario_id, loteo_id)
		SELECT id, $2::uuid FROM usuarios WHERE auth_provider_id = $1::uuid
	`, authProviderID, loteoID)
	if err != nil {
		t.Fatalf("assign loteo: %v", err)
	}
}

// deleteLoteo removes the rows in reverse dependency order: the plan's tables
// reference the loteo, and the loteo references its own DXF entity.
func deleteLoteo(t *testing.T, pool *pgxpool.Pool, loteoID string) {
	t.Helper()

	if loteoID == "" {
		return
	}

	statements := []string{
		`DELETE FROM usuario_loteos WHERE loteo_id = $1::uuid`,
		`DELETE FROM lotes WHERE loteo_id = $1::uuid`,
		`DELETE FROM manzana_calles WHERE loteo_id = $1::uuid`,
		`DELETE FROM calles WHERE loteo_id = $1::uuid`,
		`DELETE FROM manzanas WHERE loteo_id = $1::uuid`,
		`UPDATE loteos SET dxf_entidad_id = NULL WHERE id = $1::uuid`,
		`DELETE FROM dxf_entidades WHERE loteo_id = $1::uuid`,
		`DELETE FROM loteos WHERE id = $1::uuid`,
	}
	for _, statement := range statements {
		if _, err := pool.Exec(context.Background(), statement, loteoID); err != nil {
			t.Errorf("cleanup loteo: %v", err)
			return
		}
	}
}
