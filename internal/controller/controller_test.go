package controller

import (
	"accounts-api/internal/config"
	"accounts-api/internal/test"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang-jwt/jwt/v5"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const migrationsDirRelPath = "../../migrations"

func Test_CRUD(t *testing.T) {
	pdb, _ := test.StartContainerDatabase(context.Background(), t, migrationsDirRelPath)
	cont := NewAccountController(&config.Settings{}, pdb, nil, test.Logger())

	account := &Account{
		DexID:           ksuid.New().String(),
		ProviderID:      "Other",
		EthereumAddress: common.BigToAddress(big.NewInt(13)),
		EmailAddress:    "test@gmail.com",
	}

	tx, err := pdb.DBS().Writer.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() //nolint

	if err := cont.createUser(context.Background(), account, tx); err != nil {
		t.Fatal(err)
	}

	acct, err := cont.getUserAccount(context.Background(), account, tx)
	require.NoError(t, err)

	assert.Equal(t, acct.R.Wallet.EthereumAddress, account.EthereumAddress.Bytes())
	assert.Equal(t, acct.R.Email.EmailAddress, account.EmailAddress)

}

func Test_GenerateReferralCode(t *testing.T) {
	pdb, _ := test.StartContainerDatabase(context.Background(), t, migrationsDirRelPath)
	cont := NewAccountController(&config.Settings{}, pdb, nil, test.Logger())
	numUniqueCodes := 100

	unique_codes := make(map[string]interface{})
	for i := 0; i < numUniqueCodes; i++ {
		refCode, err := cont.GenerateReferralCode(context.Background())
		require.NoError(t, err)
		unique_codes[refCode] = nil
	}

	assert.Equal(t, numUniqueCodes, len(unique_codes))
}

func Test_JWTDecode(t *testing.T) {

	bodyToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVmNjRiYzQwZjMwYmIzMWY0NGQzOGM3MzhiZTc3OTEzYTM5Yzc3OTQifQ.eyJpc3MiOiJodHRwczovL2F1dGguZGV2LmRpbW8uem9uZSIsInByb3ZpZGVyX2lkIjoiZ29vZ2xlIiwic3ViIjoiQ2hVeE1ETTNOelUxTWpFNE9URXdOemswTVRJeE5EY1NCbWR2YjJkc1pRIiwiYXVkIjoiZGltby1kcml2ZXIiLCJleHAiOjE3MTgzMDQxODMsImlhdCI6MTcxNzA5NDU4Mywibm9uY2UiOiIxWEpZcWNsbTV4bUQ3NWM3WVNmSFl2aU5sdGtoaElTU1RjcUNYU0tCcHNZIiwiYXRfaGFzaCI6Im5NaHlVOEItc0twOGR5RWJNN0JCRGciLCJjX2hhc2giOiJQaDdKSERoZ1dSMXJfeGVTSDR6UmlnIiwiZW1haWwiOiJhbGx5c29uLmVuZ2xpc2hAZ21haWwuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWV9.TeI8a4cetrt52tXnvBHVqSm9cNGsKUEDqYs6HZCEQIH4RtsroZNvBc5fpIQeFKoUxzUBFH64U_geAyDgOC5zabMMBo8oeRQLj_KNwIrsrUYukHf79VCYH89J1nShMvYWuJISjw9bmnndK5GD5KKcCGXhW8qUDUqJNBTk0hI76FkBp7jx1yma3_qIcApyI7bnhxgCJhrrTZ41Y3aByZOnOXYyt-4uu7WM545Jnz9MChu27bZGA_O0RBvSObJ_M1pb7nI10bUH2DRXwo1-7BurPF-clewr4riOxv9jGFzJyVgPvpQN2vyecWWkRVqxHEB672EEQBX0M-pe-HajLYGmKw"

	p := jwt.NewParser()
	clm := jwt.MapClaims{}
	_, _, _ = p.ParseUnverified(bodyToken, &clm)

	claims := getUserAccountInfos(clm)

	assert.Equal(t, claims.DexID, "ChUxMDM3NzU1MjE4OTEwNzk0MTIxNDcSBmdvb2dsZQ")
	assert.Equal(t, claims.EmailAddress, "allyson.english@gmail.com")
	assert.Equal(t, claims.ProviderID, "google")

}
