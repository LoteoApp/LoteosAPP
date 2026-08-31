package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
)

// dniUniqueConstraint is the name of the unique index backing "one active
// cliente per DNI" (see migrations/00005_create_entity_model.sql). Only a
// 23505 violation on this specific constraint means "DNI ya está en uso";
// any other unique violation on the clientes table is a different business
// rule and must not be misreported as a duplicate DNI.
const dniUniqueConstraint = "clientes_dni_idx"

type ClienteRepository struct {
	pool *pgxpool.Pool
}

func NewClienteRepository(pool *pgxpool.Pool) *ClienteRepository {
	return &ClienteRepository{pool: pool}
}

func (repository *ClienteRepository) Create(ctx context.Context, cliente domain.Cliente) (domain.Cliente, error) {
	var created domain.Cliente

	err := repository.pool.QueryRow(ctx, `
		INSERT INTO clientes (nombre, apellido, dni, celular, email, usuario_modificacion)
		VALUES ($1, $2, $3, $4, $5, $6::uuid)
		RETURNING id::text, nombre, apellido, dni, celular, email, fecha_creacion, fecha_modificacion
	`, cliente.Nombre, cliente.Apellido, cliente.DNI, cliente.Celular, cliente.Email, cliente.UsuarioModificacion).Scan(
		&created.ID, &created.Nombre, &created.Apellido, &created.DNI,
		&created.Celular, &created.Email, &created.FechaCreacion, &created.FechaModificacion,
	)
	if err != nil {
		return domain.Cliente{}, mapClienteWriteError(err)
	}

	return created, nil
}

// Update applies a partial change to an existing active cliente. A nil
// field on update is left unchanged via COALESCE — that's what gives the
// PATCH /api/v1/clientes/{id} route correct partial-update semantics
// instead of silently wiping fields the caller didn't send.
func (repository *ClienteRepository) Update(ctx context.Context, update domain.ClienteUpdate) (domain.Cliente, error) {
	var updated domain.Cliente

	err := repository.pool.QueryRow(ctx, `
		UPDATE clientes
		SET nombre = COALESCE($2, nombre),
			apellido = COALESCE($3, apellido),
			dni = COALESCE($4, dni),
			celular = COALESCE($5, celular),
			email = COALESCE($6, email),
			usuario_modificacion = $7::uuid,
			fecha_modificacion = now()
		WHERE id = $1::uuid AND fecha_baja IS NULL
		RETURNING id::text, nombre, apellido, dni, celular, email, fecha_creacion, fecha_modificacion
	`, update.ID, update.Nombre, update.Apellido, update.DNI, update.Celular, update.Email, update.UsuarioModificacion).Scan(
		&updated.ID, &updated.Nombre, &updated.Apellido, &updated.DNI,
		&updated.Celular, &updated.Email, &updated.FechaCreacion, &updated.FechaModificacion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Cliente{}, domain.ErrClienteNoEncontrado
	}
	if err != nil {
		return domain.Cliente{}, mapClienteWriteError(err)
	}

	return updated, nil
}

func (repository *ClienteRepository) SoftDelete(ctx context.Context, id, usuarioModificacion string) error {
	tag, err := repository.pool.Exec(ctx, `
		UPDATE clientes
		SET fecha_baja = now(), usuario_modificacion = $2::uuid, fecha_modificacion = now()
		WHERE id = $1::uuid AND fecha_baja IS NULL
	`, id, usuarioModificacion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrClienteNoEncontrado
	}

	return nil
}

func (repository *ClienteRepository) List(ctx context.Context, search string) ([]domain.Cliente, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id::text, nombre, apellido, dni, celular, email, fecha_creacion, fecha_modificacion
		FROM clientes
		WHERE fecha_baja IS NULL
			AND ($1 = '' OR nombre ILIKE $2 ESCAPE '\' OR apellido ILIKE $2 ESCAPE '\' OR dni ILIKE $2 ESCAPE '\')
		ORDER BY apellido, nombre
	`, search, containsPattern(search))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialized empty rather than nil so a search with no matches
	// serializes as `"clientes": []`, not `"clientes": null`.
	clientes := make([]domain.Cliente, 0)
	for rows.Next() {
		var cliente domain.Cliente
		if err := rows.Scan(
			&cliente.ID, &cliente.Nombre, &cliente.Apellido, &cliente.DNI,
			&cliente.Celular, &cliente.Email, &cliente.FechaCreacion, &cliente.FechaModificacion,
		); err != nil {
			return nil, err
		}
		clientes = append(clientes, cliente)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return clientes, nil
}

// mapClienteWriteError translates a PostgreSQL unique-violation on the
// clientes table into the right domain error. Only pgErr.ConstraintName ==
// dniUniqueConstraint means "DNI ya está en uso"; any other unique
// violation is reported as a generic conflict with the original error kept
// as Cause, instead of being misclassified as a duplicate DNI. Any
// non-constraint error is returned unchanged, so it reaches
// response.WriteError as an unexpected failure: logged, hidden behind a
// generic 500.
func mapClienteWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != uniqueViolationCode {
		return err
	}

	if pgErr.ConstraintName == dniUniqueConstraint {
		return domain.ErrDNIEnUso
	}

	return &domain.Error{
		Kind:    domain.KindConflict,
		Code:    "unique_violation",
		Message: "El registro entra en conflicto con datos existentes",
		Cause:   err,
	}
}

// likeWildcards escapes the characters LIKE/ILIKE treat as wildcards, so a
// search for "%" matches a literal percent sign instead of every cliente.
// The escape character itself has to be doubled, and matches the ESCAPE
// clause of the queries using this pattern.
var likeWildcards = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func containsPattern(search string) string {
	return "%" + likeWildcards.Replace(search) + "%"
}
