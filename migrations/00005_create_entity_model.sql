-- +goose Up
CREATE SCHEMA IF NOT EXISTS extensions;

CREATE TABLE inmobiliarias (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    razon_social TEXT NOT NULL,
    cuit TEXT,
    telefono TEXT,
    email TEXT,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX inmobiliarias_usuario_modificacion_idx ON inmobiliarias (usuario_modificacion);

ALTER TABLE usuarios
    ADD COLUMN inmobiliaria_id UUID REFERENCES inmobiliarias (id),
    ADD COLUMN usuario_modificacion UUID REFERENCES usuarios (id),
    ADD COLUMN fecha_baja TIMESTAMPTZ;

CREATE INDEX usuarios_inmobiliaria_id_idx ON usuarios (inmobiliaria_id);
CREATE INDEX usuarios_usuario_modificacion_idx ON usuarios (usuario_modificacion);

CREATE TABLE loteos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre TEXT NOT NULL,
    ubicacion TEXT,
    descripcion TEXT,
    faja TEXT CHECK (faja IS NULL OR faja IN ('5', '6')),
    dxf_entidad_id UUID,
    tiene_agua BOOLEAN NOT NULL DEFAULT false,
    tiene_cloaca BOOLEAN NOT NULL DEFAULT false,
    tiene_luz BOOLEAN NOT NULL DEFAULT false,
    tiene_gas BOOLEAN NOT NULL DEFAULT false,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX loteos_usuario_modificacion_idx ON loteos (usuario_modificacion);

CREATE TABLE dxf_entidades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loteo_id UUID NOT NULL REFERENCES loteos (id),
    handle_dxf TEXT,
    capa TEXT NOT NULL,
    tipo TEXT,
    propiedades JSONB,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, loteo_id)
);

CREATE INDEX dxf_entidades_loteo_id_idx ON dxf_entidades (loteo_id);
CREATE INDEX dxf_entidades_usuario_modificacion_idx ON dxf_entidades (usuario_modificacion);

-- +goose StatementBegin
DO $$
DECLARE
    postgis_schema text;
BEGIN
    SELECT n.nspname INTO postgis_schema
    FROM pg_extension e
    JOIN pg_namespace n ON n.oid = e.extnamespace
    WHERE e.extname = 'postgis';

    IF postgis_schema IS NULL THEN
        CREATE SCHEMA IF NOT EXISTS extensions;
        EXECUTE 'CREATE EXTENSION postgis WITH SCHEMA extensions';
        postgis_schema := 'extensions';
    END IF;

    EXECUTE format(
        'ALTER TABLE dxf_entidades ADD COLUMN geom %I.geometry(Polygon)',
        postgis_schema
    );
    EXECUTE 'CREATE INDEX dxf_entidades_geom_idx ON dxf_entidades USING gist (geom)';
END
$$;
-- +goose StatementEnd

ALTER TABLE loteos
    ADD CONSTRAINT loteos_dxf_entidad_id_fkey
        FOREIGN KEY (dxf_entidad_id, id) REFERENCES dxf_entidades (id, loteo_id);

CREATE UNIQUE INDEX loteos_dxf_entidad_id_idx ON loteos (dxf_entidad_id)
    WHERE dxf_entidad_id IS NOT NULL;

CREATE TABLE manzanas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loteo_id UUID NOT NULL REFERENCES loteos (id),
    numero TEXT,
    dxf_entidad_id UUID,
    tiene_agua BOOLEAN NOT NULL DEFAULT false,
    tiene_cloaca BOOLEAN NOT NULL DEFAULT false,
    tiene_luz BOOLEAN NOT NULL DEFAULT false,
    tiene_gas BOOLEAN NOT NULL DEFAULT false,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, loteo_id),
    FOREIGN KEY (dxf_entidad_id, loteo_id) REFERENCES dxf_entidades (id, loteo_id)
);

