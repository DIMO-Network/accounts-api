package controller

import (
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/DIMO-Network/accounts-api/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

// LinkWalletToken godoc
// @Summary Link a wallet to existing email account; require a signed JWT from auth server
// @Success 204
// @Tag wallet
// @Failure 400 {object} controller.ErrorRes
// @Router /v1/account/link/wallet/token [post]
func (d *Controller) LinkWalletToken(c *fiber.Ctx) error {
	userAccount, err := getUserAccountClaims(c)
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
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Account already has a linked wallet, %s.", acct.R.Wallet.Address))
	}

	var tb TokenBody
	if err := c.BodyParser(&tb); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "failed to parse request body.")
	}

	var infos AccountClaims
	if _, err = jwt.ParseWithClaims(tb.Token, &infos, d.jwkResource.Keyfunc); err != nil {
		return err
	}

	if infos.EthereumAddress == nil {
		return fiber.NewError(fiber.StatusBadRequest, "Token in the body has no ethereum_address claim.")
	}

	wallet := &models.Wallet{
		AccountID: acct.ID,
		Address:   infos.EthereumAddress.Bytes(),
	}

	if err := wallet.Insert(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	acct.UpdatedAt = time.Now()
	_, err = acct.Update(c.Context(), tx, boil.Whitelist(models.AccountColumns.UpdatedAt))
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := d.cioService.SendCustomerIoEvent(acct.ID, nil, infos.EthereumAddress); err != nil {
		return fmt.Errorf("failed to send customer.io event while creating user: %w", err)
	}

	userResp, err := d.formatUserAcctResponse(acct, wallet, acct.R.Email)
	if err != nil {
		return err
	}

	return c.JSON(userResp)
}
