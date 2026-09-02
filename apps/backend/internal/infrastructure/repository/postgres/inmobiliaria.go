package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
)

// cuitUniqueConstraint is the name of the unique index backing "one active
// inmobiliaria per CUIT" (see
// migrations/00007_add_inmobiliarias_cuit_idx.sql). Only a 23505 violation
// on this specific constraint means "CUIT ya está en uso"; any other unique
// violation on the inmobiliarias table is a different business rule and must
// not be misreported as a duplicate CUIT.
const cuitUniqueConstraint = "inmobiliarias_cuit_idx"

type InmobiliariaRepository struct {
	pool *pgxpool.Pool
}

func NewInmobiliariaRepository(pool *pgxpool.Pool) *InmobiliariaRepository {
	return &InmobiliariaRepository{pool: pool}
}

func (repository *InmobiliariaRepository) Create(ctx context.Context, inmobiliaria domain.Inmobiliaria) (domain.Inmobiliaria, error) {
	var created domain.Inmobiliaria

	err := repository.pool.QueryRow(ctx, `
		INSERT INTO inmobiliarias (razon_social, cuit, telefono, email, usuario_modificacion)
		VALUES ($1, $2, $3, $4, $5::uuid)
		RETURNING id::text, razon_social, cuit, telefono, email, fecha_creacion, fecha_modificacion
	`, inmobiliaria.RazonSocial, inmobiliaria.CUIT, inmobiliaria.Telefono, inmobiliaria.Email, inmobiliaria.UsuarioModificacion).Scan(
		&created.ID, &created.RazonSocial, &created.CUIT,
		&created.Telefono, &created.Email, &created.FechaCreacion, &created.FechaModificacion,
	)
	if err != nil {
		return domain.Inmobiliaria{}, mapInmobiliariaWriteError(err)
	}

	return created, nil
}

// Update applies a partial change to an existing active inmobiliaria. A nil
// field on update is left unchanged via COALESCE — that's what gives the
// PATCH /api/v1/inmobiliarias/{id} route correct partial-update semantics
// instead of silently wiping fields the caller didn't send.
func (repository *InmobiliariaRepository) Update(ctx context.Context, update domain.InmobiliariaUpdate) (domain.Inmobiliaria, error) {
	var updated domain.Inmobiliaria

	err := repository.pool.QueryRow(ctx, `
		UPDATE inmobiliarias
		SET razon_social = COALESCE($2, razon_social),
			cuit = COALESCE($3, cuit),
			telefono = COALESCE($4, telefono),
			email = COALESCE($5, email),
			usuario_modificacion = $6::uuid,
			fecha_modificacion = now()
		WHERE id = $1::uuid AND fecha_baja IS NULL
		RETURNING id::text, razon_social, cuit, telefono, email, fecha_creacion, fecha_modificacion
	`, update.ID, update.RazonSocial, update.CUIT, update.Telefono, update.Email, update.UsuarioModificacion).Scan(
		&updated.ID, &updated.RazonSocial, &updated.CUIT,
		&updated.Telefono, &updated.Email, &updated.FechaCreacion, &updated.FechaModificacion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Inmobiliaria{}, domain.ErrInmobiliariaNoEncontrada
	}
	if err != nil {
		return domain.Inmobiliaria{}, mapInmobiliariaWriteError(err)
	}

	return updated, nil
}

func (repository *InmobiliariaRepository) SoftDelete(ctx context.Context, id, usuarioModificacion string) error {
	tag, err := repository.pool.Exec(ctx, `
		UPDATE inmobiliarias
		SET fecha_baja = now(), usuario_modificacion = $2::uuid, fecha_modificacion = now()
		WHERE id = $1::uuid AND fecha_baja IS NULL
	`, id, usuarioModificacion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInmobiliariaNoEncontrada
	}

	return nil
}

func (repository *InmobiliariaRepository) List(ctx context.Context, search string) ([]domain.Inmobiliaria, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id::text, razon_social, cuit, telefono, email, fecha_creacion, fecha_modificacion
		FROM inmobiliarias
		WHERE fecha_baja IS NULL
			AND ($1 = '' OR razon_social ILIKE $2 ESCAPE '\' OR cuit ILIKE $2 ESCAPE '\')
		ORDER BY razon_social
	`, search, containsPattern(search))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialized empty rather than nil so a search with no matches
	// serializes as `"inmobiliarias": []`, not `"inmobiliarias": null`.
	inmobiliarias := make([]domain.Inmobiliaria, 0)
	for rows.Next() {
		var inmobiliaria domain.Inmobiliaria
		if err := rows.Scan(
			&inmobiliaria.ID, &inmobiliaria.RazonSocial, &inmobiliaria.CUIT,
			&inmobiliaria.Telefono, &inmobiliaria.Email, &inmobiliaria.FechaCreacion, &inmobiliaria.FechaModificacion,
		); err != nil {
			return nil, err
		}
		inmobiliarias = append(inmobiliarias, inmobiliaria)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return inmobiliarias, nil
}

// mapInmobiliariaWriteError translates a PostgreSQL unique-violation on the
// inmobiliarias table into the right domain error. Only pgErr.ConstraintName
// == cuitUniqueConstraint means "CUIT ya está en uso"; any other unique
// violation is reported as a generic conflict with the original error kept
// as Cause. Any non-constraint error is returned unchanged, so it reaches
// response.WriteError as an unexpected failure: logged, hidden behind a
// generic 500.
func mapInmobiliariaWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != uniqueViolationCode {
		return err
	}

	if pgErr.ConstraintName == cuitUniqueConstraint {
		return domain.ErrCUITEnUso
	}

	return &domain.Error{
		Kind:    domain.KindConflict,
		Code:    "unique_violation",
		Message: "El registro entra en conflicto con datos existentes",
		Cause:   err,
	}
}
