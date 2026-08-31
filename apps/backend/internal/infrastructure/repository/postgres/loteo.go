package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
)

// A malformed UUID reaches PostgreSQL as a cast failure rather than a missing
// row, so an id that can't name anything is reported as "not found" instead
// of surfacing as an unexpected 500.
const invalidTextRepresentationCode = "22P02"

// Every insert below writes the geometry as WKT and lets PostgreSQL apply the
// implicit text -> geometry cast that PostGIS registers. No PostGIS function
// or type is named anywhere in this file, so the repository keeps working if
// the extension lives in another schema or the application role's search_path
// changes.
const insertManzanaEntitySQL = `
	WITH entidad AS (
		INSERT INTO dxf_entidades (loteo_id, handle_dxf, capa, geom, usuario_modificacion)
		VALUES ($1::uuid, NULLIF($2, ''), 'MANZANA', $3, $4::uuid)
		RETURNING id
	)
	INSERT INTO manzanas (loteo_id, dxf_entidad_id, usuario_modificacion)
	SELECT $1::uuid, entidad.id, $4::uuid FROM entidad
	RETURNING id::text
`

const insertLoteEntitySQL = `
	WITH entidad AS (
		INSERT INTO dxf_entidades (loteo_id, handle_dxf, capa, geom, usuario_modificacion)
		VALUES ($1::uuid, NULLIF($2, ''), 'LOTES', $3, $4::uuid)
		RETURNING id
	)
	INSERT INTO lotes (loteo_id, manzana_id, dxf_entidad_id, usuario_modificacion)
	SELECT $1::uuid, $5::uuid, entidad.id, $4::uuid FROM entidad
	RETURNING id::text
`

const insertCalleEntitySQL = `
	WITH entidad AS (
		INSERT INTO dxf_entidades (loteo_id, handle_dxf, capa, geom, usuario_modificacion)
		VALUES ($1::uuid, NULLIF($2, ''), 'CALLE', $3, $4::uuid)
		RETURNING id
	)
	INSERT INTO calles (loteo_id, dxf_entidad_id, usuario_modificacion)
	SELECT $1::uuid, entidad.id, $4::uuid FROM entidad
	RETURNING id::text
`

type LoteoRepository struct {
	pool *pgxpool.Pool
}

func NewLoteoRepository(pool *pgxpool.Pool) *LoteoRepository {
	return &LoteoRepository{pool: pool}
}

