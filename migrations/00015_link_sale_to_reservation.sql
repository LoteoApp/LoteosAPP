-- +goose Up
-- A venta converted from a reserva keeps a link to it. Existing and ordinary
-- ventas have no reserva, so the column stays nullable.
ALTER TABLE ventas ADD COLUMN reserva_id UUID;

-- The composite key keeps the venta on the same lote as its reserva.
ALTER TABLE ventas
    ADD CONSTRAINT ventas_reserva_lote_fk
        FOREIGN KEY (reserva_id, lote_id) REFERENCES reservas (id, lote_id);

-- One venta per reserva for its whole lifetime, even after the venta is
-- cancelada. The index also serves the reserva -> venta lookup.
CREATE UNIQUE INDEX ventas_reserva_id_idx
    ON ventas (reserva_id)
    WHERE reserva_id IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION ventas_protect_reserva_id() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NEW.reserva_id IS DISTINCT FROM OLD.reserva_id THEN
        RAISE EXCEPTION 'ventas.reserva_id is immutable'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER ventas_protect_reserva_id
    BEFORE UPDATE ON ventas
    FOR EACH ROW
    EXECUTE FUNCTION ventas_protect_reserva_id();

-- +goose Down
-- Dropping the column loses which venta came from which reserva; the
-- convertida state of the reserva and the lote history are kept.
DROP TRIGGER IF EXISTS ventas_protect_reserva_id ON ventas;
DROP FUNCTION IF EXISTS ventas_protect_reserva_id();
DROP INDEX IF EXISTS ventas_reserva_id_idx;
ALTER TABLE ventas
    DROP CONSTRAINT IF EXISTS ventas_reserva_lote_fk,
    DROP COLUMN IF EXISTS reserva_id;
