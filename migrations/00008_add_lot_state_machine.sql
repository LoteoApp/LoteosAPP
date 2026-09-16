-- +goose Up
ALTER TABLE lotes
    ADD COLUMN estado_actual TEXT NOT NULL DEFAULT 'disponible'
        CHECK (estado_actual IN ('disponible', 'reservado', 'vendido', 'finalizado'));

ALTER TABLE lote_estados
    ADD COLUMN origen TEXT,
    ADD COLUMN razon TEXT,
    ADD COLUMN reserva_id UUID,
    ADD COLUMN venta_id UUID;

ALTER TABLE reservas
    ADD CONSTRAINT reservas_id_lote_id_uniq UNIQUE (id, lote_id);

ALTER TABLE ventas
    ADD CONSTRAINT ventas_id_lote_id_uniq UNIQUE (id, lote_id);

ALTER TABLE lote_estados
    ADD CONSTRAINT lote_estados_reserva_lote_fk
        FOREIGN KEY (reserva_id, lote_id) REFERENCES reservas (id, lote_id),
    ADD CONSTRAINT lote_estados_venta_lote_fk
        FOREIGN KEY (venta_id, lote_id) REFERENCES ventas (id, lote_id);

UPDATE lote_estados
SET origen = 'sistema',
    razon = 'Evento existente antes de habilitar la máquina de estados';

INSERT INTO lote_estados (lote_id, estado, origen, usuario_modificacion)
SELECT l.id, 'disponible', 'alta', l.usuario_modificacion
FROM lotes l
WHERE NOT EXISTS (
    SELECT 1 FROM lote_estados le WHERE le.lote_id = l.id
);

ALTER TABLE lote_estados
    ALTER COLUMN origen SET NOT NULL,
    ADD CONSTRAINT lote_estados_origen_chk CHECK (
        origen IN ('alta', 'reserva', 'venta', 'cobranza', 'sistema', 'correccion')
    );

UPDATE lotes l
SET estado_actual = latest.estado,
    fecha_modificacion = GREATEST(l.fecha_modificacion, latest.fecha_creacion)
FROM (
    SELECT DISTINCT ON (le.lote_id) le.lote_id, le.estado, le.fecha_creacion
    FROM lote_estados le
    ORDER BY le.lote_id, le.fecha_creacion DESC, le.id DESC
) latest
WHERE latest.lote_id = l.id;

DROP INDEX lote_estados_lote_id_idx;
CREATE INDEX lote_estados_lote_id_fecha_creacion_id_idx
    ON lote_estados (lote_id, fecha_creacion DESC, id DESC);
CREATE INDEX lote_estados_reserva_id_idx ON lote_estados (reserva_id)
    WHERE reserva_id IS NOT NULL;
CREATE INDEX lote_estados_venta_id_idx ON lote_estados (venta_id)
    WHERE venta_id IS NOT NULL;

ALTER TABLE lote_estados
    ADD CONSTRAINT lote_estados_razon_requerida_chk CHECK (
        origen <> 'correccion'
        AND NOT (estado = 'disponible' AND origen IN ('reserva', 'venta'))
        OR NULLIF(btrim(razon), '') IS NOT NULL
    ),
    ADD CONSTRAINT lote_estados_referencia_origen_chk CHECK (
        (origen <> 'reserva' OR reserva_id IS NOT NULL)
        AND (origen NOT IN ('venta', 'cobranza') OR venta_id IS NOT NULL)
    );

-- +goose StatementBegin
CREATE FUNCTION lotes_protect_estado_actual() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.estado_actual IS DISTINCT FROM 'disponible' THEN
            RAISE EXCEPTION 'lotes.estado_actual must start as disponible'
                USING ERRCODE = 'integrity_constraint_violation';
        END IF;
        RETURN NEW;
    END IF;
    IF NEW.estado_actual IS DISTINCT FROM OLD.estado_actual
        AND pg_trigger_depth() = 1 THEN
        RAISE EXCEPTION 'lotes.estado_actual can only change by inserting into lote_estados'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER lotes_protect_estado_actual
    BEFORE INSERT OR UPDATE ON lotes
    FOR EACH ROW
    EXECUTE FUNCTION lotes_protect_estado_actual();

-- +goose StatementBegin
CREATE FUNCTION lote_estados_validate_transition() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE
    current_state TEXT;
    has_history BOOLEAN;