// Create writes the loteo and its whole plan in one transaction: a plan that
// fails halfway would otherwise leave a loteo with part of its manzanas and
// no way to tell which ones are missing.
func (repository *LoteoRepository) Create(
	ctx context.Context,
	actorAuthProviderID string,
	newLoteo domain.NewLoteo,
) (domain.Loteo, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Loteo{}, err
	}
	defer tx.Rollback(ctx)

	var loteo domain.Loteo
	var userID *string
	err = tx.QueryRow(ctx, `
		INSERT INTO loteos (nombre, ubicacion, descripcion, usuario_modificacion)
		VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), (SELECT id FROM usuarios WHERE auth_provider_id = $4::uuid))
		RETURNING id::text, nombre, COALESCE(ubicacion, ''), COALESCE(descripcion, ''),
		          fecha_creacion, usuario_modificacion::text
	`, newLoteo.Name, newLoteo.Location, newLoteo.Description, actorAuthProviderID).Scan(
		&loteo.ID, &loteo.Name, &loteo.Location, &loteo.Description,
		&loteo.CreatedAt, &userID,
	)
	if err != nil {
		return domain.Loteo{}, err
	}

	if newLoteo.Plan != nil {
		if err := insertPlan(ctx, tx, userID, &loteo, newLoteo.Plan); err != nil {
			return domain.Loteo{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Loteo{}, err
	}

	return loteo, nil
}

// insertPlano fills loteo.Manzanas and loteo.Calles in the same order as the
// plan, which is what the gateway contract promises the caller.
func insertPlan(ctx context.Context, tx pgx.Tx, userID *string, loteo *domain.Loteo, plan *domain.DxfPlan) error {
	if err := insertLoteoEntity(ctx, tx, userID, loteo.ID, plan.Loteo); err != nil {
		return err
	}

	manzanaBatch := &pgx.Batch{}
	for _, manzana := range plan.Manzanas {
		manzanaBatch.Queue(insertManzanaEntitySQL, loteo.ID, manzana.Handle, polygonWKT(manzana.Polygon), userID)
	}
	manzanaIDs, err := queueIDs(ctx, tx, manzanaBatch, len(plan.Manzanas))
	if err != nil {
		return err
	}

	// Every lote already points at a position of plan.Manzanas: the use case
	// resolved the client's reference before this ran. Checking it again costs
	// nothing and turns a caller that skipped that step into an error instead
	// of an out-of-range panic.
	loteBatch := &pgx.Batch{}
	for _, lote := range plan.Lotes {
		if lote.ManzanaIndex < 0 || lote.ManzanaIndex >= len(manzanaIDs) {
			return fmt.Errorf("lote points at manzana %d of %d", lote.ManzanaIndex, len(manzanaIDs))
		}
		loteBatch.Queue(insertLoteEntitySQL, loteo.ID, lote.Handle, polygonWKT(lote.Polygon), userID, manzanaIDs[lote.ManzanaIndex])
	}
	loteIDs, err := queueIDs(ctx, tx, loteBatch, len(plan.Lotes))
	if err != nil {
		return err
	}

	calleBatch := &pgx.Batch{}
	for _, calle := range plan.Calles {
		calleBatch.Queue(insertCalleEntitySQL, loteo.ID, calle.Handle, polygonWKT(calle.Polygon), userID)
	}
	calleIDs, err := queueIDs(ctx, tx, calleBatch, len(plan.Calles))
	if err != nil {
		return err
	}

	loteo.Manzanas = make([]domain.Manzana, len(manzanaIDs))
	for i, id := range manzanaIDs {
		loteo.Manzanas[i] = domain.Manzana{ID: id}
	}

	loteo.Lotes = make([]domain.Lote, len(loteIDs))
	for i, id := range loteIDs {
		loteo.Lotes[i] = domain.Lote{ID: id, ManzanaID: manzanaIDs[plan.Lotes[i].ManzanaIndex]}
	}

	loteo.Calles = make([]domain.Calle, len(calleIDs))
	for i, id := range calleIDs {
		loteo.Calles[i] = domain.Calle{ID: id}
	}

	return nil
}

// insertLoteoEntity stores the LOTEO ring and points the loteo at it. It
// takes two statements because dxf_entidades.loteo_id references loteos and
// loteos.dxf_entidad_id references back, so neither row can name the other
// at insert time.
func insertLoteoEntity(ctx context.Context, tx pgx.Tx, userID *string, loteoID string, entity domain.DxfEntity) error {
	var entidadID string
	err := tx.QueryRow(ctx, `
		INSERT INTO dxf_entidades (loteo_id, handle_dxf, capa, geom, usuario_modificacion)
		VALUES ($1::uuid, NULLIF($2, ''), 'LOTEO', $3, $4::uuid)
		RETURNING id::text
	`, loteoID, entity.Handle, polygonWKT(entity.Polygon), userID).Scan(&entidadID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE loteos SET dxf_entidad_id = $2::uuid, fecha_modificacion = now()
		WHERE id = $1::uuid
	`, loteoID, entidadID)

	return err
}

// queueIDs runs every queued statement in one round trip and collects the id
// each one returns. pgx needs the batch results closed before the transaction
// is used again, so the close error is checked even when a scan already
// failed.
func queueIDs(ctx context.Context, tx pgx.Tx, batch *pgx.Batch, count int) ([]string, error) {
	if count == 0 {
		return nil, nil
	}

	results := tx.SendBatch(ctx, batch)

	ids := make([]string, 0, count)
	var scanErr error
	for range count {
		var id string
		if err := results.QueryRow().Scan(&id); err != nil {
			scanErr = err
			break
		}
		ids = append(ids, id)
	}

	closeErr := results.Close()
	if scanErr != nil {
		return nil, scanErr
	}
	if closeErr != nil {
		return nil, closeErr
	}

	return ids, nil
}

func (repository *LoteoRepository) UpdateLote(
	ctx context.Context,
	actorAuthProviderID, loteoID, loteID string,
	data domain.LoteData,
) (domain.Lote, error) {
	var lote domain.Lote

	// loteo_id is part of the predicate so a caller authorized on one loteo
	// can't reach a lote of another by guessing its id.
	err := repository.pool.QueryRow(ctx, `
		UPDATE lotes
		SET numero = $4,
		    precio = $5,
		    moneda = NULLIF($6, ''),
		    superficie = $7,
		    caracteristicas = NULLIF($8, ''),
		    usuario_modificacion = (SELECT id FROM usuarios WHERE auth_provider_id = $1::uuid),
		    fecha_modificacion = now()
		WHERE id = $3::uuid AND loteo_id = $2::uuid AND fecha_baja IS NULL
		RETURNING id::text, manzana_id::text, COALESCE(numero, ''), precio::float8,
		          COALESCE(moneda, ''), superficie::float8, COALESCE(caracteristicas, '')
	`, actorAuthProviderID, loteoID, loteID,
		data.Number, data.Price, data.Currency, data.Area, data.Features).Scan(
		&lote.ID, &lote.ManzanaID, &lote.Number, &lote.Price,
		&lote.Currency, &lote.Area, &lote.Features,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Lote{}, domain.ErrLoteNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case uniqueViolationCode:
				return domain.Lote{}, domain.ErrLoteNumberInUse
			case invalidTextRepresentationCode:
				return domain.Lote{}, domain.ErrLoteNotFound
			}
		}
		return domain.Lote{}, err
	}

	return lote, nil
}

func (repository *LoteoRepository) IsAssignedToLoteo(ctx context.Context, authProviderID, loteoID string) (bool, error) {
	var assigned bool

	err := repository.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM usuario_loteos
			JOIN usuarios ON usuarios.id = usuario_loteos.usuario_id
			WHERE usuarios.auth_provider_id = $1::uuid
			  AND usuario_loteos.loteo_id = $2::uuid
			  AND usuario_loteos.fecha_baja IS NULL
			  AND usuarios.fecha_baja IS NULL
		)
	`, authProviderID, loteoID).Scan(&assigned)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return false, nil
		}
		return false, err
	}

	return assigned, nil
}

