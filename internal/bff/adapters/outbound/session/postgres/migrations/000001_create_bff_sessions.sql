-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS bff_sessions (
    id VARCHAR(128) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    keycloak_sid VARCHAR(255),
    access_token TEXT NOT NULL DEFAULT '',
    refresh_token TEXT NOT NULL DEFAULT '',
    id_token TEXT,
    roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bff_sessions_keycloak_sid ON bff_sessions(keycloak_sid);
CREATE INDEX IF NOT EXISTS idx_bff_sessions_user_id ON bff_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_bff_sessions_expires_at ON bff_sessions(expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_bff_sessions_expires_at;
DROP INDEX IF EXISTS idx_bff_sessions_user_id;
DROP INDEX IF EXISTS idx_bff_sessions_keycloak_sid;
DROP TABLE IF EXISTS bff_sessions;
-- +goose StatementEnd
