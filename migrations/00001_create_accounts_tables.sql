-- +goose Up
-- +goose StatementBegin
CREATE TABLE accounts(
    id text CONSTRAINT accounts_id PRIMARY KEY,
    country_code text CONSTRAINT accounts_country_code_check CHECK (country_code ~ '^[A-Z]{3}$'),

    referral_code text CONSTRAINT accounts_referral_code_key UNIQUE CONSTRAINT accounts_referral_code_check CHECK (referral_code ~ '^[A-Z0-9]{6}$'),
    referred_by text CONSTRAINT accounts_referred_by_fkey REFERENCES accounts (id) ON DELETE SET NULL,
    referred_at timestamptz,

    accepted_tos_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT accounts_referred_by_referred_at_check CHECK (referred_by IS NULL OR referred_at IS NOT NULL)
);

CREATE TABLE emails(
    address text PRIMARY KEY,
    account_id text NOT NULL CONSTRAINT emails_account_id_key UNIQUE CONSTRAINT emails_account_id_fkey REFERENCES accounts (id) ON DELETE CASCADE
);

CREATE TABLE wallets(
    address bytea PRIMARY KEY CONSTRAINT wallets_address_check CHECK (length(address) = 20),
    account_id text NOT NULL CONSTRAINT wallets_account_id_key UNIQUE CONSTRAINT wallets_account_id_fkey REFERENCES accounts (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE wallets;
DROP TABLE emails;
DROP TABLE accounts;
-- +goose StatementEnd
