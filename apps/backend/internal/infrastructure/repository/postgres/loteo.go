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
	"loteosapp/backend/internal/business/gateway"
)

// A malformed UUID reaches PostgreSQL as a cast failure rather than a missing
// row, so an id that can't name anything is reported as "not found" instead
// of surfacing as an unexpected 500.
const invalidTextRepresentationCode = "22P02"
const foreignKeyViolationCode = "23503"
const checkViolationCode = "23514"

// Every insert below writes the geometry as WKT and lets PostgreSQL apply the
// implicit text -> geometry cast that PostGIS registers, so the write path
// names no PostGIS function or type. Reading a ring back needs ST_AsText,
// which resolves as long as the PostGIS schema is on the search_path — the
// same assumption the repository's integration tests already make.
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

// The two read queries below keep a loteo visible only when it's assigned to
// the caller. The assignee id parameter being NULL means no limit
// (administrador, administrativo). Otherwise each assignment path is gated by
// its own boolean flag — the direct usuario_loteos path and the
// inmobiliaria_loteos path via the caller's agency — so a caller only
// reaches a loteo through the path its role actually grants. The three scope
// parameters are the last ones in each query so the predicate reads the same
// in both.

const loteoScopedPredicate = `($%[1]d::uuid IS NULL OR (
		($%[2]d AND EXISTS (
			SELECT 1 FROM usuario_loteos ul
			JOIN usuarios u ON u.id = ul.usuario_id
			WHERE ul.loteo_id = l.id AND ul.fecha_baja IS NULL
			  AND u.fecha_baja IS NULL AND u.auth_provider_id = $%[1]d::uuid
		))
		OR ($%[3]d AND EXISTS (
			SELECT 1 FROM inmobiliaria_loteos il
			JOIN usuarios u ON u.inmobiliaria_id = il.inmobiliaria_id
			WHERE il.loteo_id = l.id AND il.fecha_baja IS NULL
			  AND u.fecha_baja IS NULL AND u.auth_provider_id = $%[1]d::uuid
		))
	))`

var listLoteosSQL = `
	SELECT
		l.id::text,
		l.nombre,
		COALESCE(l.ubicacion, ''),
		COALESCE(l.descripcion, ''),
		(SELECT count(*) FROM manzanas m WHERE m.loteo_id = l.id AND m.fecha_baja IS NULL),
		(SELECT count(*) FROM lotes lo WHERE lo.loteo_id = l.id AND lo.fecha_baja IS NULL),
		(SELECT count(*) FROM calles c WHERE c.loteo_id = l.id AND c.fecha_baja IS NULL),
		l.dxf_entidad_id IS NOT NULL,
		EXISTS (
			SELECT 1 FROM archivos a
			WHERE a.loteo_id = l.id AND a.categoria = 'dxf' AND a.fecha_baja IS NULL
		),
		l.fecha_creacion
	FROM loteos l
	WHERE l.fecha_baja IS NULL
		AND ($1 = '' OR l.nombre ILIKE $2 ESCAPE '\' OR COALESCE(l.ubicacion, '') ILIKE $2 ESCAPE '\')
		AND ` + fmt.Sprintf(loteoScopedPredicate, 3, 4, 5) + `
	ORDER BY l.nombre, l.fecha_creacion
`

var getLoteoSQL = `
	SELECT
		l.id::text, l.nombre, COALESCE(l.ubicacion, ''), COALESCE(l.descripcion, ''),
		l.fecha_creacion, ST_AsText(bd.geom)
	FROM loteos l
	LEFT JOIN dxf_entidades bd ON bd.id = l.dxf_entidad_id
	WHERE l.fecha_baja IS NULL
		AND l.id = $1::uuid
		AND ` + fmt.Sprintf(loteoScopedPredicate, 2, 3, 4) + `
`