CREATE INDEX manzanas_loteo_id_idx ON manzanas (loteo_id);
CREATE INDEX manzanas_usuario_modificacion_idx ON manzanas (usuario_modificacion);
CREATE UNIQUE INDEX manzanas_loteo_numero_idx ON manzanas (loteo_id, numero)
    WHERE numero IS NOT NULL AND fecha_baja IS NULL;
CREATE UNIQUE INDEX manzanas_dxf_entidad_id_uidx ON manzanas (dxf_entidad_id)
    WHERE dxf_entidad_id IS NOT NULL;

CREATE TABLE calles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loteo_id UUID NOT NULL REFERENCES loteos (id),
    dxf_entidad_id UUID,
    nombre TEXT,
    tipo TEXT CHECK (tipo IS NULL OR tipo IN ('asfalto', 'tierra', 'brosa', 'granito')),
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, loteo_id),
    FOREIGN KEY (dxf_entidad_id, loteo_id) REFERENCES dxf_entidades (id, loteo_id)
);

CREATE INDEX calles_loteo_id_idx ON calles (loteo_id);
CREATE INDEX calles_usuario_modificacion_idx ON calles (usuario_modificacion);
CREATE UNIQUE INDEX calles_dxf_entidad_id_uidx ON calles (dxf_entidad_id)
    WHERE dxf_entidad_id IS NOT NULL;

CREATE TABLE manzana_calles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manzana_id UUID NOT NULL,
    calle_id UUID NOT NULL,
    loteo_id UUID NOT NULL,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (manzana_id, loteo_id) REFERENCES manzanas (id, loteo_id),
    FOREIGN KEY (calle_id, loteo_id) REFERENCES calles (id, loteo_id)
);

CREATE INDEX manzana_calles_manzana_id_idx ON manzana_calles (manzana_id);
CREATE INDEX manzana_calles_calle_id_idx ON manzana_calles (calle_id);
CREATE INDEX manzana_calles_loteo_id_idx ON manzana_calles (loteo_id);
CREATE INDEX manzana_calles_usuario_modificacion_idx ON manzana_calles (usuario_modificacion);
CREATE UNIQUE INDEX manzana_calles_manzana_calle_idx ON manzana_calles (manzana_id, calle_id)
    WHERE fecha_baja IS NULL;

CREATE TABLE lotes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manzana_id UUID NOT NULL,
    loteo_id UUID NOT NULL REFERENCES loteos (id),
    numero TEXT,
    precio NUMERIC(14, 2),
    moneda TEXT,
    superficie NUMERIC(12, 4),
    caracteristicas TEXT,
    dxf_entidad_id UUID,
    tiene_agua BOOLEAN NOT NULL DEFAULT false,
    tiene_cloaca BOOLEAN NOT NULL DEFAULT false,
    tiene_luz BOOLEAN NOT NULL DEFAULT false,
    tiene_gas BOOLEAN NOT NULL DEFAULT false,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (manzana_id, loteo_id) REFERENCES manzanas (id, loteo_id),
    FOREIGN KEY (dxf_entidad_id, loteo_id) REFERENCES dxf_entidades (id, loteo_id)
);

CREATE INDEX lotes_manzana_id_idx ON lotes (manzana_id);
CREATE INDEX lotes_loteo_id_idx ON lotes (loteo_id);
CREATE INDEX lotes_usuario_modificacion_idx ON lotes (usuario_modificacion);
CREATE UNIQUE INDEX lotes_loteo_numero_idx ON lotes (loteo_id, numero)
    WHERE numero IS NOT NULL AND fecha_baja IS NULL;
CREATE UNIQUE INDEX lotes_dxf_entidad_id_uidx ON lotes (dxf_entidad_id)
    WHERE dxf_entidad_id IS NOT NULL;

