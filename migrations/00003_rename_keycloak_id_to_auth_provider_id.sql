-- +goose Up
ALTER TABLE usuarios RENAME COLUMN keycloak_id TO auth_provider_id;
ALTER TABLE usuarios RENAME CONSTRAINT usuarios_keycloak_id_key TO usuarios_auth_provider_id_key;

-- +goose Down
ALTER TABLE usuarios RENAME CONSTRAINT usuarios_auth_provider_id_key TO usuarios_keycloak_id_key;
ALTER TABLE usuarios RENAME COLUMN auth_provider_id TO keycloak_id;
