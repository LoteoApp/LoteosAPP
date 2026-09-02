-- +goose Up
CREATE UNIQUE INDEX inmobiliarias_cuit_idx ON inmobiliarias (cuit)
    WHERE cuit IS NOT NULL AND fecha_baja IS NULL;

-- +goose Down
DROP INDEX IF EXISTS inmobiliarias_cuit_idx;
