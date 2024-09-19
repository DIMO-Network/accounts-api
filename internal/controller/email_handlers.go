package controller

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/DIMO-Network/accounts-api/models"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

// LinkEmail godoc
// @Summary Send a confirmation email to the authenticated user
// @Success 204
// @Tags email
// @Param confirmEmailRequest body controller.RequestEmailValidation true "Specifies the email to be linked"
// @Failure 400 {object} controller.ErrorRes
// @Failure 403 {object} controller.ErrorRes
// @Failure 500 {object} controller.ErrorRes
// @Router /v1/account/link/email [post]
func (d *Controller) LinkEmail(c *fiber.Ctx) error {
	reqTime := time.Now()

	userAccount, err := getUserAccountClaims(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	var body RequestEmailValidation
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
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	if acct.R.Email != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Account already has a linked email address %s.", acct.R.Email.Address))
	}

	if inUse, err := models.EmailExists(c.Context(), tx, body.Address); err != nil {
		return err
	} else if inUse {
		return fiber.NewError(fiber.StatusBadRequest, "Email address already linked to another account.")
	}

	if conf, err := models.EmailConfirmations(
		models.EmailConfirmationWhere.AccountID.EQ(acct.ID),
	).One(c.Context(), tx); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	} else {
		if !reqTime.After(conf.ExpiresAt) {
			if conf.Address == body.Address {
				return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("A confirmation code expiring at %s was previously sent to this address.", conf.ExpiresAt))
			} else {
				return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("A confirmation code expiring at %s was previously sent to another address, %s.", conf.ExpiresAt, conf.Address))
			}
		}
		if _, err := conf.Delete(c.Context(), tx); err != nil {
			return err
		}
	}

	if oldConf, err := models.EmailConfirmations(
		models.EmailConfirmationWhere.Address.EQ(body.Address),
	).One(c.Context(), tx); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	} else {
		if !oldConf.ExpiresAt.After(reqTime) {
			return fiber.NewError(fiber.StatusBadRequest, "A confirmation code that has not yet expired was sent to this email address for a different account.")
		}
		if _, err := oldConf.Delete(c.Context(), tx); err != nil {
			return err
		}
	}

	code, err := generateConfirmationCode()
	if err != nil {
		return err
	}
	newConf := &models.EmailConfirmation{
		AccountID: acct.ID,
		Address:   body.Address,
		ExpiresAt: reqTime.Add(d.allowedLateness),
		Code:      code,
	}

	if err := newConf.Insert(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	if err := d.emailService.SendConfirmationEmail(c.Context(), d.emailTemplate, body.Address, code); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.JSON(StandardRes{
		Message: fmt.Sprintf("Confirmation code sent to %s.", body.Address),
	})
}

// ConfirmEmail godoc
// @Summary Submit an email confirmation key
// @Accept json
// @Param confirmEmailRequest body controller.CompleteEmailValidation true "Specifies the key from the email"
// @Success 204
// @Tags email
// @Failure 400 {object} controller.ErrorRes
// @Failure 403 {object} controller.ErrorRes
// @Router /v1/account/link/email/confirm [post]
func (d *Controller) ConfirmEmail(c *fiber.Ctx) error {
	reqTime := time.Now()

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
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	conf := acct.R.EmailConfirmation

	if conf == nil {
		return fiber.NewError(fiber.StatusBadRequest, "No email confirmation in progress for this account.")
	}

	if reqTime.After(conf.ExpiresAt) {
		return fiber.NewError(fiber.StatusBadRequest, "Confirmation code expired.")
	}

	var confirmationBody CompleteEmailValidation
	if err := c.BodyParser(confirmationBody); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse request body.")
	}

	if confirmationBody.Code != conf.Code {
		return fiber.NewError(fiber.StatusBadRequest, "Incorrect confirmation code.")
	}

	emailModel := models.Email{
		AccountID: acct.ID,
		Address:   conf.Address,
	}

	if err := emailModel.Insert(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	acct.UpdatedAt = reqTime
	if _, err := acct.Update(c.Context(), tx, boil.Whitelist(models.AccountColumns.UpdatedAt)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := d.cioService.SendCustomerIoEvent(acct.ID, &conf.Address, nil); err != nil {
		return fmt.Errorf("failed to send customer.io event while linking email with confirmation: %w", err)
	}

	return c.JSON(StandardRes{
		Message: "Email linked to account.",
	})
}

// LinkEmailToken godoc
// @Summary Link an email to existing wallet account; require a signed JWT from auth server
// @Success 200 {object} controller.StandardRes
// @Tags email
// @Failure 400 {object} controller.ErrorRes
// @Router /v1/account/link/email/token [post]
func (d *Controller) LinkEmailToken(c *fiber.Ctx) error {
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

	if acct.R.Email != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Account already linked with email %s.", acct.R.Email.Address))
	}

	var tb TokenBody
	if err := c.BodyParser(&tb); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse request body.")
	}

	infos := AccountClaims{}
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
		Address:   *infos.EmailAddress,
		AccountID: acct.ID,
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

	return c.JSON(StandardRes{
		Message: fmt.Sprintf("Linked email %s.", *infos.EmailAddress),
	})
}

func generateConfirmationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n), nil
}
