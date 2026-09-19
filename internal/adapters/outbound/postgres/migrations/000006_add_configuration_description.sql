-- +goose Up
-- +goose StatementBegin
ALTER TABLE configurations ADD COLUMN description TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE configurations DROP COLUMN IF EXISTS description;
-- +goose StatementEnd