func (repository *LoteoRepository) LoteoExists(ctx context.Context, loteoID string) (bool, error) {
	var exists bool

	err := repository.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM loteos WHERE id = $1::uuid AND fecha_baja IS NULL
		)
	`, loteoID).Scan(&exists)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return false, nil
		}
		return false, err
	}

	return exists, nil
}

// RecordDxfFile supersedes any DXF already recorded for the loteo and inserts
// the new one in a single transaction, so a reader never sees two active DXF
// rows for one loteo. The object at file.StorageKey is written by the caller
// before this runs.
func (repository *LoteoRepository) RecordDxfFile(
	ctx context.Context,
	actorAuthProviderID, loteoID string,
	file domain.NewLoteoDxfFile,
) (domain.LoteoDxfFile, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.LoteoDxfFile{}, err
	}
	defer tx.Rollback(ctx)

	var present int
	err = tx.QueryRow(ctx, `
		SELECT 1 FROM loteos WHERE id = $1::uuid AND fecha_baja IS NULL
		FOR UPDATE
	`, loteoID).Scan(&present)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.LoteoDxfFile{}, domain.ErrLoteoNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return domain.LoteoDxfFile{}, domain.ErrLoteoNotFound
		}
		return domain.LoteoDxfFile{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE archivos
		SET fecha_baja = now(), fecha_modificacion = now()
		WHERE loteo_id = $1::uuid AND categoria = 'dxf' AND fecha_baja IS NULL
	`, loteoID); err != nil {
		return domain.LoteoDxfFile{}, err
	}

	var recorded domain.LoteoDxfFile
	err = tx.QueryRow(ctx, `
		INSERT INTO archivos (
			loteo_id, nombre, nombre_original, categoria,
			storage_key, mime_type, hash_sha256, usuario_modificacion, fecha
		)
		VALUES (
			$1::uuid, 'original.dxf', NULLIF($2, ''), 'dxf',
			$3, NULLIF($4, ''), NULLIF($5, ''),
			(SELECT id FROM usuarios WHERE auth_provider_id = $6::uuid), now()
		)
		RETURNING id::text, storage_key, COALESCE(nombre_original, ''),
		          COALESCE(mime_type, ''), COALESCE(hash_sha256, ''), fecha_creacion
	`, loteoID, file.OriginalName, file.StorageKey, file.MimeType, file.Sha256, actorAuthProviderID).Scan(
		&recorded.ID, &recorded.StorageKey, &recorded.OriginalName,
		&recorded.MimeType, &recorded.Sha256, &recorded.CreatedAt,
	)
	if err != nil {
		return domain.LoteoDxfFile{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.LoteoDxfFile{}, err
	}

	return recorded, nil
}

// polygonWKT renders a ring as WKT. The ring arrives normalized, with each
// vertex listed once, and WKT closes a ring by repeating the first vertex.
func polygonWKT(polygon domain.Polygon) string {
	var builder strings.Builder
	builder.WriteString("POLYGON((")

	for _, point := range polygon {
		writePoint(&builder, point)
		builder.WriteByte(',')
	}
	writePoint(&builder, polygon[0])

	builder.WriteString("))")

	return builder.String()
}

func writePoint(builder *strings.Builder, point domain.Point) {
	// -1 precision emits the shortest decimal that parses back to the same
	// float64, so no coordinate is silently rounded on the way to the column.
	builder.WriteString(strconv.FormatFloat(point.X, 'f', -1, 64))
	builder.WriteByte(' ')
	builder.WriteString(strconv.FormatFloat(point.Y, 'f', -1, 64))
}
