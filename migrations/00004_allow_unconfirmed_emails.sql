-- +goose Up
-- +goose StatementBegin
ALTER TABLE emails ADD COLUMN confirmed_at timestamptz;
UPDATE emails SET confirmed_at = current_timestamp; -- Grandfather these in. They were all confirmed and were from testers.

DROP TABLE email_confirmations;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE email_confirmations(
    address text CONSTRAINT email_confirmations_pkey PRIMARY KEY,
    account_id text NOT NULL CONSTRAINT email_confirmations_account_id_key UNIQUE CONSTRAINT email_confirmations_account_id_fkey REFERENCES accounts (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    code text NOT NULL CONSTRAINT email_confirmations_code_check CHECK (code ~ '^[0-9]{6}$')
)

ALTER TABLE emails DROP COLUMN confirmed_at;
-- +goose StatementEnd
