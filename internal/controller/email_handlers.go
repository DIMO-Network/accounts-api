package controller

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/DIMO-Network/accounts-api/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

// LinkEmail godoc
// @Summary Add an unconfirmed email to the account.
// @Success 204
// @Tags email
// @Param confirmEmailRequest body controller.RequestEmailValidation true "Specifies the email to be linked"
// @Failure 400 {object} controller.ErrorRes
// @Failure 403 {object} controller.ErrorRes
// @Failure 500 {object} controller.ErrorRes
// @Router /v1/account/link/email [post]
func (d *Controller) LinkEmail(c *fiber.Ctx) error {
	userAccount, err := getUserAccountClaims(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	var body AddEmailRequest
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	if !emailPattern.MatchString(body.Address) {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Email address %q is invalid.", body.Address))
	}

	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint

	acct, err := d.getUserAccount(c.Context(), userAccount, tx)
	if err != nil {
		return err
	}

	if acct.R.Email != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Account already has a linked email address %s.", acct.R.Email.Address))
	}

	if inUse, err := models.EmailExists(c.Context(), tx, body.Address); err != nil {
		return err
	} else if inUse {
		return fiber.NewError(fiber.StatusBadRequest, "Email address already linked to another account.")
	}

	email := models.Email{
		Address:     body.Address,
		AccountID:   acct.ID,
		ConfirmedAt: null.TimeFromPtr(nil),
	}

	if err := email.Insert(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	acct.UpdatedAt = time.Now()
	if _, err := acct.Update(c.Context(), tx, boil.Whitelist(models.AccountColumns.UpdatedAt)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	d.log.Info().Str("account", acct.ID).Msgf("Added unconfirmed email %s to account.", body.Address)

	if err := d.cioService.SendCustomerIoEvent(acct.ID, &body.Address, nil); err != nil {
		return fmt.Errorf("failed to send customer.io event while adding unconfirmed email: %w", err)
	}

	return c.JSON(StandardRes{
		Message: fmt.Sprintf("Added unconfirmed email %s to account.", body.Address),
	})
}

// LinkEmailToken godoc
// @Summary Link an email to existing wallet account; require a signed JWT from auth server
// @Param linkEmailRequest body controller.TokenBody true "Includes the email token"
// @Tags email
// @Success 200 {object} controller.StandardRes
// @Failure 400 {object} controller.ErrorRes
// @Router /v1/account/link/email/token [post]
func (d *Controller) LinkEmailToken(c *fiber.Ctx) error {
	userAccount, err := getUserAccountClaims(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint

	acct, err := d.getUserAccount(c.Context(), userAccount, tx)
	if err != nil {
		return err
	}

	if acct.R.Email != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Account already linked with email %s.", acct.R.Email.Address))
	}

	var tb TokenBody
	if err := c.BodyParser(&tb); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse request body.")
	}

	var infos AccountClaims
	if _, err = jwt.ParseWithClaims(tb.Token, &infos, d.jwkResource.Keyfunc); err != nil {
		return err
	}

	if infos.EmailAddress == nil {
		return fiber.NewError(fiber.StatusBadRequest, "Token in the body does not have an email claim.")
	}

	emailConflict, err := models.Emails(
		models.EmailWhere.Address.EQ(*infos.EmailAddress),
	).One(c.Context(), tx)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	} else {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Email %s already linked to account %s.", *infos.EmailAddress, emailConflict.AccountID))
	}

	email := models.Email{
		Address:     *infos.EmailAddress,
		AccountID:   acct.ID,
		ConfirmedAt: null.TimeFrom(time.Now()),
	}

	if err := email.Insert(c.Context(), tx, boil.Infer()); err != nil {
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

	if err := d.cioService.SendCustomerIoEvent(acct.ID, infos.EmailAddress, nil); err != nil {
		return fmt.Errorf("failed to send customer.io event while linking email with token: %w", err)
	}

	d.log.Info().Str("account", acct.ID).Msgf("Linked email %s.", *infos.EmailAddress)

	return c.JSON(StandardRes{
		Message: fmt.Sprintf("Linked email %s.", *infos.EmailAddress),
	})
}
