-- +goose Up
ALTER TABLE ventas
    ADD COLUMN idempotency_key TEXT,
    ADD COLUMN idempotency_payload_hash TEXT;

ALTER TABLE ventas
    ADD CONSTRAINT ventas_idempotency_key_chk CHECK (
        idempotency_key IS NULL
        OR (char_length(btrim(idempotency_key)) BETWEEN 1 AND 128
            AND idempotency_key = btrim(idempotency_key))
    ),
    ADD CONSTRAINT ventas_idempotency_payload_hash_chk CHECK (
        idempotency_payload_hash IS NULL
        OR idempotency_payload_hash ~ '^[0-9a-f]{64}$'
    );

CREATE UNIQUE INDEX ventas_usuario_alta_idempotency_key_idx
    ON ventas (usuario_alta, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS ventas_usuario_alta_idempotency_key_idx;
ALTER TABLE ventas
    DROP CONSTRAINT IF EXISTS ventas_idempotency_payload_hash_chk,
    DROP CONSTRAINT IF EXISTS ventas_idempotency_key_chk,
    DROP COLUMN IF EXISTS idempotency_payload_hash,
    DROP COLUMN IF EXISTS idempotency_key;
