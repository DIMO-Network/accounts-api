-- +goose Up
-- +goose StatementBegin

CREATE TABLE accounts(
    id TEXT PRIMARY KEY CHECK(length(id)=27),
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    country_code TEXT CHECK(length(country_code)=3),
    customer_io_id TEXT,
    accepted_tos_at timestamptz,
    referral_code TEXT UNIQUE CHECK(length(referral_code)=6) CHECK (referral_code ~ '^[A-Z0-9]+$'),
    referred_by TEXT CHECK(length(referred_by)=6),
    referred_at timestamptz
);

ALTER TABLE accounts 
    ADD CONSTRAINT complete_referral_infos
    CHECK (
        (referred_by IS NULL AND referred_at IS NULL) OR
        (referred_by IS NOT NULL AND referred_at IS NOT NULL)
    );

ALTER TABLE accounts 
    ADD CONSTRAINT fk_referred_by FOREIGN KEY (referred_by)
    REFERENCES accounts(referral_code) ON DELETE SET NULL;

CREATE TABLE emails(
    email_address TEXT PRIMARY KEY,
    account_id TEXT UNIQUE NOT NULL
        CONSTRAINT emails_account_id_fkey REFERENCES accounts(id) ON DELETE CASCADE,
    confirmed BOOLEAN NOT NULL,
    confirmation_sent_at timestamptz,
    confirmation_code TEXT
);

ALTER TABLE emails 
    ADD CONSTRAINT complete_email_conf_infos
    CHECK (
        (confirmation_sent_at IS NULL AND confirmation_code IS NULL) OR
        (confirmation_sent_at IS NOT NULL AND confirmation_code IS NOT NULL)
    );

CREATE TABLE wallets(
    ethereum_address BYTEA PRIMARY KEY,
        CONSTRAINT wallets_ethereum_address_check CHECK (length(ethereum_address) = 20),
    account_id TEXT UNIQUE NOT NULL
        CONSTRAINT wallets_account_id_fkey REFERENCES accounts(id) ON DELETE CASCADE,
    "provider" TEXT
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE wallets;
DROP TABLE emails;
DROP TABLE accounts;
-- +goose StatementEnd
