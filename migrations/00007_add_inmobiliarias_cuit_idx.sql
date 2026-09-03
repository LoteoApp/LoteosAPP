-- +goose Up

-- The ABM stores a CUIT as 11 digits, but nothing enforced that before this
-- migration, so a row loaded by hand can still carry separators. Normalizing
-- first is what makes the index below comparable: without it "30-71234567-8"
-- and "30712345678" would coexist as two different active inmobiliarias.
UPDATE inmobiliarias
SET cuit = regexp_replace(cuit, '[^0-9]', '', 'g'),
    fecha_modificacion = now()
WHERE cuit IS NOT NULL
    AND cuit <> regexp_replace(cuit, '[^0-9]', '', 'g');

-- A value that isn't 11 digits after normalizing can't be repaired here
-- without guessing, and a duplicate can't be resolved without knowing which
-- agency is the real one. Both stop the migration instead of being silently
-- dropped or deduplicated.
-- +goose StatementBegin
DO $$
DECLARE
    invalid_count bigint;
    duplicate_count bigint;
BEGIN
    SELECT count(*) INTO invalid_count
    FROM inmobiliarias
    WHERE cuit IS NOT NULL AND fecha_baja IS NULL AND cuit !~ '^[0-9]{11}$';

    IF invalid_count > 0 THEN
        RAISE EXCEPTION 'inmobiliarias: % active row(s) have a cuit that is not 11 digits; fix them before applying this migration', invalid_count;
    END IF;

    SELECT count(*) INTO duplicate_count
    FROM (
        SELECT cuit
        FROM inmobiliarias
        WHERE cuit IS NOT NULL AND fecha_baja IS NULL
        GROUP BY cuit
        HAVING count(*) > 1
    ) AS duplicates;

    IF duplicate_count > 0 THEN
        RAISE EXCEPTION 'inmobiliarias: % cuit value(s) are shared by more than one active row; resolve them before applying this migration', duplicate_count;
    END IF;
END
$$;
-- +goose StatementEnd

CREATE UNIQUE INDEX inmobiliarias_cuit_idx ON inmobiliarias (cuit)
    WHERE cuit IS NOT NULL AND fecha_baja IS NULL;

-- +goose Down

-- The normalization is not undone: the separators the rows used to carry are
-- not recorded anywhere, and a plain-digit cuit is valid with or without this
-- index.
DROP INDEX IF EXISTS inmobiliarias_cuit_idx;
