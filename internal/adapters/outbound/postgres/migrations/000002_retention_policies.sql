-- +goose Up
-- +goose StatementBegin
ALTER TABLE test_suites ADD COLUMN IF NOT EXISTS retention_policy JSONB NOT NULL DEFAULT '{}'::jsonb;
CREATE INDEX IF NOT EXISTS idx_test_runs_finished_at ON test_runs(finished_at);
CREATE INDEX IF NOT EXISTS idx_artifacts_status_created_at ON artifacts(status, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_artifacts_status_created_at;
DROP INDEX IF EXISTS idx_test_runs_finished_at;
ALTER TABLE test_suites DROP COLUMN IF EXISTS retention_policy;
-- +goose StatementEnd
