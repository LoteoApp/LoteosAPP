-- +goose Up
WITH ranked_dxf AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY loteo_id
               ORDER BY fecha_creacion DESC, id DESC
           ) AS position
    FROM archivos
    WHERE categoria = 'dxf'
      AND loteo_id IS NOT NULL
      AND fecha_baja IS NULL
)
UPDATE archivos
SET fecha_baja = now(),
    fecha_modificacion = now()
FROM ranked_dxf
WHERE archivos.id = ranked_dxf.id
  AND ranked_dxf.position > 1;

CREATE UNIQUE INDEX archivos_loteo_active_dxf_idx
    ON archivos (loteo_id)
    WHERE categoria = 'dxf' AND fecha_baja IS NULL;

-- +goose Down
DROP INDEX IF EXISTS archivos_loteo_active_dxf_idx;
