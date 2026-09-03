package postgres_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

// unrestrictedScope is the scope an administrador/administrativo read uses:
// no assignee, so every loteo is visible.
var unrestrictedScope = gateway.LoteoScope{}

func userScope(authProviderID string) gateway.LoteoScope {
	return gateway.LoteoScope{AssigneeAuthProviderID: &authProviderID, ByUserAssignment: true}
}

func agencyScope(authProviderID string) gateway.LoteoScope {
	return gateway.LoteoScope{AssigneeAuthProviderID: &authProviderID, ByAgencyAssignment: true}
}

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

	t.Run("update manzana", func(t *testing.T) {
		if _, err := repository.UpdateManzana(ctx, actor, newUUID(t), newUUID(t), domain.ManzanaData{Number: "1"}); err == nil {
			t.Error("UpdateManzana() should fail when the database is unreachable")
		}
	})

	t.Run("update calle", func(t *testing.T) {
		if _, err := repository.UpdateCalle(ctx, actor, newUUID(t), newUUID(t), domain.CalleData{Name: "A"}); err == nil {
			t.Error("UpdateCalle() should fail when the database is unreachable")
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

	t.Run("loteo exists", func(t *testing.T) {
		// Same reasoning: an outage must not read as "the loteo is gone".
		exists, err := repository.LoteoExists(ctx, newUUID(t))
		if err == nil {
			t.Error("LoteoExists() should fail when the database is unreachable")
		}
		if exists {
			t.Error("LoteoExists() should not report existence it could not read")
		}
	})

	t.Run("record dxf file", func(t *testing.T) {
		if _, err := repository.RecordDxfFile(ctx, actor, newUUID(t), domain.NewLoteoDxfFile{
			StorageKey: "loteos/x/original.dxf", OriginalName: "plano.dxf",
		}); err == nil {
			t.Error("RecordDxfFile() should fail when the database is unreachable")
		}
	})

	t.Run("list", func(t *testing.T) {
		if _, err := repository.List(ctx, "", unrestrictedScope); err == nil {
			t.Error("List() should fail when the database is unreachable")
		}
	})

	t.Run("get", func(t *testing.T) {
		if _, err := repository.Get(ctx, newUUID(t), unrestrictedScope); err == nil {
			t.Error("Get() should fail when the database is unreachable")
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

	t.Run("create rejects an invalid actor id without storing the loteo", func(t *testing.T) {
		name := "Loteo Invalid Actor " + newUUID(t)

		_, err := repository.Create(context.Background(), "not-a-uuid", domain.NewLoteo{Name: name})
		if err == nil {
			t.Fatal("Create() error = nil, want an invalid actor id error")
		}

		assertLoteoNotStored(t, pool, name)
	})

	t.Run("create rolls back an invalid plan", func(t *testing.T) {
		name := "Loteo Invalid Plan " + newUUID(t)
		plan := testPlan()
		plan.Lotes[0].ManzanaIndex = len(plan.Manzanas)

		_, err := repository.Create(context.Background(), actor, domain.NewLoteo{Name: name, Plan: plan})
		if err == nil {
			t.Fatal("Create() error = nil, want an invalid manzana reference error")
		}

		assertLoteoNotStored(t, pool, name)
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

	t.Run("update a manzana", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)
		calleID := loteo.Calles[0].ID

		manzana, err := repository.UpdateManzana(context.Background(), actor, loteo.ID, loteo.Manzanas[0].ID, domain.ManzanaData{
			Number: "A", HasWater: true, HasPower: true, CalleIDs: []string{calleID},
		})
		if err != nil {
			t.Fatalf("UpdateManzana() error = %v", err)
		}
		if manzana.Number != "A" || !manzana.HasWater || manzana.HasSewer || !manzana.HasPower || manzana.HasGas {
			t.Errorf("UpdateManzana() = %#v", manzana)
		}
		if len(manzana.CalleIDs) != 1 || manzana.CalleIDs[0] != calleID {
			t.Errorf("CalleIDs = %#v, want [%q]", manzana.CalleIDs, calleID)
		}
		if len(manzana.Polygon) == 0 {
			t.Error("UpdateManzana() should keep the polygon")
		}

		loaded, err := repository.Get(context.Background(), loteo.ID, unrestrictedScope)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if loaded.Manzanas[0].Number != "A" || len(loaded.Manzanas[0].CalleIDs) != 1 {
			t.Errorf("Get() manzana = %#v", loaded.Manzanas[0])
		}
	})

	t.Run("update replaces the streets of a manzana", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)
		manzanaID := loteo.Manzanas[0].ID
		calleID := loteo.Calles[0].ID

		if _, err := repository.UpdateManzana(context.Background(), actor, loteo.ID, manzanaID, domain.ManzanaData{
			Number: "1", CalleIDs: []string{calleID},
		}); err != nil {
			t.Fatalf("UpdateManzana() error = %v", err)
		}

		manzana, err := repository.UpdateManzana(context.Background(), actor, loteo.ID, manzanaID, domain.ManzanaData{Number: "1"})
		if err != nil {
			t.Fatalf("UpdateManzana() error = %v", err)
		}
		if len(manzana.CalleIDs) != 0 {
			t.Errorf("CalleIDs = %#v, want none", manzana.CalleIDs)
		}
	})

	t.Run("update rejects a manzana of another loteo", func(t *testing.T) {
		propio := createLoteoWithPlan(t, pool, repository, actor)
		ajeno := createLoteoWithPlan(t, pool, repository, actor)

		_, err := repository.UpdateManzana(context.Background(), actor, propio.ID, ajeno.Manzanas[0].ID, domain.ManzanaData{Number: "1"})
		if !errors.Is(err, domain.ErrManzanaNotFound) {
			t.Fatalf("UpdateManzana() error = %v, want %v", err, domain.ErrManzanaNotFound)
		}
	})

	t.Run("update rejects a manzana number already used in the loteo", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)

		if _, err := repository.UpdateManzana(context.Background(), actor, loteo.ID, loteo.Manzanas[0].ID, domain.ManzanaData{Number: "1"}); err != nil {
			t.Fatalf("UpdateManzana() error = %v", err)
		}
		_, err := repository.UpdateManzana(context.Background(), actor, loteo.ID, loteo.Manzanas[1].ID, domain.ManzanaData{Number: "1"})
		if !errors.Is(err, domain.ErrManzanaNumberInUse) {
			t.Fatalf("UpdateManzana() error = %v, want %v", err, domain.ErrManzanaNumberInUse)
		}
	})

	t.Run("update rejects a calle that is not in the loteo", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)
		ajeno := createLoteoWithPlan(t, pool, repository, actor)

		_, err := repository.UpdateManzana(context.Background(), actor, loteo.ID, loteo.Manzanas[0].ID, domain.ManzanaData{
			Number: "1", CalleIDs: []string{ajeno.Calles[0].ID},
		})
		if !errors.Is(err, domain.ErrUnknownCalle) {
			t.Fatalf("UpdateManzana() error = %v, want %v", err, domain.ErrUnknownCalle)
		}
	})

	t.Run("update rejects a soft-deleted calle", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)
		calleID := loteo.Calles[0].ID
		if _, err := pool.Exec(context.Background(), `
			UPDATE calles
			SET fecha_baja = now(), usuario_modificacion = (SELECT id FROM usuarios WHERE auth_provider_id = $1::uuid)
			WHERE id = $2::uuid
		`, actor, calleID); err != nil {
			t.Fatalf("soft-delete calle: %v", err)
		}

		_, err := repository.UpdateManzana(context.Background(), actor, loteo.ID, loteo.Manzanas[0].ID, domain.ManzanaData{
			Number: "1", CalleIDs: []string{calleID},
		})
		if !errors.Is(err, domain.ErrUnknownCalle) {
			t.Fatalf("UpdateManzana() error = %v, want %v", err, domain.ErrUnknownCalle)
		}
	})

	t.Run("update a calle", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)

		calle, err := repository.UpdateCalle(context.Background(), actor, loteo.ID, loteo.Calles[0].ID, domain.CalleData{
			Name: "Los Álamos", Type: domain.CalleTypeAsfalto,
		})
		if err != nil {
			t.Fatalf("UpdateCalle() error = %v", err)
		}
		if calle.Name != "Los Álamos" || calle.Type != domain.CalleTypeAsfalto {
			t.Errorf("UpdateCalle() = %#v", calle)
		}
		if len(calle.Polygon) == 0 {
			t.Error("UpdateCalle() should keep the polygon")
		}

		cleared, err := repository.UpdateCalle(context.Background(), actor, loteo.ID, loteo.Calles[0].ID, domain.CalleData{
			Name: "Los Álamos",
		})
		if err != nil {
			t.Fatalf("UpdateCalle() error = %v", err)
		}
		if cleared.Type != "" {
			t.Errorf("Type = %q, want empty after clearing", cleared.Type)
		}
	})

	t.Run("update rejects a calle of another loteo", func(t *testing.T) {
		propio := createLoteoWithPlan(t, pool, repository, actor)
		ajeno := createLoteoWithPlan(t, pool, repository, actor)

		_, err := repository.UpdateCalle(context.Background(), actor, propio.ID, ajeno.Calles[0].ID, domain.CalleData{Name: "A"})
		if !errors.Is(err, domain.ErrCalleNotFound) {
			t.Fatalf("UpdateCalle() error = %v, want %v", err, domain.ErrCalleNotFound)
		}
	})

	t.Run("update rejects an invalid calle type", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)

		_, err := repository.UpdateCalle(context.Background(), actor, loteo.ID, loteo.Calles[0].ID, domain.CalleData{
			Name: "A", Type: "cemento",
		})
		if !errors.Is(err, domain.ErrInvalidCalleType) {
			t.Fatalf("UpdateCalle() error = %v, want %v", err, domain.ErrInvalidCalleType)
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

	t.Run("loteo exists", func(t *testing.T) {
		loteo, err := repository.Create(context.Background(), actor, domain.NewLoteo{Name: "Loteo " + newUUID(t)})
		t.Cleanup(func() { deleteLoteo(t, pool, loteo.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		for name, tc := range map[string]struct {
			id   string
			want bool
		}{
			"a real loteo": {loteo.ID, true},
			"unknown uuid": {newUUID(t), false},
			"not a uuid":   {"'; DROP TABLE loteos; --", false},
			"empty":        {"", false},
		} {
			t.Run(name, func(t *testing.T) {
				got, err := repository.LoteoExists(context.Background(), tc.id)
				if err != nil {
					t.Fatalf("LoteoExists() error = %v", err)
				}
				if got != tc.want {
					t.Fatalf("LoteoExists(%q) = %v, want %v", tc.id, got, tc.want)
				}
			})
		}
	})

	t.Run("records the dxf file", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)
		key := "loteos/" + loteo.ID + "/original.dxf"

		file, err := repository.RecordDxfFile(context.Background(), actor, loteo.ID, domain.NewLoteoDxfFile{
			StorageKey: key, OriginalName: "plano.dxf", MimeType: "application/dxf", Sha256: "abc123",
		})
		if err != nil {
			t.Fatalf("RecordDxfFile() error = %v", err)
		}
		if file.ID == "" || file.CreatedAt.IsZero() {
			t.Errorf("RecordDxfFile() = %#v, want an id and a fecha_creacion", file)
		}
		if file.StorageKey != key || file.OriginalName != "plano.dxf" || file.Sha256 != "abc123" {
			t.Errorf("RecordDxfFile() = %#v", file)
		}

		var (
			total       int
			categoria   string
			storageKey  string
			original    string
			hash        string
			userPresent bool
		)
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*) FILTER (WHERE fecha_baja IS NULL),
			       max(categoria), max(storage_key), max(nombre_original), max(hash_sha256),
			       bool_and(usuario_modificacion IS NOT NULL)
			FROM archivos WHERE loteo_id = $1::uuid
		`, loteo.ID).Scan(&total, &categoria, &storageKey, &original, &hash, &userPresent); err != nil {
			t.Fatalf("read archivos: %v", err)
		}
		if total != 1 || categoria != "dxf" || storageKey != key || original != "plano.dxf" || hash != "abc123" || !userPresent {
			t.Errorf("archivos row: total=%d categoria=%q key=%q original=%q hash=%q user=%v",
				total, categoria, storageKey, original, hash, userPresent)
		}
	})

	t.Run("recording again supersedes the previous dxf file", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)
		key := "loteos/" + loteo.ID + "/original.dxf"

		for _, name := range []string{"primero.dxf", "segundo.dxf"} {
			if _, err := repository.RecordDxfFile(context.Background(), actor, loteo.ID, domain.NewLoteoDxfFile{
				StorageKey: key, OriginalName: name, MimeType: "application/dxf", Sha256: name,
			}); err != nil {
				t.Fatalf("RecordDxfFile(%q) error = %v", name, err)
			}
		}

		var active, all int
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*) FILTER (WHERE fecha_baja IS NULL), count(*)
			FROM archivos WHERE loteo_id = $1::uuid AND categoria = 'dxf'
		`, loteo.ID).Scan(&active, &all); err != nil {
			t.Fatalf("read archivos: %v", err)
		}
		if active != 1 || all != 2 {
			t.Fatalf("archivos: %d active, %d total, want 1 and 2", active, all)
		}
	})

	t.Run("concurrent recordings leave one active dxf file", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)
		files := []domain.NewLoteoDxfFile{
			{StorageKey: "loteos/" + loteo.ID + "/dxf/first.dxf", OriginalName: "first.dxf", MimeType: "application/dxf", Sha256: "first"},
			{StorageKey: "loteos/" + loteo.ID + "/dxf/second.dxf", OriginalName: "second.dxf", MimeType: "application/dxf", Sha256: "second"},
		}

		start := make(chan struct{})
		errorsByCall := make(chan error, len(files))
		var workers sync.WaitGroup
		for _, file := range files {
			workers.Add(1)
			go func(file domain.NewLoteoDxfFile) {
				defer workers.Done()
				<-start
				_, err := repository.RecordDxfFile(context.Background(), actor, loteo.ID, file)
				errorsByCall <- err
			}(file)
		}
		close(start)
		workers.Wait()
		close(errorsByCall)
		for err := range errorsByCall {
			if err != nil {
				t.Fatalf("RecordDxfFile() error = %v", err)
			}
		}

		var active, all int
		if err := pool.QueryRow(context.Background(), `
			SELECT count(*) FILTER (WHERE fecha_baja IS NULL), count(*)
			FROM archivos WHERE loteo_id = $1::uuid AND categoria = 'dxf'
		`, loteo.ID).Scan(&active, &all); err != nil {
			t.Fatalf("read archivos: %v", err)
		}
		if active != 1 || all != 2 {
			t.Fatalf("archivos: %d active, %d total, want 1 and 2", active, all)
		}
	})

	t.Run("recording for an unknown loteo returns not found", func(t *testing.T) {
		for name, id := range map[string]string{
			"unknown uuid": newUUID(t),
			"not a uuid":   "nope",
		} {
			t.Run(name, func(t *testing.T) {
				_, err := repository.RecordDxfFile(context.Background(), actor, id, domain.NewLoteoDxfFile{
					StorageKey: "loteos/x/original.dxf", OriginalName: "plano.dxf",
				})
				if !errors.Is(err, domain.ErrLoteoNotFound) {
					t.Fatalf("RecordDxfFile() error = %v, want %v", err, domain.ErrLoteoNotFound)
				}
			})
		}
	})

	t.Run("lists active loteos with their plan counts, ordered by name", func(t *testing.T) {
		prefix := "ZZ List " + newUUID(t) + " "
		conPlano := createNamedLoteo(t, pool, repository, actor, prefix+"B", testPlan())
		sinPlano := createNamedLoteo(t, pool, repository, actor, prefix+"A", nil)

		summaries, err := repository.List(context.Background(), prefix, unrestrictedScope)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(summaries) != 2 {
			t.Fatalf("List() = %d loteos, want 2", len(summaries))
		}

		// Ordered by name: "...A" (sin plano) before "...B" (con plano).
		if summaries[0].ID != sinPlano.ID || summaries[1].ID != conPlano.ID {
			t.Fatalf("List() order = %q, %q; want %q then %q",
				summaries[0].Name, summaries[1].Name, sinPlano.Name, conPlano.Name)
		}
		if summaries[0].HasPlan {
			t.Error("the loteo without a plan should report HasPlan = false")
		}
		got := summaries[1]
		if !got.HasPlan {
			t.Error("the loteo with a plan should report HasPlan = true")
		}
		if got.ManzanaCount != 2 || got.LoteCount != 2 || got.CalleCount != 1 {
			t.Errorf("counts = %d manzanas, %d lotes, %d calles; want 2, 2, 1",
				got.ManzanaCount, got.LoteCount, got.CalleCount)
		}
		if got.HasDxfFile {
			t.Error("HasDxfFile should be false until a DXF is recorded")
		}
	})

	t.Run("filters the listing by name or ubicacion", func(t *testing.T) {
		token := newUUID(t)
		match := createNamedLoteo(t, pool, repository, actor, "Loteo "+token, nil)
		createNamedLoteo(t, pool, repository, actor, "Loteo "+newUUID(t), nil)

		summaries, err := repository.List(context.Background(), token, unrestrictedScope)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(summaries) != 1 || summaries[0].ID != match.ID {
			t.Fatalf("List(%q) = %#v, want just %q", token, summaries, match.ID)
		}
	})

	t.Run("gets a loteo with its geometry", func(t *testing.T) {
		loteo := createLoteoWithPlan(t, pool, repository, actor)

		got, err := repository.Get(context.Background(), loteo.ID, unrestrictedScope)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(got.Boundary) != 4 {
			t.Errorf("boundary = %d vertices, want 4", len(got.Boundary))
		}
		if len(got.Manzanas) != 2 || len(got.Lotes) != 2 || len(got.Calles) != 1 {
			t.Fatalf("Get() = %d manzanas, %d lotes, %d calles", len(got.Manzanas), len(got.Lotes), len(got.Calles))
		}
		for _, manzana := range got.Manzanas {
			if len(manzana.Polygon) != 4 {
				t.Errorf("manzana %q polygon = %d vertices, want 4", manzana.ID, len(manzana.Polygon))
			}
		}
		for _, lote := range got.Lotes {
			if len(lote.Polygon) != 4 {
				t.Errorf("lote %q polygon = %d vertices, want 4", lote.ID, len(lote.Polygon))
			}
			if lote.ManzanaID == "" {
				t.Errorf("lote %q should name its manzana", lote.ID)
			}
		}
		if len(got.Calles[0].Polygon) != 4 {
			t.Errorf("calle polygon = %d vertices, want 4", len(got.Calles[0].Polygon))
		}
	})

	t.Run("gets a loteo registered without a plan", func(t *testing.T) {
		loteo := createNamedLoteo(t, pool, repository, actor, "Loteo "+newUUID(t), nil)

		got, err := repository.Get(context.Background(), loteo.ID, unrestrictedScope)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.Boundary != nil || len(got.Manzanas) != 0 || len(got.Lotes) != 0 || len(got.Calles) != 0 {
			t.Errorf("Get() = %#v, want no geometry", got)
		}
	})

	t.Run("get treats an unknown or unparseable id as not found", func(t *testing.T) {
		for name, id := range map[string]string{
			"unknown uuid": newUUID(t),
			"not a uuid":   "'; DROP TABLE loteos; --",
			"empty":        "",
		} {
			t.Run(name, func(t *testing.T) {
				_, err := repository.Get(context.Background(), id, unrestrictedScope)
				if !errors.Is(err, domain.ErrLoteoNotFound) {
					t.Fatalf("Get(%q) error = %v, want %v", id, err, domain.ErrLoteoNotFound)
				}
			})
		}
	})

	t.Run("scopes the listing and detail to a caller's direct assignments", func(t *testing.T) {
		viewer := createUsuario(t, pool)
		propio := createLoteoWithPlan(t, pool, repository, actor)
		ajeno := createLoteoWithPlan(t, pool, repository, actor)
		assignLoteo(t, pool, viewer, propio.ID)

		summaries, err := repository.List(context.Background(), "", userScope(viewer))
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if !containsLoteo(summaries, propio.ID) {
			t.Error("List() should include a loteo assigned to the caller")
		}
		if containsLoteo(summaries, ajeno.ID) {
			t.Error("List() should not include a loteo the caller isn't assigned to")
		}

		if _, err := repository.Get(context.Background(), propio.ID, userScope(viewer)); err != nil {
			t.Errorf("Get() on an assigned loteo error = %v", err)
		}
		if _, err := repository.Get(context.Background(), ajeno.ID, userScope(viewer)); !errors.Is(err, domain.ErrLoteoNotFound) {
			t.Errorf("Get() on an unassigned loteo error = %v, want %v", err, domain.ErrLoteoNotFound)
		}
	})

	t.Run("scopes visibility through the caller's inmobiliaria", func(t *testing.T) {
		viewer := createUsuario(t, pool)
		loteo := createLoteoWithPlan(t, pool, repository, actor)
		inmobiliaria := createInmobiliaria(t, pool)
		linkUsuarioToInmobiliaria(t, pool, viewer, inmobiliaria)
		assignInmobiliariaToLoteo(t, pool, inmobiliaria, loteo.ID)

		summaries, err := repository.List(context.Background(), "", agencyScope(viewer))
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if !containsLoteo(summaries, loteo.ID) {
			t.Error("List() should include a loteo assigned to the caller's inmobiliaria")
		}
		if _, err := repository.Get(context.Background(), loteo.ID, agencyScope(viewer)); err != nil {
			t.Errorf("Get() through the caller's inmobiliaria error = %v", err)
		}
	})

	t.Run("an assignment path the scope does not enable stays invisible", func(t *testing.T) {
		// A direct usuario_loteos assignment is not reachable with an
		// agency-only scope, and an inmobiliaria_loteos assignment is not
		// reachable with a user-only scope: an agrimensor tied to an agency
		// by mistake can't borrow its loteos, and vice versa.
		directViewer := createUsuario(t, pool)
		directLoteo := createLoteoWithPlan(t, pool, repository, actor)
		assignLoteo(t, pool, directViewer, directLoteo.ID)

		agencyViewer := createUsuario(t, pool)
		agencyLoteo := createLoteoWithPlan(t, pool, repository, actor)
		inmobiliaria := createInmobiliaria(t, pool)
		linkUsuarioToInmobiliaria(t, pool, agencyViewer, inmobiliaria)
		assignInmobiliariaToLoteo(t, pool, inmobiliaria, agencyLoteo.ID)

		directSeenByAgencyScope, err := repository.List(context.Background(), "", agencyScope(directViewer))
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if containsLoteo(directSeenByAgencyScope, directLoteo.ID) {
			t.Error("a direct assignment must not be visible through an agency-only scope")
		}
		if _, err := repository.Get(context.Background(), directLoteo.ID, agencyScope(directViewer)); !errors.Is(err, domain.ErrLoteoNotFound) {
			t.Errorf("Get() error = %v, want %v", err, domain.ErrLoteoNotFound)
		}

		agencySeenByUserScope, err := repository.List(context.Background(), "", userScope(agencyViewer))
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if containsLoteo(agencySeenByUserScope, agencyLoteo.ID) {
			t.Error("an agency assignment must not be visible through a user-only scope")
		}
		if _, err := repository.Get(context.Background(), agencyLoteo.ID, userScope(agencyViewer)); !errors.Is(err, domain.ErrLoteoNotFound) {
			t.Errorf("Get() error = %v, want %v", err, domain.ErrLoteoNotFound)
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

func assertLoteoNotStored(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()

	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM loteos WHERE nombre = $1`, name).Scan(&count); err != nil {
		t.Fatalf("count loteos named %q: %v", name, err)
	}
	if count != 0 {
		t.Errorf("stored %d loteos named %q after Create() failed", count, name)
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

func createNamedLoteo(t *testing.T, pool *pgxpool.Pool, repository *postgres.LoteoRepository, actor, name string, plan *domain.DxfPlan) domain.Loteo {
	t.Helper()

	loteo, err := repository.Create(context.Background(), actor, domain.NewLoteo{Name: name, Plan: plan})
	t.Cleanup(func() { deleteLoteo(t, pool, loteo.ID) })
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	return loteo
}

func containsLoteo(summaries []domain.LoteoSummary, id string) bool {
	for _, summary := range summaries {
		if summary.ID == id {
			return true
		}
	}

	return false
}

func createInmobiliaria(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO inmobiliarias (razon_social) VALUES ($1) RETURNING id::text
	`, "Inmobiliaria "+newUUID(t)).Scan(&id); err != nil {
		t.Fatalf("create inmobiliaria: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM inmobiliarias WHERE id = $1::uuid`, id); err != nil {
			t.Errorf("cleanup inmobiliaria: %v", err)
		}
	})

	return id
}

func linkUsuarioToInmobiliaria(t *testing.T, pool *pgxpool.Pool, authProviderID, inmobiliariaID string) {
	t.Helper()

	if _, err := pool.Exec(context.Background(), `
		UPDATE usuarios SET inmobiliaria_id = $2::uuid WHERE auth_provider_id = $1::uuid
	`, authProviderID, inmobiliariaID); err != nil {
		t.Fatalf("link usuario to inmobiliaria: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			UPDATE usuarios SET inmobiliaria_id = NULL WHERE auth_provider_id = $1::uuid
		`, authProviderID); err != nil {
			t.Errorf("cleanup usuario inmobiliaria link: %v", err)
		}
	})
}

func assignInmobiliariaToLoteo(t *testing.T, pool *pgxpool.Pool, inmobiliariaID, loteoID string) {
	t.Helper()

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO inmobiliaria_loteos (inmobiliaria_id, loteo_id) VALUES ($1::uuid, $2::uuid)
	`, inmobiliariaID, loteoID); err != nil {
		t.Fatalf("assign inmobiliaria to loteo: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `
			DELETE FROM inmobiliaria_loteos WHERE inmobiliaria_id = $1::uuid AND loteo_id = $2::uuid
		`, inmobiliariaID, loteoID); err != nil {
			t.Errorf("cleanup inmobiliaria loteo assignment: %v", err)
		}
	})
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
		`DELETE FROM archivos WHERE loteo_id = $1::uuid`,
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
