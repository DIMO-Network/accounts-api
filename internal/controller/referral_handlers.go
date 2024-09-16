package controller

import (
	"accounts-api/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

var referralAlphabet = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
var referralAlphabetLen = len(referralAlphabet)

func (d *Controller) GenerateReferralCode(ctx context.Context) (string, error) {
	for {
		// Generate a random 6-character code
		codeBytes := make([]byte, 6)
		for i := range codeBytes {
			codeBytes[i] = referralAlphabet[rand.Intn(referralAlphabetLen)]
		}
		code := string(codeBytes)

		if exists, err := models.Accounts(models.AccountWhere.ReferralCode.EQ(code)).Exists(ctx, d.dbs.DBS().Reader); err != nil {
			return "", err
		} else if !exists {
			return code, nil
		}
	}
}

// SubmitReferralCode godoc
// @Summary Takes the referral code, validates and stores it
// @Param submitReferralCodeRequest body controller.SubmitReferralCodeRequest true "ReferralCode is the 6-digit, alphanumeric referral code from another user."
// @Success 200 {object} controller.SubmitReferralCodeResponse
// @Failure 400 {object} controller.ErrorRes
// @Failure 500 {object} controller.ErrorRes
// @Router /v1/accounts/submit-referral-code [post]
func (d *Controller) SubmitReferralCode(c *fiber.Ctx) error {
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
		return fmt.Errorf("failed to get user account to submit referral code: %w", err)
	}

	if acct.ReferredAt.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "User was already referred.")
	}

	if acct.R.Wallet == nil {
		return fiber.NewError(fiber.StatusBadRequest, "Add a wallet before submitting a referral code.")
	}

	var body SubmitReferralCodeRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse request body.")
	}

	d.log.Info().Str("userId", acct.ID).Msgf("Got referral code %s.", body.ReferralCode)
	referralCode := strings.ToUpper(strings.TrimSpace(body.ReferralCode))
	if !referralCodeRegex.MatchString(referralCode) {
		return fiber.NewError(fiber.StatusBadRequest, "Referral code must be 6 characters and consist of digits and upper-case letters.")
	}

	referrer, err := models.Accounts(
		models.AccountWhere.ReferralCode.EQ(referralCode),
		qm.Load(models.AccountRels.Wallet),
	).One(c.Context(), tx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fiber.NewError(fiber.StatusBadRequest, "No user with that referral code found.")
		}
		return err
	}

	if referrer.R.Wallet == nil {
		return fiber.NewError(fiber.StatusBadRequest, "No user with that referral code found.")
	}

	if referrer.ID == acct.ID {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot refer self.")
	}

	// No circular referrals.
	if referrer.ReferredBy.Valid && referrer.ReferredBy.String == acct.ID {
		return fiber.NewError(fiber.StatusBadRequest, "Referrer was referred by this user.")
	}

	acct.ReferredBy = null.StringFrom(referrer.ID)
	acct.ReferredAt = null.TimeFrom(time.Now())
	if _, err := acct.Update(c.Context(), tx, boil.Whitelist(models.AccountColumns.ReferredBy, models.AccountColumns.ReferredAt, models.AccountColumns.UpdatedAt)); err != nil {
		return err
	}

	return c.JSON(SubmitReferralCodeResponse{
		Message: "Referral code successfully submitted.",
	})
}
