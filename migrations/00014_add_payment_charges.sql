-- +goose Up
-- A cobro may carry charges beyond the cuota (municipal or provincial taxes,
-- administrative fees, honorarios, services), and each one may be in its own
-- currency: a cuota in USD collected together with services in ARS. They hang
-- from the cobro that collected them, so cuota_id is no longer required.
ALTER TABLE cargos_adicionales
    ADD COLUMN cobro_id UUID REFERENCES cobros (id),
    ALTER COLUMN cuota_id DROP NOT NULL;

ALTER TABLE cargos_adicionales
    DROP CONSTRAINT IF EXISTS cargos_adicionales_tipo_check;

ALTER TABLE cargos_adicionales
    ADD CONSTRAINT cargos_adicionales_tipo_chk CHECK (
        tipo IN (
            'impuesto_municipal',
            'impuesto_provincial',
            'gasto_administrativo',
            'honorarios',
            'servicios',
            'cargo_inmobiliaria',
            'otros'
        )
    ),
    ADD CONSTRAINT cargos_adicionales_monto_chk CHECK (monto > 0),
    ADD CONSTRAINT cargos_adicionales_moneda_chk CHECK (
        char_length(btrim(moneda)) BETWEEN 1 AND 10 AND moneda = btrim(moneda)
    ),
    ADD CONSTRAINT cargos_adicionales_origen_chk CHECK (
        cobro_id IS NOT NULL OR cuota_id IS NOT NULL
    );

CREATE INDEX cargos_adicionales_cobro_id_idx ON cargos_adicionales (cobro_id);

-- +goose Down
DROP INDEX IF EXISTS cargos_adicionales_cobro_id_idx;

DELETE FROM cargos_adicionales WHERE cuota_id IS NULL;

ALTER TABLE cargos_adicionales
    DROP CONSTRAINT IF EXISTS cargos_adicionales_origen_chk,
    DROP CONSTRAINT IF EXISTS cargos_adicionales_moneda_chk,
    DROP CONSTRAINT IF EXISTS cargos_adicionales_monto_chk,
    DROP CONSTRAINT IF EXISTS cargos_adicionales_tipo_chk;

ALTER TABLE cargos_adicionales
    ADD CONSTRAINT cargos_adicionales_tipo_check CHECK (
        tipo IN ('impuesto_municipal', 'impuesto_provincial', 'cargo_inmobiliaria', 'otros')
    );

ALTER TABLE cargos_adicionales
    ALTER COLUMN cuota_id SET NOT NULL,
    DROP COLUMN IF EXISTS cobro_id;
