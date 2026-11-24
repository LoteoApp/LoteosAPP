-- +goose Up

-- The ABM already stores a CUIT as 11 digits, and 00007 normalized the rows
-- that existed then. Rows loaded by hand afterwards can still carry
-- separators, and the unique index compares text: "30-71234567-8" and
-- "30712345678" coexist as two active inmobiliarias. The check below makes the
-- database refuse anything but 11 digits, so the index always compares
-- comparable values.
--
-- Both problems are detected on the normalized value before anything is
-- rewritten. A value that isn't 11 digits can't be repaired without guessing,
-- and a duplicate can't be resolved without knowing which agency is the real
-- one, so either stops the migration instead of being silently altered.
-- +goose StatementBegin
DO $$
DECLARE
    invalid_count bigint;
    duplicate_count bigint;
BEGIN
    SELECT count(*) INTO invalid_count
    FROM inmobiliarias
    WHERE cuit IS NOT NULL AND regexp_replace(cuit, '[^0-9]', '', 'g') !~ '^[0-9]{11}$';

    IF invalid_count > 0 THEN
        RAISE EXCEPTION 'inmobiliarias: % row(s) have a cuit that is not 11 digits once normalized; fix them before applying this migration', invalid_count;
    END IF;

    SELECT count(*) INTO duplicate_count
    FROM (
        SELECT regexp_replace(cuit, '[^0-9]', '', 'g') AS normalized
        FROM inmobiliarias
        WHERE cuit IS NOT NULL AND fecha_baja IS NULL
        GROUP BY normalized
        HAVING count(*) > 1
    ) AS duplicates;

    IF duplicate_count > 0 THEN
        RAISE EXCEPTION 'inmobiliarias: % cuit value(s) are shared by more than one active row once normalized; resolve them before applying this migration', duplicate_count;
    END IF;
END
$$;
-- +goose StatementEnd

UPDATE inmobiliarias
SET cuit = regexp_replace(cuit, '[^0-9]', '', 'g'),
    fecha_modificacion = now()
WHERE cuit IS NOT NULL
    AND cuit <> regexp_replace(cuit, '[^0-9]', '', 'g');

ALTER TABLE inmobiliarias
    ADD CONSTRAINT inmobiliarias_cuit_format_check
    CHECK (cuit IS NULL OR cuit ~ '^[0-9]{11}$');

-- +goose Down

-- The normalization is not undone: the separators the rows used to carry are
-- not recorded anywhere, and a plain-digit cuit is valid with or without the
-- constraint.
ALTER TABLE inmobiliarias
    DROP CONSTRAINT IF EXISTS inmobiliarias_cuit_format_check;