CREATE TABLE usuario_loteos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL REFERENCES usuarios (id),
    loteo_id UUID NOT NULL REFERENCES loteos (id),
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX usuario_loteos_usuario_id_idx ON usuario_loteos (usuario_id);
CREATE INDEX usuario_loteos_loteo_id_idx ON usuario_loteos (loteo_id);
CREATE INDEX usuario_loteos_usuario_modificacion_idx ON usuario_loteos (usuario_modificacion);
CREATE UNIQUE INDEX usuario_loteos_usuario_loteo_idx ON usuario_loteos (usuario_id, loteo_id)
    WHERE fecha_baja IS NULL;

CREATE TABLE inmobiliaria_loteos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inmobiliaria_id UUID NOT NULL REFERENCES inmobiliarias (id),
    loteo_id UUID NOT NULL REFERENCES loteos (id),
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX inmobiliaria_loteos_inmobiliaria_id_idx ON inmobiliaria_loteos (inmobiliaria_id);
CREATE INDEX inmobiliaria_loteos_loteo_id_idx ON inmobiliaria_loteos (loteo_id);
CREATE INDEX inmobiliaria_loteos_usuario_modificacion_idx ON inmobiliaria_loteos (usuario_modificacion);
CREATE UNIQUE INDEX inmobiliaria_loteos_inmobiliaria_loteo_idx ON inmobiliaria_loteos (inmobiliaria_id, loteo_id)
    WHERE fecha_baja IS NULL;

CREATE TABLE clientes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre TEXT NOT NULL,
    apellido TEXT NOT NULL,
    dni TEXT NOT NULL,
    celular TEXT,
    email TEXT,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX clientes_usuario_modificacion_idx ON clientes (usuario_modificacion);
CREATE UNIQUE INDEX clientes_dni_idx ON clientes (dni)
    WHERE fecha_baja IS NULL;

CREATE TABLE archivos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loteo_id UUID REFERENCES loteos (id),
    lote_id UUID REFERENCES lotes (id),
    nombre TEXT NOT NULL,
    nombre_original TEXT,
    categoria TEXT NOT NULL CHECK (categoria IN ('dxf', 'foto', 'plano', 'documento_legal')),
    tipo_documento TEXT,
    url TEXT,
    storage_key TEXT,
    mime_type TEXT,
    hash_sha256 TEXT,
    estado_procesamiento TEXT NOT NULL DEFAULT 'pendiente'
        CHECK (estado_procesamiento IN ('pendiente', 'procesado', 'error')),
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha TIMESTAMPTZ,
    fecha_procesamiento TIMESTAMPTZ,
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT archivos_loteo_xor_lote_chk CHECK (
        (loteo_id IS NOT NULL AND lote_id IS NULL)
        OR (loteo_id IS NULL AND lote_id IS NOT NULL)
    )
);

CREATE INDEX archivos_loteo_id_idx ON archivos (loteo_id);
CREATE INDEX archivos_lote_id_idx ON archivos (lote_id);
CREATE INDEX archivos_usuario_modificacion_idx ON archivos (usuario_modificacion);

CREATE TABLE reservas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lote_id UUID NOT NULL REFERENCES lotes (id),
    cliente_id UUID NOT NULL REFERENCES clientes (id),
    vendedor_id UUID NOT NULL REFERENCES usuarios (id),
    usuario_alta UUID NOT NULL REFERENCES usuarios (id),
    estado_actual TEXT NOT NULL DEFAULT 'activa'
        CHECK (estado_actual IN ('activa', 'vencida', 'cancelada', 'convertida')),
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_vencimiento TIMESTAMPTZ NOT NULL,
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX reservas_lote_id_idx ON reservas (lote_id);
CREATE INDEX reservas_cliente_id_idx ON reservas (cliente_id);
CREATE INDEX reservas_vendedor_id_idx ON reservas (vendedor_id);
CREATE INDEX reservas_usuario_alta_idx ON reservas (usuario_alta);
CREATE INDEX reservas_usuario_modificacion_idx ON reservas (usuario_modificacion);
CREATE INDEX reservas_fecha_vencimiento_idx ON reservas (fecha_vencimiento);
CREATE UNIQUE INDEX reservas_lote_id_activa_idx ON reservas (lote_id)
    WHERE estado_actual = 'activa';

