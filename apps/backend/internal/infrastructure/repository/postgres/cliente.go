package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
)

const clienteColumns = `id::text, nombre, apellido, dni,
	coalesce(celular, ''), coalesce(email, ''), fecha_creacion, fecha_modificacion`

type ClientRepository struct {
	pool *pgxpool.Pool
}

func NewClientRepository(pool *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{pool: pool}
}

func (repository *ClientRepository) Create(
	ctx context.Context,
	cliente domain.Cliente,
	usuarioModificacion string,
) (domain.Cliente, error) {
	var created domain.Cliente

	err := repository.pool.QueryRow(ctx, `
		INSERT INTO clientes (nombre, apellido, dni, celular, email, usuario_modificacion)
		VALUES ($1, $2, $3, nullif($4, ''), nullif($5, ''), $6::uuid)
		RETURNING `+clienteColumns,
		cliente.Nombre, cliente.Apellido, cliente.DNI, cliente.Celular, cliente.Email, usuarioModificacion,
	).Scan(scanTargets(&created)...)
	if err != nil {
		if isDuplicateDNI(err) {
			return domain.Cliente{}, domain.ErrDNIEnUso
		}
		return domain.Cliente{}, err
	}

	return created, nil
}

// List returns active clients, optionally filtered by name, surname or DNI.
func (repository *ClientRepository) List(ctx context.Context, buscar string) ([]domain.Cliente, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT `+clienteColumns+`
		FROM clientes
		WHERE fecha_baja IS NULL
			AND ($1 = '' OR nombre || ' ' || apellido ILIKE '%' || $1 || '%' OR dni ILIKE '%' || $1 || '%')
		ORDER BY apellido, nombre
	`, buscar)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clientes := make([]domain.Cliente, 0)
	for rows.Next() {
		var cliente domain.Cliente
		if err := rows.Scan(scanTargets(&cliente)...); err != nil {
			return nil, err
		}
		clientes = append(clientes, cliente)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return clientes, nil
}

func (repository *ClientRepository) Update(
	ctx context.Context,
	cliente domain.Cliente,
	usuarioModificacion string,
) (domain.Cliente, error) {
	var updated domain.Cliente

	err := repository.pool.QueryRow(ctx, `
		UPDATE clientes
		SET nombre = $2, apellido = $3, dni = $4, celular = nullif($5, ''),
			email = nullif($6, ''), usuario_modificacion = $7::uuid, fecha_modificacion = now()
		WHERE id = $1::uuid AND fecha_baja IS NULL
		RETURNING `+clienteColumns,
		cliente.ID, cliente.Nombre, cliente.Apellido, cliente.DNI,
		cliente.Celular, cliente.Email, usuarioModificacion,
	).Scan(scanTargets(&updated)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Cliente{}, domain.ErrClienteNoEncontrado
	}
	if err != nil {
		if isDuplicateDNI(err) {
			return domain.Cliente{}, domain.ErrDNIEnUso
		}
		return domain.Cliente{}, err
	}

	return updated, nil
}

// SoftDelete marks the client as inactive by setting fecha_baja, which also
// releases its DNI: clientes_dni_idx only covers rows still active.
func (repository *ClientRepository) SoftDelete(ctx context.Context, id, usuarioModificacion string) error {
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

func scanTargets(cliente *domain.Cliente) []any {
	return []any{
		&cliente.ID, &cliente.Nombre, &cliente.Apellido, &cliente.DNI,
		&cliente.Celular, &cliente.Email, &cliente.FechaCreacion, &cliente.FechaModificacion,
	}
}

func isDuplicateDNI(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