// List returns the active loteos as summaries. search filters by nombre or
// ubicacion; scope keeps only the loteos the caller may see. An assignee id
// that can't be parsed as a UUID yields an empty result rather than an error.
func (repository *LoteoRepository) List(
	ctx context.Context,
	search string,
	scope gateway.LoteoScope,
) ([]domain.LoteoSummary, error) {
	rows, err := repository.pool.Query(ctx, listLoteosSQL,
		search, containsPattern(search),
		scope.AssigneeAuthProviderID, scope.ByUserAssignment, scope.ByAgencyAssignment,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return []domain.LoteoSummary{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	// Empty, not nil, so no matches serializes as "loteos": [].
	loteos := make([]domain.LoteoSummary, 0)
	for rows.Next() {
		var loteo domain.LoteoSummary
		if err := rows.Scan(
			&loteo.ID, &loteo.Name, &loteo.Location, &loteo.Description,
			&loteo.ManzanaCount, &loteo.LoteCount, &loteo.CalleCount,
			&loteo.HasPlan, &loteo.HasDxfFile, &loteo.CreatedAt,
		); err != nil {
			return nil, err
		}
		loteos = append(loteos, loteo)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return loteos, nil
}

// Get returns one loteo with its plan. A missing loteo, an unparseable id, or
// a loteo outside the caller's scope all read as domain.ErrLoteoNotFound.
func (repository *LoteoRepository) Get(
	ctx context.Context,
	loteoID string,
	scope gateway.LoteoScope,
) (domain.Loteo, error) {
	var (
		loteo       domain.Loteo
		boundaryWKT *string
	)
	err := repository.pool.QueryRow(ctx, getLoteoSQL,
		loteoID, scope.AssigneeAuthProviderID, scope.ByUserAssignment, scope.ByAgencyAssignment,
	).Scan(
		&loteo.ID, &loteo.Name, &loteo.Location, &loteo.Description,
		&loteo.CreatedAt, &boundaryWKT,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Loteo{}, domain.ErrLoteoNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return domain.Loteo{}, domain.ErrLoteoNotFound
		}
		return domain.Loteo{}, err
	}

	if loteo.Boundary, err = polygonFromNullableWKT(boundaryWKT); err != nil {
		return domain.Loteo{}, err
	}
	if loteo.Manzanas, err = repository.getManzanas(ctx, loteo.ID); err != nil {
		return domain.Loteo{}, err
	}
	if loteo.Lotes, err = repository.getLotes(ctx, loteo.ID); err != nil {
		return domain.Loteo{}, err
	}
	if loteo.Calles, err = repository.getCalles(ctx, loteo.ID); err != nil {
		return domain.Loteo{}, err
	}

	return loteo, nil
}

func (repository *LoteoRepository) getManzanas(ctx context.Context, loteoID string) ([]domain.Manzana, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT m.id::text, COALESCE(m.numero, ''),
		       m.tiene_agua, m.tiene_cloaca, m.tiene_luz, m.tiene_gas,
		       ST_AsText(de.geom),
		       COALESCE(
		         (SELECT array_agg(mc.calle_id::text ORDER BY mc.fecha_creacion, mc.id)
		          FROM manzana_calles mc
		          JOIN calles c ON c.id = mc.calle_id AND c.loteo_id = m.loteo_id AND c.fecha_baja IS NULL
		          WHERE mc.manzana_id = m.id AND mc.loteo_id = m.loteo_id AND mc.fecha_baja IS NULL),
		         '{}'::text[]
		       )
		FROM manzanas m
		LEFT JOIN dxf_entidades de ON de.id = m.dxf_entidad_id
		WHERE m.loteo_id = $1::uuid AND m.fecha_baja IS NULL
		ORDER BY m.fecha_creacion, m.id
	`, loteoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	manzanas := make([]domain.Manzana, 0)
	for rows.Next() {
		manzana, err := scanManzana(rows)
		if err != nil {
			return nil, err
		}
		manzanas = append(manzanas, manzana)
	}

	return manzanas, rows.Err()
}

type manzanaRow interface {
	Scan(dest ...any) error
}

func scanManzana(row manzanaRow) (domain.Manzana, error) {
	var (
		manzana domain.Manzana
		wkt     *string
	)
	if err := row.Scan(
		&manzana.ID, &manzana.Number,
		&manzana.HasWater, &manzana.HasSewer, &manzana.HasPower, &manzana.HasGas,
		&wkt, &manzana.CalleIDs,
	); err != nil {
		return domain.Manzana{}, err
	}
	if manzana.CalleIDs == nil {
		manzana.CalleIDs = []string{}
	}
	polygon, err := polygonFromNullableWKT(wkt)
	if err != nil {
		return domain.Manzana{}, err
	}
	manzana.Polygon = polygon
	return manzana, nil
}

const getManzanaSQL = `
		SELECT m.id::text, COALESCE(m.numero, ''),
		       m.tiene_agua, m.tiene_cloaca, m.tiene_luz, m.tiene_gas,
		       ST_AsText(de.geom),
		       COALESCE(
		         (SELECT array_agg(mc.calle_id::text ORDER BY mc.fecha_creacion, mc.id)
		          FROM manzana_calles mc
		          JOIN calles c ON c.id = mc.calle_id AND c.loteo_id = m.loteo_id AND c.fecha_baja IS NULL
		          WHERE mc.manzana_id = m.id AND mc.loteo_id = m.loteo_id AND mc.fecha_baja IS NULL),
		         '{}'::text[]
		       )
		FROM manzanas m
		LEFT JOIN dxf_entidades de ON de.id = m.dxf_entidad_id
		WHERE m.id = $2::uuid AND m.loteo_id = $1::uuid AND m.fecha_baja IS NULL
	`

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func getManzanaWithQueryer(ctx context.Context, queryer queryRower, loteoID, manzanaID string) (domain.Manzana, error) {
	manzana, err := scanManzana(queryer.QueryRow(ctx, getManzanaSQL, loteoID, manzanaID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Manzana{}, domain.ErrManzanaNotFound
	}
	return manzana, err
}

func (repository *LoteoRepository) getLotes(ctx context.Context, loteoID string) ([]domain.Lote, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT lo.id::text, lo.manzana_id::text, COALESCE(lo.numero, ''),
		       lo.precio::float8, COALESCE(lo.moneda, ''), lo.superficie::float8,
		       COALESCE(lo.caracteristicas, ''), ST_AsText(de.geom)
		FROM lotes lo
		LEFT JOIN dxf_entidades de ON de.id = lo.dxf_entidad_id
		WHERE lo.loteo_id = $1::uuid AND lo.fecha_baja IS NULL
		ORDER BY lo.fecha_creacion, lo.id
	`, loteoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lotes := make([]domain.Lote, 0)
	for rows.Next() {
		var (
			lote domain.Lote
			wkt  *string
		)
		if err := rows.Scan(
			&lote.ID, &lote.ManzanaID, &lote.Number,
			&lote.Price, &lote.Currency, &lote.Area,
			&lote.Features, &wkt,
		); err != nil {
			return nil, err
		}
		if lote.Polygon, err = polygonFromNullableWKT(wkt); err != nil {
			return nil, err
		}
		lotes = append(lotes, lote)
	}

	return lotes, rows.Err()
}

func (repository *LoteoRepository) getCalles(ctx context.Context, loteoID string) ([]domain.Calle, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT c.id::text, COALESCE(c.nombre, ''), COALESCE(c.tipo, ''), ST_AsText(de.geom)
		FROM calles c
		LEFT JOIN dxf_entidades de ON de.id = c.dxf_entidad_id
		WHERE c.loteo_id = $1::uuid AND c.fecha_baja IS NULL
		ORDER BY c.fecha_creacion, c.id
	`, loteoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	calles := make([]domain.Calle, 0)
	for rows.Next() {
		var (
			calle domain.Calle
			wkt   *string
		)
		if err := rows.Scan(&calle.ID, &calle.Name, &calle.Type, &wkt); err != nil {
			return nil, err
		}
		if calle.Polygon, err = polygonFromNullableWKT(wkt); err != nil {
			return nil, err
		}
		calles = append(calles, calle)
	}

	return calles, rows.Err()
}

const getCalleSQL = `
		SELECT c.id::text, COALESCE(c.nombre, ''), COALESCE(c.tipo, ''), ST_AsText(de.geom)
		FROM calles c
		LEFT JOIN dxf_entidades de ON de.id = c.dxf_entidad_id
		WHERE c.id = $2::uuid AND c.loteo_id = $1::uuid AND c.fecha_baja IS NULL
	`

func getCalleWithQueryer(ctx context.Context, queryer queryRower, loteoID, calleID string) (domain.Calle, error) {
	var (
		calle domain.Calle
		wkt   *string
	)
	err := queryer.QueryRow(ctx, getCalleSQL, loteoID, calleID).Scan(&calle.ID, &calle.Name, &calle.Type, &wkt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Calle{}, domain.ErrCalleNotFound
	}
	if err != nil {
		return domain.Calle{}, err
	}
	if calle.Polygon, err = polygonFromNullableWKT(wkt); err != nil {
		return domain.Calle{}, err
	}
	return calle, nil
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

func (repository *LoteoRepository) UpdateManzana(
	ctx context.Context,
	actorAuthProviderID, loteoID, manzanaID string,
	data domain.ManzanaData,
) (domain.Manzana, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Manzana{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE manzanas
		SET numero = $4,
		    tiene_agua = $5,
		    tiene_cloaca = $6,
		    tiene_luz = $7,
		    tiene_gas = $8,
		    usuario_modificacion = (SELECT id FROM usuarios WHERE auth_provider_id = $1::uuid),
		    fecha_modificacion = now()
		WHERE id = $3::uuid AND loteo_id = $2::uuid AND fecha_baja IS NULL
	`, actorAuthProviderID, loteoID, manzanaID,
		data.Number, data.HasWater, data.HasSewer, data.HasPower, data.HasGas)
	if err != nil {
		return domain.Manzana{}, mapManzanaUpdateError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Manzana{}, domain.ErrManzanaNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE manzana_calles
		SET fecha_baja = now(),
		    usuario_modificacion = (SELECT id FROM usuarios WHERE auth_provider_id = $1::uuid),
		    fecha_modificacion = now()
		WHERE manzana_id = $3::uuid AND loteo_id = $2::uuid AND fecha_baja IS NULL
	`, actorAuthProviderID, loteoID, manzanaID); err != nil {
		return domain.Manzana{}, err
	}

	for _, calleID := range data.CalleIDs {
		tag, err := tx.Exec(ctx, `
			INSERT INTO manzana_calles (manzana_id, calle_id, loteo_id, usuario_modificacion)
			SELECT $3::uuid, c.id, $2::uuid,
			       (SELECT id FROM usuarios WHERE auth_provider_id = $1::uuid)
			FROM calles c
			WHERE c.id = $4::uuid AND c.loteo_id = $2::uuid AND c.fecha_baja IS NULL
		`, actorAuthProviderID, loteoID, manzanaID, calleID)
		if err != nil {
			return domain.Manzana{}, mapManzanaCalleError(err)
		}
		if tag.RowsAffected() == 0 {
			return domain.Manzana{}, domain.ErrUnknownCalle
		}
	}

	manzana, err := getManzanaWithQueryer(ctx, tx, loteoID, manzanaID)
	if err != nil {
		return domain.Manzana{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Manzana{}, err
	}

	return manzana, nil
}

func mapManzanaUpdateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolationCode:
			return domain.ErrManzanaNumberInUse
		case invalidTextRepresentationCode:
			return domain.ErrManzanaNotFound
		}
	}
	return err
}

func mapManzanaCalleError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case foreignKeyViolationCode, invalidTextRepresentationCode:
			return domain.ErrUnknownCalle
		}
	}
	return err
}

func (repository *LoteoRepository) UpdateCalle(
	ctx context.Context,
	actorAuthProviderID, loteoID, calleID string,
	data domain.CalleData,
) (domain.Calle, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return domain.Calle{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE calles
		SET nombre = $4,
		    tipo = NULLIF($5, ''),
		    usuario_modificacion = (SELECT id FROM usuarios WHERE auth_provider_id = $1::uuid),
		    fecha_modificacion = now()
		WHERE id = $3::uuid AND loteo_id = $2::uuid AND fecha_baja IS NULL
	`, actorAuthProviderID, loteoID, calleID, data.Name, data.Type)
	if err != nil {
		return domain.Calle{}, mapCalleUpdateError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Calle{}, domain.ErrCalleNotFound
	}

	calle, err := getCalleWithQueryer(ctx, tx, loteoID, calleID)
	if err != nil {
		return domain.Calle{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Calle{}, err
	}

	return calle, nil
}

func mapCalleUpdateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case checkViolationCode:
			return domain.ErrInvalidCalleType
		case invalidTextRepresentationCode:
			return domain.ErrCalleNotFound
		}
	}
	return err
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