BEGIN
    SELECT estado_actual INTO current_state
    FROM lotes
    WHERE id = NEW.lote_id AND fecha_baja IS NULL
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'active lote not found'
            USING ERRCODE = 'foreign_key_violation';
    END IF;

    NEW.fecha_creacion := clock_timestamp();

    SELECT EXISTS (
        SELECT 1 FROM lote_estados WHERE lote_id = NEW.lote_id
    ) INTO has_history;

    IF NOT has_history THEN
        IF NEW.estado IS DISTINCT FROM 'disponible' OR NEW.origen IS DISTINCT FROM 'alta' THEN
            RAISE EXCEPTION 'first lote state must be disponible'
                USING ERRCODE = 'check_violation',
                      CONSTRAINT = 'lote_estados_transition_chk';
        END IF;
        RETURN NEW;
    END IF;

    IF NOT (CASE
        WHEN current_state = 'disponible' AND NEW.estado = 'reservado'
            THEN NEW.origen = 'reserva'
        WHEN current_state = 'disponible' AND NEW.estado = 'vendido'
            THEN NEW.origen = 'venta'
        WHEN current_state = 'reservado' AND NEW.estado = 'disponible'
            THEN NEW.origen IN ('reserva', 'sistema')
        WHEN current_state = 'reservado' AND NEW.estado = 'vendido'
            THEN NEW.origen = 'venta'
        WHEN current_state = 'vendido' AND NEW.estado = 'disponible'
            THEN NEW.origen IN ('venta', 'sistema')
        WHEN current_state = 'vendido' AND NEW.estado = 'finalizado'
            THEN NEW.origen = 'cobranza'
        ELSE false
    END) THEN
        RAISE EXCEPTION 'invalid lote state transition from % to %', current_state, NEW.estado
            USING ERRCODE = 'check_violation',
                  CONSTRAINT = 'lote_estados_transition_chk';
    END IF;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER lote_estados_validate_transition
    BEFORE INSERT ON lote_estados
    FOR EACH ROW
    EXECUTE FUNCTION lote_estados_validate_transition();

-- +goose StatementBegin
CREATE FUNCTION lote_estados_apply_current() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE lotes
    SET estado_actual = NEW.estado,
        usuario_modificacion = COALESCE(NEW.usuario_modificacion, usuario_modificacion),
        fecha_modificacion = GREATEST(fecha_modificacion, NEW.fecha_creacion)
    WHERE id = NEW.lote_id;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER lote_estados_apply_current
    AFTER INSERT ON lote_estados
    FOR EACH ROW
    EXECUTE FUNCTION lote_estados_apply_current();

CREATE TRIGGER lote_estados_reject_mutation
    BEFORE UPDATE OR DELETE ON lote_estados
    FOR EACH ROW
    EXECUTE FUNCTION estado_historial_reject_mutation();

CREATE TRIGGER lote_estados_reject_truncate
    BEFORE TRUNCATE ON lote_estados
    FOR EACH STATEMENT
    EXECUTE FUNCTION estado_historial_reject_mutation();

-- +goose StatementBegin
CREATE FUNCTION lotes_seed_estado_inicial() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO lote_estados (lote_id, estado, origen, usuario_modificacion)
    VALUES (NEW.id, 'disponible', 'alta', NEW.usuario_modificacion);
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER lotes_seed_estado_inicial
    AFTER INSERT ON lotes
    FOR EACH ROW
    EXECUTE FUNCTION lotes_seed_estado_inicial();

-- +goose Down
-- State events remain compatible with the previous schema and are preserved.
DROP TRIGGER IF EXISTS lotes_seed_estado_inicial ON lotes;
DROP TRIGGER IF EXISTS lote_estados_reject_truncate ON lote_estados;
DROP TRIGGER IF EXISTS lote_estados_reject_mutation ON lote_estados;
DROP TRIGGER IF EXISTS lote_estados_apply_current ON lote_estados;
DROP TRIGGER IF EXISTS lote_estados_validate_transition ON lote_estados;
DROP TRIGGER IF EXISTS lotes_protect_estado_actual ON lotes;
DROP FUNCTION IF EXISTS lotes_seed_estado_inicial();
DROP FUNCTION IF EXISTS lote_estados_apply_current();
DROP FUNCTION IF EXISTS lote_estados_validate_transition();
DROP FUNCTION IF EXISTS lotes_protect_estado_actual();

DROP INDEX IF EXISTS lote_estados_venta_id_idx;
DROP INDEX IF EXISTS lote_estados_reserva_id_idx;
DROP INDEX IF EXISTS lote_estados_lote_id_fecha_creacion_id_idx;
CREATE INDEX lote_estados_lote_id_idx ON lote_estados (lote_id);

ALTER TABLE lote_estados
    DROP CONSTRAINT IF EXISTS lote_estados_referencia_origen_chk,
    DROP CONSTRAINT IF EXISTS lote_estados_razon_requerida_chk,
    DROP COLUMN IF EXISTS venta_id,
    DROP COLUMN IF EXISTS reserva_id,
    DROP COLUMN IF EXISTS razon,
    DROP COLUMN IF EXISTS origen;

ALTER TABLE ventas DROP CONSTRAINT IF EXISTS ventas_id_lote_id_uniq;
ALTER TABLE reservas DROP CONSTRAINT IF EXISTS reservas_id_lote_id_uniq;

ALTER TABLE lotes DROP COLUMN IF EXISTS estado_actual;