CREATE TABLE reserva_estados (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reserva_id UUID NOT NULL REFERENCES reservas (id),
    estado TEXT NOT NULL CHECK (estado IN ('activa', 'vencida', 'cancelada', 'convertida')),
    razon TEXT,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX reserva_estados_reserva_id_idx ON reserva_estados (reserva_id);
CREATE INDEX reserva_estados_usuario_modificacion_idx ON reserva_estados (usuario_modificacion);

-- +goose StatementBegin
CREATE FUNCTION reservas_protect_estado_actual() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.estado_actual IS DISTINCT FROM 'activa' THEN
            RAISE EXCEPTION 'reservas.estado_actual must start as activa'
                USING ERRCODE = 'integrity_constraint_violation';
        END IF;
        RETURN NEW;
    END IF;
    IF NEW.estado_actual IS DISTINCT FROM OLD.estado_actual
        AND pg_trigger_depth() = 1 THEN
        RAISE EXCEPTION 'reservas.estado_actual can only change by inserting into reserva_estados'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER reservas_protect_estado_actual
    BEFORE INSERT OR UPDATE ON reservas
    FOR EACH ROW
    EXECUTE FUNCTION reservas_protect_estado_actual();

-- +goose StatementBegin
CREATE FUNCTION reserva_estados_apply_current() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE reservas
    SET estado_actual = NEW.estado, fecha_modificacion = now()
    WHERE id = NEW.reserva_id;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER reserva_estados_apply_current
    AFTER INSERT ON reserva_estados
    FOR EACH ROW
    EXECUTE FUNCTION reserva_estados_apply_current();

-- +goose StatementBegin
CREATE FUNCTION estado_historial_reject_mutation() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    RAISE EXCEPTION '% is append-only', TG_TABLE_NAME
        USING ERRCODE = 'integrity_constraint_violation';
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER reserva_estados_reject_mutation
    BEFORE UPDATE OR DELETE ON reserva_estados
    FOR EACH ROW
    EXECUTE FUNCTION estado_historial_reject_mutation();

CREATE TRIGGER reserva_estados_reject_truncate
    BEFORE TRUNCATE ON reserva_estados
    FOR EACH STATEMENT
    EXECUTE FUNCTION estado_historial_reject_mutation();

-- +goose StatementBegin
CREATE FUNCTION reservas_seed_estado_inicial() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO reserva_estados (reserva_id, estado, usuario_modificacion)
    VALUES (NEW.id, 'activa', NEW.usuario_alta);
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER reservas_seed_estado_inicial
    AFTER INSERT ON reservas
    FOR EACH ROW
    EXECUTE FUNCTION reservas_seed_estado_inicial();

CREATE TABLE ventas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lote_id UUID NOT NULL REFERENCES lotes (id),
    cliente_id UUID NOT NULL REFERENCES clientes (id),
    modalidad_pago TEXT NOT NULL
        CHECK (modalidad_pago IN ('contado', 'financiado', 'entrega_financiada')),
    monto NUMERIC(14, 2) NOT NULL,
    moneda TEXT NOT NULL,
    vendedor_id UUID NOT NULL REFERENCES usuarios (id),
    usuario_alta UUID NOT NULL REFERENCES usuarios (id),
    estado_actual TEXT NOT NULL DEFAULT 'activa'
        CHECK (estado_actual IN ('activa', 'completada', 'cancelada')),
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ventas_lote_id_idx ON ventas (lote_id);
CREATE INDEX ventas_cliente_id_idx ON ventas (cliente_id);
CREATE INDEX ventas_vendedor_id_idx ON ventas (vendedor_id);
CREATE INDEX ventas_usuario_alta_idx ON ventas (usuario_alta);
CREATE INDEX ventas_usuario_modificacion_idx ON ventas (usuario_modificacion);
CREATE UNIQUE INDEX ventas_lote_id_activa_idx ON ventas (lote_id)
    WHERE estado_actual <> 'cancelada';

CREATE TABLE venta_estados (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    venta_id UUID NOT NULL REFERENCES ventas (id),
    estado TEXT NOT NULL CHECK (estado IN ('activa', 'completada', 'cancelada')),
    razon TEXT,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX venta_estados_venta_id_idx ON venta_estados (venta_id);
CREATE INDEX venta_estados_usuario_modificacion_idx ON venta_estados (usuario_modificacion);

-- +goose StatementBegin
CREATE FUNCTION ventas_protect_estado_actual() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.estado_actual IS DISTINCT FROM 'activa' THEN
            RAISE EXCEPTION 'ventas.estado_actual must start as activa'
                USING ERRCODE = 'integrity_constraint_violation';
        END IF;
        RETURN NEW;
    END IF;
    IF NEW.estado_actual IS DISTINCT FROM OLD.estado_actual
        AND pg_trigger_depth() = 1 THEN
        RAISE EXCEPTION 'ventas.estado_actual can only change by inserting into venta_estados'
            USING ERRCODE = 'integrity_constraint_violation';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER ventas_protect_estado_actual
    BEFORE INSERT OR UPDATE ON ventas
    FOR EACH ROW
    EXECUTE FUNCTION ventas_protect_estado_actual();

-- +goose StatementBegin
CREATE FUNCTION venta_estados_apply_current() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    UPDATE ventas
    SET estado_actual = NEW.estado, fecha_modificacion = now()
    WHERE id = NEW.venta_id;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER venta_estados_apply_current
    AFTER INSERT ON venta_estados
    FOR EACH ROW
    EXECUTE FUNCTION venta_estados_apply_current();

CREATE TRIGGER venta_estados_reject_mutation
    BEFORE UPDATE OR DELETE ON venta_estados
    FOR EACH ROW
    EXECUTE FUNCTION estado_historial_reject_mutation();

CREATE TRIGGER venta_estados_reject_truncate
    BEFORE TRUNCATE ON venta_estados
    FOR EACH STATEMENT
    EXECUTE FUNCTION estado_historial_reject_mutation();

-- +goose StatementBegin
CREATE FUNCTION ventas_seed_estado_inicial() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    INSERT INTO venta_estados (venta_id, estado, usuario_modificacion)
    VALUES (NEW.id, 'activa', NEW.usuario_alta);
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER ventas_seed_estado_inicial
    AFTER INSERT ON ventas
    FOR EACH ROW
    EXECUTE FUNCTION ventas_seed_estado_inicial();

CREATE TABLE planes_pago (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    venta_id UUID NOT NULL REFERENCES ventas (id),
    monto_entrega NUMERIC(14, 2),
    cantidad_cuotas INTEGER NOT NULL CHECK (cantidad_cuotas > 0),
    tasa_interes NUMERIC(8, 4),
    periodicidad TEXT,
    moneda TEXT NOT NULL,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_entrega TIMESTAMPTZ,
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX planes_pago_venta_id_idx ON planes_pago (venta_id);
CREATE INDEX planes_pago_usuario_modificacion_idx ON planes_pago (usuario_modificacion);
CREATE UNIQUE INDEX planes_pago_venta_id_activa_idx ON planes_pago (venta_id)
    WHERE fecha_baja IS NULL;

CREATE TABLE lote_estados (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lote_id UUID NOT NULL REFERENCES lotes (id),
    estado TEXT NOT NULL CHECK (estado IN ('disponible', 'reservado', 'vendido', 'finalizado')),
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX lote_estados_lote_id_idx ON lote_estados (lote_id);
CREATE INDEX lote_estados_usuario_modificacion_idx ON lote_estados (usuario_modificacion);

CREATE TABLE cuotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_pago_id UUID NOT NULL REFERENCES planes_pago (id),
    numero INTEGER NOT NULL CHECK (numero > 0),
    monto NUMERIC(14, 2) NOT NULL,
    estado TEXT NOT NULL CHECK (estado IN ('pendiente', 'pagada', 'vencida')),
    fecha_vencimiento TIMESTAMPTZ NOT NULL,
    fecha_pago TIMESTAMPTZ,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cuotas_plan_pago_id_idx ON cuotas (plan_pago_id);
CREATE INDEX cuotas_usuario_modificacion_idx ON cuotas (usuario_modificacion);
CREATE INDEX cuotas_fecha_vencimiento_idx ON cuotas (fecha_vencimiento);
CREATE UNIQUE INDEX cuotas_plan_numero_idx ON cuotas (plan_pago_id, numero);

CREATE TABLE cargos_adicionales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cuota_id UUID NOT NULL REFERENCES cuotas (id),
    monto NUMERIC(14, 2) NOT NULL,
    moneda TEXT NOT NULL,
    tipo TEXT NOT NULL
        CHECK (tipo IN ('impuesto_municipal', 'impuesto_provincial', 'cargo_inmobiliaria', 'otros')),
    observacion TEXT,
    usuario_modificacion UUID REFERENCES usuarios (id),
    fecha_baja TIMESTAMPTZ,
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    fecha_modificacion TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cargos_adicionales_cuota_id_idx ON cargos_adicionales (cuota_id);
CREATE INDEX cargos_adicionales_usuario_modificacion_idx ON cargos_adicionales (usuario_modificacion);

ALTER TABLE inmobiliarias ENABLE ROW LEVEL SECURITY;
ALTER TABLE loteos ENABLE ROW LEVEL SECURITY;
ALTER TABLE dxf_entidades ENABLE ROW LEVEL SECURITY;
ALTER TABLE manzanas ENABLE ROW LEVEL SECURITY;
ALTER TABLE calles ENABLE ROW LEVEL SECURITY;
ALTER TABLE manzana_calles ENABLE ROW LEVEL SECURITY;
ALTER TABLE lotes ENABLE ROW LEVEL SECURITY;
ALTER TABLE usuario_loteos ENABLE ROW LEVEL SECURITY;
ALTER TABLE inmobiliaria_loteos ENABLE ROW LEVEL SECURITY;
ALTER TABLE clientes ENABLE ROW LEVEL SECURITY;
ALTER TABLE archivos ENABLE ROW LEVEL SECURITY;
ALTER TABLE reservas ENABLE ROW LEVEL SECURITY;
ALTER TABLE reserva_estados ENABLE ROW LEVEL SECURITY;
ALTER TABLE ventas ENABLE ROW LEVEL SECURITY;
ALTER TABLE venta_estados ENABLE ROW LEVEL SECURITY;
ALTER TABLE planes_pago ENABLE ROW LEVEL SECURITY;
ALTER TABLE lote_estados ENABLE ROW LEVEL SECURITY;
ALTER TABLE cuotas ENABLE ROW LEVEL SECURITY;
ALTER TABLE cargos_adicionales ENABLE ROW LEVEL SECURITY;

-- +goose StatementBegin
DO $$
DECLARE
    table_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'inmobiliarias',
        'loteos',
        'dxf_entidades',
        'manzanas',
        'calles',
        'manzana_calles',
        'lotes',
        'usuario_loteos',
        'inmobiliaria_loteos',
        'clientes',
        'archivos',
        'reservas',
        'reserva_estados',
        'ventas',
        'venta_estados',
        'planes_pago',
        'lote_estados',
        'cuotas',
        'cargos_adicionales'
    ]
    LOOP
        IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'anon') THEN
            EXECUTE format('REVOKE ALL ON TABLE %I FROM anon', table_name);
        END IF;
        IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'authenticated') THEN
            EXECUTE format('REVOKE ALL ON TABLE %I FROM authenticated', table_name);
        END IF;
    END LOOP;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    table_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY[
        'inmobiliarias',
        'loteos',
        'dxf_entidades',
        'manzanas',
        'calles',
        'manzana_calles',
        'lotes',
        'usuario_loteos',
        'inmobiliaria_loteos',
        'clientes',
        'archivos',
        'reservas',
        'reserva_estados',
        'ventas',
        'venta_estados',
        'planes_pago',
        'lote_estados',
        'cuotas',
        'cargos_adicionales'
    ]
    LOOP
        IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'anon') THEN
            EXECUTE format('GRANT ALL ON TABLE %I TO anon', table_name);
        END IF;
        IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'authenticated') THEN
            EXECUTE format('GRANT ALL ON TABLE %I TO authenticated', table_name);
        END IF;
    END LOOP;
