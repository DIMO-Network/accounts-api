-- +goose Up
-- +goose StatementBegin
CREATE TABLE email_confirmations(
    address text CONSTRAINT email_confirmations_pkey PRIMARY KEY,
    account_id text NOT NULL CONSTRAINT email_confirmations_account_id_key UNIQUE CONSTRAINT email_confirmations_account_id_fkey REFERENCES accounts (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    code text NOT NULL CONSTRAINT email_confirmations_code_check CHECK (code ~ '^[0-9]{6}$')
)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE email_confirmations;
-- +goose StatementEnd
