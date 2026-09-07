-- +goose Up
-- +goose StatementBegin
ALTER TABLE runner_profiles ADD COLUMN IF NOT EXISTS active_deadline_seconds BIGINT NULL;
ALTER TABLE runner_profiles ADD COLUMN IF NOT EXISTS runtime_class_name VARCHAR(128) NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE runner_profiles DROP COLUMN IF EXISTS active_deadline_seconds;
ALTER TABLE runner_profiles DROP COLUMN IF EXISTS runtime_class_name;
-- +goose StatementEnd
