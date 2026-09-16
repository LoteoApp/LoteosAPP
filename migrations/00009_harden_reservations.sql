-- +goose Up
ALTER TABLE reservas
    ADD COLUMN idempotency_key TEXT,
    ADD COLUMN idempotency_payload_hash TEXT;

ALTER TABLE reservas
    ADD CONSTRAINT reservas_idempotency_key_chk CHECK (
        idempotency_key IS NULL
        OR (char_length(btrim(idempotency_key)) BETWEEN 1 AND 128
            AND idempotency_key = btrim(idempotency_key))
    ),
    ADD CONSTRAINT reservas_idempotency_payload_hash_chk CHECK (
        idempotency_payload_hash IS NULL
        OR idempotency_payload_hash ~ '^[0-9a-f]{64}$'
    );

CREATE UNIQUE INDEX reservas_usuario_alta_idempotency_key_idx
    ON reservas (usuario_alta, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

DROP INDEX IF EXISTS reservas_fecha_vencimiento_idx;
CREATE INDEX reservas_active_expiration_idx
    ON reservas (fecha_vencimiento, id)
    WHERE estado_actual = 'activa';

-- A reservation can only be seeded as active and can only move once from
-- active to one of its terminal states. The application still supplies the
-- business reason; this trigger protects the append-only state machine for
-- direct database writers as well.
-- +goose StatementBegin
CREATE FUNCTION reserva_estados_validate_transition() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    current_state TEXT;
    has_history BOOLEAN;
BEGIN
    IF NEW.estado = 'cancelada'
        AND char_length(btrim(COALESCE(NEW.razon, ''))) = 0 THEN
        RAISE EXCEPTION 'cancelled reservation state requires a reason'
            USING ERRCODE = 'check_violation',
                  CONSTRAINT = 'reserva_estados_reason_chk';
    END IF;

    SELECT estado_actual INTO current_state
    FROM reservas
    WHERE id = NEW.reserva_id
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'reservation not found'
            USING ERRCODE = 'foreign_key_violation';
    END IF;

    SELECT EXISTS (
        SELECT 1 FROM reserva_estados WHERE reserva_id = NEW.reserva_id
    ) INTO has_history;

    IF NOT has_history THEN
        IF NEW.estado IS DISTINCT FROM 'activa' THEN
            RAISE EXCEPTION 'first reservation state must be activa'
                USING ERRCODE = 'check_violation',
                      CONSTRAINT = 'reserva_estados_transition_chk';
        END IF;
        RETURN NEW;
    END IF;

    IF current_state <> 'activa'
        OR NEW.estado NOT IN ('vencida', 'cancelada', 'convertida') THEN
        RAISE EXCEPTION 'invalid reservation state transition from % to %', current_state, NEW.estado
            USING ERRCODE = 'check_violation',
                  CONSTRAINT = 'reserva_estados_transition_chk';
    END IF;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER reserva_estados_validate_transition
    BEFORE INSERT ON reserva_estados
    FOR EACH ROW
    EXECUTE FUNCTION reserva_estados_validate_transition();

-- Commercial identity and the expiration instant are immutable for the whole
-- audit lifetime. estado_actual remains changed only by the history trigger
-- already installed by migration 00005.
-- +goose StatementBegin
CREATE FUNCTION reservas_protect_immutable_fields() RETURNS trigger
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

CREATE TRIGGER reservas_protect_immutable_fields
    BEFORE UPDATE ON reservas
    FOR EACH ROW
    EXECUTE FUNCTION reservas_protect_immutable_fields();

-- +goose Down
DROP TRIGGER IF EXISTS reservas_protect_immutable_fields ON reservas;
DROP FUNCTION IF EXISTS reservas_protect_immutable_fields();
DROP TRIGGER IF EXISTS reserva_estados_validate_transition ON reserva_estados;
DROP FUNCTION IF EXISTS reserva_estados_validate_transition();
DROP INDEX IF EXISTS reservas_active_expiration_idx;
CREATE INDEX reservas_fecha_vencimiento_idx ON reservas (fecha_vencimiento);
DROP INDEX IF EXISTS reservas_usuario_alta_idempotency_key_idx;
ALTER TABLE reservas
    DROP CONSTRAINT IF EXISTS reservas_idempotency_payload_hash_chk,
    DROP CONSTRAINT IF EXISTS reservas_idempotency_key_chk,
    DROP COLUMN IF EXISTS idempotency_payload_hash,
    DROP COLUMN IF EXISTS idempotency_key;
