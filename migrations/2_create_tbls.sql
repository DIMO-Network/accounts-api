-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
SET search_path TO accounts_api, public;

CREATE TABLE accounts(
    id CHAR(27),
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    country_code CHAR(3),
    customer_io_id TEXT,
    agreed_tos_at timestamptz DEFAULT NOW(),
    referral_code CHAR(12),
    referred_by CHAR(12),
    referred_at timestamptz,

    PRIMARY KEY (id)
);

CREATE TABLE emails(
    email_address TEXT PRIMARY KEY,
    account_id TEXT UNIQUE NOT NULL
        CONSTRAINT emails_account_id_fkey REFERENCES accounts(id) ON DELETE CASCADE,
    confirmed BOOLEAN NOT NULL,
    confirmation_sent timestamptz,
    code TEXT
);

CREATE TYPE wallet_provider AS ENUM(
    'In-App',
    'Other'
    );

CREATE TABLE wallets(
    ethereum_address BYTEA PRIMARY KEY,
    account_id TEXT UNIQUE NOT NULL
        CONSTRAINT wallets_account_id_fkey REFERENCES accounts(id) ON DELETE CASCADE,
        CONSTRAINT wallets_ethereum_address_check CHECK (length(ethereum_address) = 20),
    confirmed BOOLEAN NOT NULL,
    "provider" wallet_provider DEFAULT 'Other',
    confirmation_sent timestamptz,
    challenge TEXT
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

SET search_path TO accounts_api, public;

DROP TABLE wallets;
DROP TABLE emails;
DROP TABLE accounts;

DROP TYPE wallet_provider;
-- +goose StatementEnd
