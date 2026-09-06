-- Removes a loteo created by scripts/smoke-loteos.sh, in reverse dependency
-- order: the plan's tables reference the loteo, and the loteo references its
-- own DXF entity.
--
--   psql "$DATABASE_URL" -f scripts/smoke-loteos-cleanup.sql -v loteo_id="'<uuid>'"

BEGIN;

DELETE FROM usuario_loteos WHERE loteo_id = :loteo_id::uuid;
ALTER TABLE lote_estados DISABLE TRIGGER lote_estados_reject_mutation;
DELETE FROM lote_estados
WHERE lote_id IN (SELECT id FROM lotes WHERE loteo_id = :loteo_id::uuid);
ALTER TABLE lote_estados ENABLE TRIGGER lote_estados_reject_mutation;
DELETE FROM lotes WHERE loteo_id = :loteo_id::uuid;
DELETE FROM manzana_calles WHERE loteo_id = :loteo_id::uuid;
DELETE FROM calles WHERE loteo_id = :loteo_id::uuid;
DELETE FROM manzanas WHERE loteo_id = :loteo_id::uuid;
UPDATE loteos SET dxf_entidad_id = NULL WHERE id = :loteo_id::uuid;
DELETE FROM dxf_entidades WHERE loteo_id = :loteo_id::uuid;
DELETE FROM loteos WHERE id = :loteo_id::uuid;

COMMIT;
