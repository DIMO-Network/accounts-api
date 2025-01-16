package controller

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DIMO-Network/accounts-api/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// LinkEmail godoc
// @Summary Add an unconfirmed email to the account.
// @Success 204
// @Tags email
// @Param confirmEmailRequest body controller.AddEmailRequest true "Specifies the email to be linked"
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

	normalAddr := normalizeEmail(body.Address)

	if !emailPattern.MatchString(normalAddr) {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Email address %q is invalid.", normalAddr))
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

	logger := d.log.With().Str("account", acct.ID).Logger()
	c.Locals("logger", &logger)

	if acct.R.Email != nil {
		if acct.R.Email.Address == normalAddr {
			return c.JSON(StandardRes{Message: "Account already linked to this email."})
		}
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Account already has a linked email address %s.", acct.R.Email.Address))
	}

	if existingUse, err := models.FindEmail(c.Context(), tx, normalAddr); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	} else {
		logger.Warn().Msgf("Tried to link email %s in use by account %s.", normalAddr, existingUse.AccountID)
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Email address %s already linked to another account.", normalAddr))
	}

	email := models.Email{
		Address:     normalAddr,
		AccountID:   acct.ID,
		ConfirmedAt: null.TimeFromPtr(nil),
	}

	if err := email.Insert(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	if _, err := acct.Update(c.Context(), tx, boil.Whitelist(models.AccountColumns.UpdatedAt)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	logger.Info().Msgf("Added unconfirmed email %s to account.", normalAddr)

	if err := d.cioService.SetEmail(c.Context(), acct.ID, normalAddr); err != nil {
		d.log.Err(err).Str("account", acct.ID).Msg("Failed to send email to Customer.io.")
	}

	return c.JSON(StandardRes{
		Message: fmt.Sprintf("Linked unconfirmed email %s to account.", normalAddr),
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

	logger := d.log.With().Str("account", acct.ID).Logger()
	c.Locals("logger", &logger)

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

	normalEmail := normalizeEmail(*infos.EmailAddress)

	emailConflict, err := models.Emails(
		models.EmailWhere.Address.EQ(normalEmail),
		models.EmailWhere.AccountID.NEQ(acct.ID),
	).One(c.Context(), tx)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	} else {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Email %s already linked to account %s.", normalEmail, emailConflict.AccountID))
	}

	if acct.R.Email != nil {
		if acct.R.Email.Address != normalEmail {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Account already linked to email %s.", acct.R.Email.Address))
		}
		if acct.R.Email.ConfirmedAt.Valid {
			return fiber.NewError(fiber.StatusBadRequest, "Email already confirmed.")
		}
		_, err := acct.R.Email.Delete(c.Context(), tx)
		if err != nil {
			return err
		}
	}

	email := models.Email{
		Address:     normalEmail,
		AccountID:   acct.ID,
		ConfirmedAt: null.TimeFrom(time.Now()),
	}

	if err := email.Insert(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	_, err = acct.Update(c.Context(), tx, boil.Whitelist(models.AccountColumns.UpdatedAt))
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := d.cioService.SetEmail(c.Context(), acct.ID, normalEmail); err != nil {
		logger.Err(err).Str("account", acct.ID).Msg("Failed to send email to Customer.io.")
	}

	logger.Info().Msgf("Linked confirmed email %s.", normalEmail)

	return c.JSON(StandardRes{
		Message: fmt.Sprintf("Linked email %s.", normalEmail),
	})
}
