-- +goose Up
ALTER TABLE reservas
    ADD COLUMN inmobiliaria_id UUID REFERENCES inmobiliarias (id);

-- Preserve the agency that was responsible for historical reservations when
-- that information can be recovered from the seller. Reservations created by
-- administrative users without an agency intentionally remain NULL.
UPDATE reservas r
SET inmobiliaria_id = u.inmobiliaria_id
FROM usuarios u
WHERE u.id = r.vendedor_id
  AND u.inmobiliaria_id IS NOT NULL
  AND r.inmobiliaria_id IS NULL;

CREATE INDEX reservas_inmobiliaria_id_idx
    ON reservas (inmobiliaria_id, fecha_creacion DESC, id DESC);

-- The agency is part of the commercial identity and must remain stable for
-- visibility and audit purposes even if the user changes agency later.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION reservas_protect_immutable_fields() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NEW.id IS DISTINCT FROM OLD.id
        OR NEW.lote_id IS DISTINCT FROM OLD.lote_id
        OR NEW.cliente_id IS DISTINCT FROM OLD.cliente_id
        OR NEW.vendedor_id IS DISTINCT FROM OLD.vendedor_id
        OR NEW.usuario_alta IS DISTINCT FROM OLD.usuario_alta
        OR NEW.inmobiliaria_id IS DISTINCT FROM OLD.inmobiliaria_id
        OR NEW.fecha_vencimiento IS DISTINCT FROM OLD.fecha_vencimiento
        OR NEW.fecha_creacion IS DISTINCT FROM OLD.fecha_creacion
        OR NEW.idempotency_key IS DISTINCT FROM OLD.idempotency_key
        OR NEW.idempotency_payload_hash IS DISTINCT FROM OLD.idempotency_payload_hash THEN
        RAISE EXCEPTION 'reservation identity and expiration fields are immutable'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS reservas_inmobiliaria_id_idx;

-- Restore the function installed by 00009 before removing the column it
-- references. This keeps a rollback executable on a disposable database.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION reservas_protect_immutable_fields() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NEW.id IS DISTINCT FROM OLD.id
        OR NEW.lote_id IS DISTINCT FROM OLD.lote_id
        OR NEW.cliente_id IS DISTINCT FROM OLD.cliente_id
        OR NEW.vendedor_id IS DISTINCT FROM OLD.vendedor_id
        OR NEW.usuario_alta IS DISTINCT FROM OLD.usuario_alta
        OR NEW.fecha_vencimiento IS DISTINCT FROM OLD.fecha_vencimiento
        OR NEW.fecha_creacion IS DISTINCT FROM OLD.fecha_creacion
        OR NEW.idempotency_key IS DISTINCT FROM OLD.idempotency_key
        OR NEW.idempotency_payload_hash IS DISTINCT FROM OLD.idempotency_payload_hash THEN
        RAISE EXCEPTION 'reservation identity and expiration fields are immutable'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

ALTER TABLE reservas
    DROP COLUMN IF EXISTS inmobiliaria_id;
