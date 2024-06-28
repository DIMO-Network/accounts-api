-- +goose Up
-- +goose StatementBegin

CREATE TABLE users (
    id text PRIMARY KEY,
    email_address text,
    email_confirmed boolean NOT NULL,
    email_confirmation_sent_at timestamp with time zone,
    email_confirmation_key text,
    created_at timestamptz zone NOT NULL,
    country_code character(3),
    ethereum_address bytea,
    agreed_tos_at timestamp with time zone,
    auth_provider_id text NOT NULL,
    ethereum_challenge text,
    ethereum_challenge_sent timestamp with time zone,
    ethereum_confirmed boolean NOT NULL,
    in_app_wallet boolean DEFAULT false NOT NULL,
    referral_code TEXT UNIQUE CHECK (length(referral_code)=6) CHECK (referral_code ~ '^[A-Z0-9]+$'),
    referred_at timestamp with time zone,
    referring_user_id text
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE users;

-- +goose StatementEnd
