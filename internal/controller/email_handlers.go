package controller

import (
	"math/rand"

	"github.com/gofiber/fiber/v2"
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
	// userAccount, err := getUserAccountClaims(c)
	// if err != nil {
	// 	d.log.Err(err).Msg("failed to parse user")
	// 	return err
	// }

	// var body RequestEmailValidation
	// if err := c.BodyParser(&body); err != nil {
	// 	return err
	// }

	// tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
	// if err != nil {
	// 	return err
	// }
	// defer tx.Rollback() //nolint

	// acct, err := d.getUserAccount(c.Context(), userAccount, tx)
	// if err != nil {
	// 	if !errors.Is(err, sql.ErrNoRows) {
	// 		return err
	// 	}
	// }

	// if acct.R.Wallet == nil {
	// 	return fmt.Errorf("email-first accounts must associate wallet before updating email")
	// }

	// if acct.R.Email != nil && acct.R.Email.Confirmed {
	// 	return fmt.Errorf("email address already linked with account")
	// }

	// if emlAssociated, err := models.Emails(models.EmailWhere.EmailAddress.EQ(body.EmailAddress)).One(c.Context(), tx); err != nil {
	// 	if !errors.Is(err, sql.ErrNoRows) {
	// 		return err
	// 	}
	// } else if emlAssociated != nil && emlAssociated.Confirmed {
	// 	// TODO AE: note that this does imply someone can link a non-confirmed email to their account
	// 	// for example, by not completing this step
	// 	return fmt.Errorf("email address linked to another account")
	// }

	// confKey := generateConfirmationKey()
	// userEmail := &models.Email{
	// 	AccountID:          acct.ID,
	// 	EmailAddress:       body.EmailAddress,
	// 	Confirmed:          false,
	// 	ConfirmationCode:   null.StringFrom(confKey),
	// 	ConfirmationSentAt: null.TimeFrom(time.Now()),
	// }

	// if err := userEmail.Insert(c.Context(), tx, boil.Infer()); err != nil {
	// 	return err
	// }

	// if err := d.emailService.SendConfirmationEmail(c.Context(), d.emailTemplate, body.EmailAddress, confKey); err != nil {
	// 	return err
	// }

	// if err := tx.Commit(); err != nil {
	// 	return err
	// }

	// return c.SendStatus(fiber.StatusNoContent)
	return nil
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
	// userAccount, err := getUserAccountClaims(c)
	// if err != nil {
	// 	d.log.Err(err).Msg("failed to parse user")
	// 	return err
	// }

	// tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
	// if err != nil {
	// 	return err
	// }
	// defer tx.Rollback() //nolint

	// acct, err := d.getUserAccount(c.Context(), userAccount, tx)
	// if err != nil {
	// 	if !errors.Is(err, sql.ErrNoRows) {
	// 		return err
	// 	}
	// }

	// if acct.R.Email == nil {
	// 	return fmt.Errorf("no email address associated with user account")
	// }

	// // can we be linking muttiple email addrs to the same account?
	// if acct.R.Email.Confirmed {
	// 	return fmt.Errorf("email already confirmed")
	// }

	// if !acct.R.Email.ConfirmationSentAt.Valid || !acct.R.Email.ConfirmationCode.Valid {
	// 	return fmt.Errorf("email confirmation never sent")
	// }

	// if time.Since(acct.R.Email.ConfirmationSentAt.Time) > d.allowedLateness {
	// 	return fmt.Errorf("email confirmation message expired")
	// }

	// confirmationBody := new(CompleteEmailValidation)
	// if err := c.BodyParser(confirmationBody); err != nil {
	// 	return err
	// }

	// if confirmationBody.Key != acct.R.Email.ConfirmationCode.String {
	// 	return fmt.Errorf("email confirmation code invalid")
	// }

	// acct.R.Email.Confirmed = true
	// acct.R.Email.ConfirmationCode = null.StringFromPtr(nil)
	// acct.R.Email.ConfirmationSentAt = null.TimeFromPtr(nil)
	// if _, err := acct.R.Email.Update(c.Context(), tx, boil.Infer()); err != nil {
	// 	return err
	// }

	// if err := d.cioService.SendCustomerIoEvent(acct.ID, &acct.R.Email.EmailAddress, nil); err != nil {
	// 	return fmt.Errorf("failed to send customer.io event while linking email with confirmation: %w", err)
	// }

	return c.SendStatus(fiber.StatusNoContent)
}

// LinkEmailToken godoc
// @Summary Link an email to existing wallet account; require a signed JWT from auth server
// @Success 204
// @Tags email
// @Failure 400 {object} controller.ErrorRes
// @Router /v1/account/link/email/token [post]
func (d *Controller) LinkEmailToken(c *fiber.Ctx) error {
	// userAccount, err := getUserAccountClaims(c)
	// if err != nil {
	// 	d.log.Err(err).Msg("failed to parse user")
	// 	return err
	// }

	// tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
	// if err != nil {
	// 	return err
	// }
	// defer tx.Rollback() //nolint

	// acct, err := d.getUserAccount(c.Context(), userAccount, tx)
	// if err != nil {
	// 	return err
	// }

	// if acct.R.Wallet == nil {
	// 	return fmt.Errorf("no wallet associated with user account")
	// }

	// if acct.R.Email != nil {
	// 	return fmt.Errorf("account already has linked email")
	// }

	// var tb TokenBody
	// if err := c.BodyParser(&tb); err != nil {
	// 	return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse request body.")
	// }

	// infos := AccountClaims{}
	// if _, err = jwt.ParseWithClaims(tb.Token, &infos, d.jwkResource.Keyfunc); err != nil {
	// 	return err
	// }

	// if infos.EmailAddress == nil || !emailPattern.MatchString(*infos.EmailAddress) {
	// 	return errors.New("failed to parse email address from token")
	// }

	// email := models.Email{
	// 	AccountID:    acct.ID,
	// 	Confirmed:    true,
	// 	EmailAddress: *infos.EmailAddress,
	// }

	// if err := email.Insert(c.Context(), tx, boil.Infer()); err != nil {
	// 	return err
	// }

	// if err := tx.Commit(); err != nil {
	// 	return err
	// }

	// if err := d.cioService.SendCustomerIoEvent(acct.ID, infos.EmailAddress, nil); err != nil {
	// 	return fmt.Errorf("failed to send customer.io event while linking email with token: %w", err)
	// }

	return c.SendStatus(fiber.StatusNoContent)
}

func generateConfirmationKey() string {
	o := make([]rune, 6)
	for i := range o {
		o[i] = digits[rand.Intn(10)]
	}
	return string(o)
}
