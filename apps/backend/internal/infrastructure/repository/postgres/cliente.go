package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
)

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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return domain.Cliente{}, domain.ErrDNIEnUso
		}
		return domain.Cliente{}, err
	}

	return created, nil
}

func (repository *ClienteRepository) Update(ctx context.Context, cliente domain.Cliente) (domain.Cliente, error) {
	var updated domain.Cliente

	err := repository.pool.QueryRow(ctx, `
		UPDATE clientes
		SET nombre = $2, apellido = $3, dni = $4, celular = $5, email = $6,
			usuario_modificacion = $7::uuid, fecha_modificacion = now()
		WHERE id = $1::uuid AND fecha_baja IS NULL
		RETURNING id::text, nombre, apellido, dni, celular, email, fecha_creacion, fecha_modificacion
	`, cliente.ID, cliente.Nombre, cliente.Apellido, cliente.DNI, cliente.Celular, cliente.Email, cliente.UsuarioModificacion).Scan(
		&updated.ID, &updated.Nombre, &updated.Apellido, &updated.DNI,
		&updated.Celular, &updated.Email, &updated.FechaCreacion, &updated.FechaModificacion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Cliente{}, domain.ErrClienteNoEncontrado
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return domain.Cliente{}, domain.ErrDNIEnUso
		}
		return domain.Cliente{}, err
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
			AND ($1 = '' OR nombre ILIKE '%' || $1 || '%' OR apellido ILIKE '%' || $1 || '%' OR dni ILIKE '%' || $1 || '%')
		ORDER BY apellido, nombre
	`, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clientes []domain.Cliente
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
