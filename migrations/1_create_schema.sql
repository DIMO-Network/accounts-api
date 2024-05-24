-- +goose Up
-- +goose StatementBegin
REVOKE CREATE ON schema public FROM public;
CREATE SCHEMA IF NOT EXISTS accounts_api;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA accounts_api CASCADE;
GRANT CREATE, USAGE ON schema public TO public;
-- +goose StatementEnd
