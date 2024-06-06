package controller

import (
	"accounts-api/models"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

// LinkEmail godoc
// @Summary Send a confirmation email to the authenticated user
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/link/email [post]
func (d *Controller) LinkEmail(c *fiber.Ctx) error {
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
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	// TODO AE: do we want to allow multiple email addresses to be associated with an account
	if acct.R.Email != nil && acct.R.Email.Confirmed {
		return fmt.Errorf("email address already associated with account")
	}

	confKey := generateConfirmationKey()
	userEmail := &models.Email{
		AccountID:        acct.ID,
		DexID:            userAccount.DexID,
		EmailAddress:     userAccount.EmailAddress,
		Confirmed:        false,
		Code:             null.StringFrom(confKey),
		ConfirmationSent: null.TimeFrom(time.Now()),
	}

	if err := d.emailService.SendConfirmationEmail(c.Context(), d.emailTemplate, userAccount.EmailAddress, confKey); err != nil {
		return err
	}

	if _, err := userEmail.Update(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ConfirmEmail godoc
// @Summary Submit an email confirmation key
// @Accept json
// @Param confirmEmailRequest body controllers.ConfirmEmailRequest true "Specifies the key from the email"
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Router /v1/user/confirm-email [post]
func (d *Controller) ConfirmEmail(c *fiber.Ctx) error {
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
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	if acct.R.Email == nil {
		return fmt.Errorf("no email address associated with user account")
	}

	// can we be linking multtiple email addrs to the same account?
	if acct.R.Email.Confirmed {
		return fmt.Errorf("email already confirmed")
	}

	if !acct.R.Email.ConfirmationSent.Valid || !acct.R.Email.Code.Valid {
		return fmt.Errorf("email confirmation never sent")
	}

	if time.Since(acct.R.Email.ConfirmationSent.Time) > d.allowedLateness {
		return fmt.Errorf("email confirmation message expired")
	}

	confirmationBody := new(ConfirmEmailRequest)
	if err := c.BodyParser(confirmationBody); err != nil {
		return err
	}

	if confirmationBody.Key != acct.R.Email.Code.String {
		return fmt.Errorf("email confirmation code invalid")
	}

	acct.R.Email.Confirmed = true
	acct.R.Email.Code = null.StringFromPtr(nil)
	acct.R.Email.ConfirmationSent = null.TimeFromPtr(nil)
	if _, err := acct.R.Email.Update(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// LinkEmailToken godoc
// @Summary Link an email to existing wallet account; require a signed JWT from auth server
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Router /v1/link/email/token [post]
func (d *Controller) LinkEmailToken(c *fiber.Ctx) error {
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

	if acct.R.Wallet == nil {
		return fmt.Errorf("no wallet associated with user account")
	}

	// TODO AE: unless we want to allow more than one email to be associated with an account...?
	if acct.R.Email != nil {
		return fmt.Errorf("account already has linked email")
	}

	var tb TokenBody
	if err := c.BodyParser(&tb); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse request body.")
	}

	tbClaims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tb.Token, &tbClaims, d.jwkResource.Keyfunc)
	if err != nil {
		return err
	}

	infos := getUserAccountInfos(tbClaims)
	email := models.Email{
		AccountID:    acct.ID,
		DexID:        acct.DexID,
		Confirmed:    true,
		EmailAddress: infos.EmailAddress,
	}

	if err := email.Insert(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func generateConfirmationKey() string {
	o := make([]rune, 6)
	for i := range o {
		o[i] = digits[rand.Intn(10)]
	}
	return string(o)
}
