-- +goose Up
-- +goose StatementBegin
ALTER TABLE accounts ALTER COLUMN referral_code SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE accounts ALTER COLUMN referral_code SET NULL;
-- +goose StatementEnd
