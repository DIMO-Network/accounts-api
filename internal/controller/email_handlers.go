package controller

import (
	"accounts-api/models"
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// SendConfirmationEmail godoc
// @Summary Send a confirmation email to the authenticated user
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/user/send-confirmation-email [post]
func (d *Controller) SendConfirmationEmail(c *fiber.Ctx) error {
	userAccount, err := getUserAccountInfos(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}
	logger := d.log.With().Str("userId", userAccount.ID).Logger()

	acct, err := models.Accounts(
		models.AccountWhere.ID.EQ(userAccount.ID),
		qm.Load(models.AccountRels.Email),
		qm.Load(models.AccountRels.Wallet),
	).One(c.Context(), d.dbs.DBS().Reader)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return errorResponseHandler(c, err, fiber.StatusInternalServerError)
		}
	}

	userEmail := acct.R.Email

	if userEmail == nil {
		return errorResponseHandler(c, fmt.Errorf("no email address associated with user account"), fiber.StatusBadRequest)
	}

	if userEmail.Confirmed {
		return errorResponseHandler(c, fmt.Errorf("email already confirmed"), fiber.StatusBadRequest)
	}

	if userEmail.ConfirmationSent.Valid && time.Since(userEmail.ConfirmationSent.Time) < d.allowedLateness {
		logger.Error().Msgf("Rejecting confirmation email request: sent one at %s.", userEmail.ConfirmationSent.Time)
		return errorResponseHandler(c, errors.New("email confirmation sent recently, please wait"), fiber.StatusConflict)
	}

	key := generateConfirmationKey()
	userEmail.Code = null.StringFrom(key)
	userEmail.ConfirmationSent = null.TimeFrom(time.Now())

	auth := smtp.PlainAuth("", d.Settings.EmailUsername, d.Settings.EmailPassword, d.Settings.EmailHost)
	addr := fmt.Sprintf("%s:%s", d.Settings.EmailHost, d.Settings.EmailPort)

	var partsBuffer bytes.Buffer
	w := multipart.NewWriter(&partsBuffer)
	defer w.Close() //nolint

	p, err := w.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain"}, "Content-Transfer-Encoding": {"quoted-printable"}})
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	pw := quotedprintable.NewWriter(p)
	if _, err := pw.Write([]byte("Hi,\r\n\r\nYour email verification code is: " + key + "\r\n")); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	pw.Close()

	h, err := w.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html"}, "Content-Transfer-Encoding": {"quoted-printable"}})
	if err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	hw := quotedprintable.NewWriter(h)
	if err := d.emailTemplate.Execute(hw, struct{ Key string }{key}); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}
	hw.Close()

	var buffer bytes.Buffer
	buffer.WriteString("From: DIMO <" + d.Settings.EmailFrom + ">\r\n" +
		"To: " + userEmail.EmailAddress + "\r\n" +
		"Subject: [DIMO] Verification Code\r\n" +
		"Content-Type: multipart/alternative; boundary=\"" + w.Boundary() + "\"\r\n" +
		"\r\n")
	if _, err := partsBuffer.WriteTo(&buffer); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if err := smtp.SendMail(addr, auth, d.Settings.EmailFrom, []string{userEmail.EmailAddress}, buffer.Bytes()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
	}

	if _, err := userEmail.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
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
	userAccount, err := getUserAccountInfos(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	acct, err := models.Accounts(
		models.AccountWhere.ID.EQ(userAccount.ID),
		qm.Load(models.AccountRels.Email),
		qm.Load(models.AccountRels.Wallet),
	).One(c.Context(), d.dbs.DBS().Reader)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return errorResponseHandler(c, err, fiber.StatusInternalServerError)
		}
	}

	userEmail := acct.R.Email
	if userEmail.Confirmed {
		return errorResponseHandler(c, fmt.Errorf("email already confirmed"), fiber.StatusBadRequest)
	}

	if !userEmail.ConfirmationSent.Valid || !userEmail.Code.Valid {
		return errorResponseHandler(c, fmt.Errorf("email confirmation never sent"), fiber.StatusBadRequest)
	}

	if time.Since(userEmail.ConfirmationSent.Time) > d.allowedLateness {
		return errorResponseHandler(c, fmt.Errorf("email confirmation message expired"), fiber.StatusBadRequest)
	}

	confirmationBody := new(ConfirmEmailRequest)
	if err := c.BodyParser(confirmationBody); err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}

	if confirmationBody.Key != userEmail.Code.String {
		return errorResponseHandler(c, fmt.Errorf("email confirmation code invalid"), fiber.StatusBadRequest)
	}

	userEmail.Confirmed = true
	userEmail.Code = null.StringFromPtr(nil)
	userEmail.ConfirmationSent = null.TimeFromPtr(nil)
	if _, err := userEmail.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return errorResponseHandler(c, err, fiber.StatusInternalServerError)
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
