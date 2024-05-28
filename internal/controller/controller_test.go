package controller

import (
	"accounts-api/internal/config"
	"accounts-api/internal/test"
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
)

const migrationsDirRelPath = "../../migrations"

func Test_GetAccount(t *testing.T) {
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

	wallet, err := cont.createUser(context.Background(), account, tx)
	assert.NoError(t, err)

	assert.Equal(t, wallet.EthereumAddress, account.EthereumAddress.Bytes())

}
