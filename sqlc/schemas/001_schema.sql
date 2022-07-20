-- +goose Up
-- +goose StatementBegin
SET timezone = 'UTC';
CREATE SCHEMA IF NOT EXISTS budget_schema;
SET search_path TO budget_schema,public;
-- +goose StatementEnd


-- +goose Down
SET search_path TO public;
DROP SCHEMA IF EXISTS budget_schema CASCADE;