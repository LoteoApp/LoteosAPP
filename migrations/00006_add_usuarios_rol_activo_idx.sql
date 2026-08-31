-- +goose Up
CREATE INDEX usuarios_rol_activo_idx ON usuarios (rol)
    WHERE fecha_baja IS NULL;

-- +goose Down
DROP INDEX IF EXISTS usuarios_rol_activo_idx;
