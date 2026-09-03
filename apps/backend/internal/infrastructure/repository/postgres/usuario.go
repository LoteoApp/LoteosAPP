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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
		}
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

// ListByRoles returns the users holding any of roles, ordered by apellido
// and nombre. An empty roles slice matches nothing rather than every role,
// so a caller can't accidentally list the whole table by forgetting to
// scope it.
func (repository *UserRepository) ListByRoles(ctx context.Context, roles []domain.Rol, includeInactive bool) ([]domain.Usuario, error) {
	if len(roles) == 0 {
		return []domain.Usuario{}, nil
	}

	rolNames := make([]string, len(roles))
	for i, rol := range roles {
		rolNames[i] = string(rol)
	}

	rows, err := repository.pool.Query(ctx, `
		SELECT `+usuarioColumns+`
		FROM usuarios
		WHERE rol = ANY($1)
			AND ($2 OR fecha_baja IS NULL)
		ORDER BY apellido, nombre
	`, rolNames, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialized empty rather than nil so an empty result serializes as
	// `"usuarios": []`, not `"usuarios": null`.
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
			usuario_modificacion = $4::uuid,
			updated_at = now()
		WHERE id = $1::uuid AND fecha_baja IS NULL
		RETURNING `+usuarioColumns, update.ID, update.Nombre, update.Apellido, update.UsuarioModificacion).
		Scan(scanTargets(&usuario)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
		}
		return domain.Usuario{}, err
	}

	return usuario, nil
}

// SoftDelete gives a user de baja. A retry after a lost response lands on a
// row that's already inactive: reconcileMissingUpdate distinguishes that
// (domain.ErrUsuarioDadoDeBaja, so a caller can treat it as already done)
// from the row not existing at all (domain.ErrUsuarioNoEncontrado), closing
// the race between DeactivateUser's own read of the row and this write.
func (repository *UserRepository) SoftDelete(ctx context.Context, id, usuarioModificacion string) error {
	tag, err := repository.pool.Exec(ctx, `
		UPDATE usuarios
		SET fecha_baja = now(), usuario_modificacion = $2::uuid, updated_at = now()
		WHERE id = $1::uuid AND fecha_baja IS NULL
	`, id, usuarioModificacion)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return domain.ErrUsuarioNoEncontrado
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return repository.reconcileMissingUpdate(ctx, id, domain.ErrUsuarioDadoDeBaja)
	}

	return nil
}

// Reactivate clears fecha_baja on an inactive user, the inverse of
// SoftDelete. It only matches a currently inactive row; a retry that lands
// on an already-active row is reconciled the same way SoftDelete reconciles
// an already-inactive one (see reconcileMissingUpdate).
func (repository *UserRepository) Reactivate(ctx context.Context, id, usuarioModificacion string) error {
	tag, err := repository.pool.Exec(ctx, `
		UPDATE usuarios
		SET fecha_baja = NULL, usuario_modificacion = $2::uuid, updated_at = now()
		WHERE id = $1::uuid AND fecha_baja IS NOT NULL
	`, id, usuarioModificacion)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentationCode {
			return domain.ErrUsuarioNoEncontrado
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return repository.reconcileMissingUpdate(ctx, id, domain.ErrUsuarioYaActivo)
	}

	return nil
}

// reconcileMissingUpdate runs after a state-guarded UPDATE (SoftDelete or
// Reactivate) affects no rows. id is syntactically valid at this point — an
// invalid one is already caught before Exec returns a row count — so 0 rows
// affected means either id doesn't exist, or it exists but was already in
// the state the caller wanted (alreadyInTargetState), which is what makes a
// retried baja/reactivación idempotent instead of a false "not found".
func (repository *UserRepository) reconcileMissingUpdate(ctx context.Context, id string, alreadyInTargetState error) error {
	var exists bool
	err := repository.pool.QueryRow(ctx, `SELECT true FROM usuarios WHERE id = $1::uuid`, id).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrUsuarioNoEncontrado
	}
	if err != nil {
		return err
	}

	return alreadyInTargetState
}

func scanTargets(usuario *domain.Usuario) []any {
	return []any{
		&usuario.ID, &usuario.AuthProviderID, &usuario.Email, &usuario.Nombre,
		&usuario.Apellido, &usuario.Rol, &usuario.PerfilCompleto, &usuario.FechaBaja,
		&usuario.CreatedAt,
	}
}
