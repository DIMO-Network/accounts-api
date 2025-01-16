package controller

import (
	_ "embed"
	"fmt"

	"github.com/DIMO-Network/accounts-api/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

// LinkWalletToken godoc
// @Summary Link a wallet to an existing account.
// @Param linkWalletRequest body controller.TokenBody true "JWT with an ethereum_address claim."
// @Success 200 {object} controller.StandardRes
// @Failure 400 {object} controller.ErrorRes
// @Tags wallet
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

	logger := d.log.With().Str("account", acct.ID).Logger()
	c.Locals("logger", &logger)

	if acct.R.Wallet != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Account already has a linked wallet, %s.", acct.R.Wallet.Address))
	}

	if acct.R.Email == nil {
		return fmt.Errorf("no email or wallet associated with account")
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

	_, err = acct.Update(c.Context(), tx, boil.Whitelist(models.AccountColumns.UpdatedAt))
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := d.cioService.SetWallet(c.Context(), acct.ID, *infos.EthereumAddress); err != nil {
		logger.Err(err).Msg("Failed to send wallet to Customer.io.")
	}

	return c.JSON(StandardRes{
		Message: fmt.Sprintf("Linked wallet %s.", *infos.EthereumAddress),
	})
}
