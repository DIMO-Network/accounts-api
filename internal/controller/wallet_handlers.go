package controller

import (
	"accounts-api/models"
	_ "embed"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

// LinkWalletToken godoc
// @Summary Link a wallet to existing email account; require a signed JWT from auth server
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Router /v1/link/wallet/token [post]
func (d *Controller) LinkWalletToken(c *fiber.Ctx) error {
	userAccount, err := getuserAccountInfosToken(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint

	acct, err := d.getUserAccount(c.Context(), userAccount, tx)
	if err != nil {
		return err
	}

	if acct.R.Email == nil {
		return errors.New("no email address associated with user account")
	}

	if acct.R.Wallet != nil {
		return errors.New("account already has linked wallet")

	}

	var tb TokenBody
	if err := c.BodyParser(&tb); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "failed to parse request body.")
	}

	// TODO AE: this is a hack, we need to parse and verify the token
	tbClaims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tb.Token, &tbClaims, d.jwkResource.Keyfunc)
	if err != nil {
		fmt.Println(err)
		return err
	}

	infos := getUserAccountInfos(tbClaims)

	wallet := models.Wallet{
		AccountID:       acct.ID,
		EthereumAddress: infos.EthereumAddress.Bytes(),
		DexID:           infos.DexID,
		Confirmed:       true,
		Provider:        null.StringFrom("Turnkey"), // where does this come from?
	}

	if err := wallet.Insert(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
