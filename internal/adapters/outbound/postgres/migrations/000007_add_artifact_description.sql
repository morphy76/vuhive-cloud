-- +goose Up
-- +goose StatementBegin
ALTER TABLE artifacts ADD COLUMN description TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE artifacts DROP COLUMN IF EXISTS description;
-- +goose StatementEnd
