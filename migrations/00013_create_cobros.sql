-- +goose Up
CREATE TABLE cobros (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    venta_id UUID NOT NULL REFERENCES ventas (id),
    tipo TEXT NOT NULL CHECK (tipo IN ('pago', 'cancelacion_total')),
    monto NUMERIC(14, 2) NOT NULL CHECK (monto > 0),
    moneda TEXT NOT NULL,
    medio_pago TEXT NOT NULL
        CHECK (medio_pago IN ('efectivo', 'transferencia', 'cheque', 'otro')),
    observacion TEXT,
    usuario_alta UUID NOT NULL REFERENCES usuarios (id),
    fecha_pago TIMESTAMPTZ NOT NULL,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cobros_venta_id_idx ON cobros (venta_id);
CREATE INDEX cobros_usuario_alta_idx ON cobros (usuario_alta);

ALTER TABLE cuotas
    ADD COLUMN cobro_id UUID REFERENCES cobros (id),
    ADD CONSTRAINT cuotas_cobro_pagada_chk CHECK (cobro_id IS NULL OR (estado = 'pagada' AND fecha_pago IS NOT NULL));

CREATE INDEX cuotas_cobro_id_idx ON cuotas (cobro_id);

-- planes_pago.fecha_entrega already marks when the entrega was collected;
-- the cobro that collected it is what the receipt and the statement show.
ALTER TABLE planes_pago
    ADD COLUMN cobro_entrega_id UUID REFERENCES cobros (id),
    ADD CONSTRAINT planes_pago_cobro_entrega_chk CHECK (cobro_entrega_id IS NULL OR fecha_entrega IS NOT NULL);

CREATE INDEX planes_pago_cobro_entrega_id_idx ON planes_pago (cobro_entrega_id);

ALTER TABLE cobros ENABLE ROW LEVEL SECURITY;

-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE cobros FROM anon;
    END IF;
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE cobros FROM authenticated;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE planes_pago
    DROP CONSTRAINT IF EXISTS planes_pago_cobro_entrega_chk,
    DROP COLUMN IF EXISTS cobro_entrega_id;

ALTER TABLE cuotas
    DROP CONSTRAINT IF EXISTS cuotas_cobro_pagada_chk,
    DROP COLUMN IF EXISTS cobro_id;

DROP TABLE IF EXISTS cobros;