END
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS ventas_seed_estado_inicial ON ventas;
DROP TRIGGER IF EXISTS venta_estados_reject_truncate ON venta_estados;
DROP TRIGGER IF EXISTS venta_estados_reject_mutation ON venta_estados;
DROP TRIGGER IF EXISTS venta_estados_apply_current ON venta_estados;
DROP TRIGGER IF EXISTS ventas_protect_estado_actual ON ventas;
DROP FUNCTION IF EXISTS ventas_seed_estado_inicial();
DROP FUNCTION IF EXISTS venta_estados_apply_current();
DROP FUNCTION IF EXISTS ventas_protect_estado_actual();
DROP TRIGGER IF EXISTS reservas_seed_estado_inicial ON reservas;
DROP TRIGGER IF EXISTS reserva_estados_reject_truncate ON reserva_estados;
DROP TRIGGER IF EXISTS reserva_estados_reject_mutation ON reserva_estados;
DROP TRIGGER IF EXISTS reserva_estados_apply_current ON reserva_estados;
DROP TRIGGER IF EXISTS reservas_protect_estado_actual ON reservas;
DROP FUNCTION IF EXISTS reservas_seed_estado_inicial();
DROP FUNCTION IF EXISTS reserva_estados_apply_current();
DROP FUNCTION IF EXISTS reservas_protect_estado_actual();
DROP FUNCTION IF EXISTS estado_historial_reject_mutation();
DROP TABLE IF EXISTS cargos_adicionales;
DROP TABLE IF EXISTS cuotas;
DROP TABLE IF EXISTS planes_pago;
DROP TABLE IF EXISTS venta_estados;
DROP TABLE IF EXISTS ventas;
DROP TABLE IF EXISTS reserva_estados;
DROP TABLE IF EXISTS reservas;
DROP TABLE IF EXISTS lote_estados;
DROP TABLE IF EXISTS archivos;
DROP TABLE IF EXISTS manzana_calles;
DROP TABLE IF EXISTS lotes;
DROP TABLE IF EXISTS calles;
DROP TABLE IF EXISTS manzanas;
ALTER TABLE loteos DROP CONSTRAINT IF EXISTS loteos_dxf_entidad_id_fkey;
DROP TABLE IF EXISTS dxf_entidades;
DROP TABLE IF EXISTS usuario_loteos;
DROP TABLE IF EXISTS inmobiliaria_loteos;
DROP TABLE IF EXISTS loteos;
DROP TABLE IF EXISTS clientes;
ALTER TABLE usuarios DROP COLUMN IF EXISTS fecha_baja;
ALTER TABLE usuarios DROP COLUMN IF EXISTS usuario_modificacion;
ALTER TABLE usuarios DROP COLUMN IF EXISTS inmobiliaria_id;
DROP TABLE IF EXISTS inmobiliarias;
