-- +goose Up
-- +goose StatementBegin

create or replace function ksuid() returns text as $$
declare
	v_time timestamp with time zone := null;
	v_seconds numeric(50) := null;
	v_numeric numeric(50) := null;
	v_epoch numeric(50) = 1400000000; -- 2014-05-13T16:53:20Z
	v_base62 text := '';
	v_alphabet char array[62] := array[
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
		'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J',
		'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 
		'U', 'V', 'W', 'X', 'Y', 'Z', 
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 
		'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't',
		'u', 'v', 'w', 'x', 'y', 'z'];
	i integer := 0;
begin

	-- Get the current time
	v_time := clock_timestamp();

	-- Extract epoch seconds
	v_seconds := EXTRACT(EPOCH FROM v_time) - v_epoch;

	-- Generate a KSUID in a numeric variable
	v_numeric := v_seconds * pow(2::numeric(50), 128) -- 32 bits for seconds and 128 bits for randomness
		+ ((random()::numeric(70,20) * pow(2::numeric(70,20), 48))::numeric(50) * pow(2::numeric(50), 80)::numeric(50))
		+ ((random()::numeric(70,20) * pow(2::numeric(70,20), 40))::numeric(50) * pow(2::numeric(50), 40)::numeric(50))
		+  (random()::numeric(70,20) * pow(2::numeric(70,20), 40))::numeric(50);

	-- Encode it to base-62
	while v_numeric <> 0 loop
		v_base62 := v_base62 || v_alphabet[mod(v_numeric, 62) + 1];
		v_numeric := div(v_numeric, 62);
	end loop;
	v_base62 := reverse(v_base62);
	v_base62 := lpad(v_base62, 27, '0');

	return v_base62;
	
end $$ language plpgsql;


INSERT INTO accounts (
    id,
    created_at
) 
SELECT * FROM (
    SELECT created_at, ksuid() as temp FROM users
) AS u
WHERE 
    id = temp AND created_at = created_at
    ;


INSERT INTO accounts (
    id,
    customer_io_id,
    country_code,
    accepted_tos_at,
    referral_code,
    referred_by,
    referred_at
) 
SELECT 
    ksuid() as id,
    id as customer_io_id,
    country_code as country_code,
    agreed_tos_at as accepted_tos_at,
    referral_code as referral_code,
    referring_user_id as referred_by,
    referred_at as referred_at
FROM users
;

INSERT INTO emails (
    email_address,
    account_id,
    confirmed,
    confirmation_sent_at,
    confirmation_code
)
    SELECT 
    u.email_address, 
    a.id as account_id, 
    u.email_confirmed as confirmed, 
    u.email_confirmation_sent_at as confirmation_sent_at, 
    u.email_confirmation_key as confirmation_code
FROM users u 
LEFT JOIN 
    accounts a 
ON a.customer_io_id = u.id
WHERE u.email_address IS NOT NULL
AND u.email_confirmed IS TRUE;




-- ALTER TABLE accounts 
--     ADD CONSTRAINT complete_referral_infos
--     CHECK (
--         (referred_by IS NULL AND referred_at IS NULL) OR
--         (referred_by IS NOT NULL AND referred_at IS NOT NULL)
--     );

    -- CHECK(length(referred_by)=6),
        -- REFERENCES accounts(referral_code) ON DELETE SET NULL,


    -- referral_code TEXT UNIQUE,
    -- --  CHECK(length(referral_code)=6) CHECK (referral_code ~ '^[A-Z0-9]+$'),
    -- referred_by TEXT,
    -- -- CHECK(length(referred_by)=6),
    --     -- REFERENCES accounts(referral_code) ON DELETE SET NULL,