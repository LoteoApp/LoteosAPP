-- +goose Up
-- A contado sale is paid in full when it is registered, so the sale itself
-- may move the lote from vendido to finalizado; cobranza still does it when
-- the last cuota of a financed sale is paid.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION lote_estados_validate_transition() RETURNS trigger
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
            THEN NEW.origen IN ('cobranza', 'venta')
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

-- +goose Down
-- Lotes already finalizados by a venta keep their history; only new
-- transitions go back to requiring cobranza.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION lote_estados_validate_transition() RETURNS trigger
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
            THEN NEW.origen IN ('cobranza')
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
