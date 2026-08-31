package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
)

const uniqueViolationCode = "23505"

const usuarioColumns = `id::text, auth_provider_id::text, email, nombre, apellido, rol, perfil_completo, fecha_baja, created_at`

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (repository *UserRepository) Create(ctx context.Context, usuario domain.Usuario) (domain.Usuario, error) {
	var created domain.Usuario

	err := repository.pool.QueryRow(ctx, `
		INSERT INTO usuarios (auth_provider_id, email, nombre, apellido, rol, perfil_completo)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		RETURNING `+usuarioColumns, usuario.AuthProviderID, usuario.Email, usuario.Nombre,
		usuario.Apellido, usuario.Rol, usuario.PerfilCompleto).Scan(scanTargets(&created)...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return domain.Usuario{}, domain.ErrEmailEnUso
		}
		return domain.Usuario{}, err
	}

	return created, nil
}

func (repository *UserRepository) FindByAuthProviderID(ctx context.Context, authProviderID string) (domain.Usuario, error) {
	var usuario domain.Usuario

	err := repository.pool.QueryRow(ctx, `
		SELECT `+usuarioColumns+`
		FROM usuarios
		WHERE auth_provider_id = $1::uuid
	`, authProviderID).Scan(scanTargets(&usuario)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
	}
	if err != nil {
		return domain.Usuario{}, err
	}

	return usuario, nil
}

func (repository *UserRepository) FindByID(ctx context.Context, id string) (domain.Usuario, error) {
	var usuario domain.Usuario

	err := repository.pool.QueryRow(ctx, `
		SELECT `+usuarioColumns+`
		FROM usuarios
		WHERE id = $1::uuid
	`, id).Scan(scanTargets(&usuario)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
	}
	if err != nil {
		return domain.Usuario{}, err
	}

	return usuario, nil
}

func (repository *UserRepository) UpdateProfile(ctx context.Context, authProviderID, nombre, apellido string) (domain.Usuario, error) {
	var usuario domain.Usuario

	err := repository.pool.QueryRow(ctx, `
		UPDATE usuarios
		SET nombre = $2, apellido = $3, perfil_completo = true, updated_at = now()
		WHERE auth_provider_id = $1::uuid
		RETURNING `+usuarioColumns, authProviderID, nombre, apellido).Scan(scanTargets(&usuario)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
	}
	if err != nil {
		return domain.Usuario{}, err
	}

	return usuario, nil
}

func (repository *UserRepository) ListByRol(ctx context.Context, rol domain.Rol, includeInactive bool) ([]domain.Usuario, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT `+usuarioColumns+`
		FROM usuarios
		WHERE rol = $1
			AND ($2 OR fecha_baja IS NULL)
		ORDER BY apellido, nombre, email
	`, rol, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialized empty rather than nil so an empty result serializes as
	// `"agrimensores": []`, not `"agrimensores": null`.
	usuarios := make([]domain.Usuario, 0)
	for rows.Next() {
		var usuario domain.Usuario
		if err := rows.Scan(scanTargets(&usuario)...); err != nil {
			return nil, err
		}
		usuarios = append(usuarios, usuario)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return usuarios, nil
}

// Update applies a partial change to an active user. A nil field is left
// unchanged via COALESCE, which is what gives PATCH its partial-update
// semantics instead of wiping the fields the caller didn't send.
func (repository *UserRepository) Update(ctx context.Context, update domain.UsuarioUpdate) (domain.Usuario, error) {
	var usuario domain.Usuario

	err := repository.pool.QueryRow(ctx, `
		UPDATE usuarios
		SET nombre = COALESCE($2, nombre),
			apellido = COALESCE($3, apellido),
			perfil_completo = perfil_completo
				OR (COALESCE($2, nombre) <> '' AND COALESCE($3, apellido) <> ''),
			usuario_modificacion = $4::uuid,
			updated_at = now()
		WHERE id = $1::uuid AND fecha_baja IS NULL
		RETURNING `+usuarioColumns, update.ID, update.Nombre, update.Apellido, update.UsuarioModificacion).
		Scan(scanTargets(&usuario)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
	}
	if err != nil {
		return domain.Usuario{}, err
	}

	return usuario, nil
}

func (repository *UserRepository) SoftDelete(ctx context.Context, id, usuarioModificacion string) error {
	tag, err := repository.pool.Exec(ctx, `
		UPDATE usuarios
		SET fecha_baja = now(), usuario_modificacion = $2::uuid, updated_at = now()
		WHERE id = $1::uuid AND fecha_baja IS NULL
	`, id, usuarioModificacion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUsuarioNoEncontrado
	}

	return nil
}

func scanTargets(usuario *domain.Usuario) []any {
	return []any{
		&usuario.ID, &usuario.AuthProviderID, &usuario.Email, &usuario.Nombre,
		&usuario.Apellido, &usuario.Rol, &usuario.PerfilCompleto, &usuario.FechaBaja,
		&usuario.CreatedAt,
	}
}
