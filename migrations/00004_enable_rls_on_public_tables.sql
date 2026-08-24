-- +goose Up
-- Tables in the public schema are reachable through Supabase's Data API
-- (PostgREST) with the anon key that ships in the frontend bundle. Every
-- read and write goes through the Go backend, which connects as the table
-- owner and therefore bypasses RLS, so no policy is defined: enabling RLS
-- without policies denies the API roles while leaving the backend working.
-- FORCE ROW LEVEL SECURITY must not be added; it would also block the owner.
ALTER TABLE usuarios ENABLE ROW LEVEL SECURITY;
ALTER TABLE goose_db_version ENABLE ROW LEVEL SECURITY;

-- The anon and authenticated roles only exist on Supabase; the guard keeps
-- this migration runnable against a plain PostgreSQL instance.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'anon') THEN
        REVOKE ALL ON TABLE usuarios FROM anon;
        REVOKE ALL ON TABLE goose_db_version FROM anon;
    END IF;

    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'authenticated') THEN
        REVOKE ALL ON TABLE usuarios FROM authenticated;
        REVOKE ALL ON TABLE goose_db_version FROM authenticated;
    END IF;
END
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'anon') THEN
        GRANT ALL ON TABLE usuarios TO anon;
        GRANT ALL ON TABLE goose_db_version TO anon;
    END IF;

    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'authenticated') THEN
        GRANT ALL ON TABLE usuarios TO authenticated;
        GRANT ALL ON TABLE goose_db_version TO authenticated;
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE goose_db_version DISABLE ROW LEVEL SECURITY;
ALTER TABLE usuarios DISABLE ROW LEVEL SECURITY;
