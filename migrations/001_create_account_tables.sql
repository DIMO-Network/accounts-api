-- +goose Up
-- +goose StatementBegin
CREATE TABLE accounts (
    id text CONSTRAINT accounts_pkey PRIMARY KEY,
    country_code text,
    accepted_tos_at timestamptz,
    referral_code text NOT NULL CONSTRAINT accounts_referral_code_key UNIQUE,
    referred_by text CONSTRAINT accounts_referred_by_fkey REFERENCES accounts (id) ON DELETE SET NULL,
    referred_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT accounts_referred_by_referred_at_check CHECK (referred_by IS NULL OR referred_at IS NOT NULL)
);

CREATE TABLE emails (
    address text CONSTRAINT emails_address_pkey PRIMARY KEY,
    account_id text NOT NULL CONSTRAINT emails_account_id_key UNIQUE CONSTRAINT emails_account_id_fkey REFERENCES accounts (id) ON DELETE CASCADE,
    confirmed boolean NOT NULL DEFAULT FALSE,
    confirmation_sent_at timestamptz,
    confirmation_code text,
    CONSTRAINT emails_confirmed_confirmation_sent_at_confirmation_code_check CHECK (confirmed AND confirmation_sent_at IS NULL AND confirmation_code IS NULL OR NOT confirmed AND (confirmation_sent_at IS NULL AND confirmation_code IS NULL OR confirmation_sent_at IS NOT NULL AND confirmation_code IS NOT NULL))
);

CREATE TABLE wallets (
    address bytea CONSTRAINT wallets_pkey PRIMARY KEY CONSTRAINT wallets_address_check CHECK (length(address) = 20),
    account_id text NOT NULL CONSTRAINT wallets_account_id_key UNIQUE CONSTRAINT emails_account_id_fkey REFERENCES accounts (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE wallets;
DROP TABLE emails;
DROP TABLE accounts;
-- +goose StatementEnd
