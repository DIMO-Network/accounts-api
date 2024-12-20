package controller

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/DIMO-Network/accounts-api/models"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

var referralCodeAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (d *Controller) GenerateReferralCode(ctx context.Context) (string, error) {
	for {
		// Generate a random 6-character code
		codeB := make([]byte, 6)
		for i := range codeB {
			codeB[i] = referralCodeAlphabet[rand.IntN(len(referralCodeAlphabet))]
		}
		code := string(codeB)

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
// @Success 200 {object} controller.StandardRes
// @Failure 400 {object} controller.ErrorRes
// @Failure 500 {object} controller.ErrorRes
// @Tags referral
// @Router /v1/account/referral/submit [post]
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
		return err
	}

	logger := d.log.With().Str("account", acct.ID).Logger()
	c.Locals("logger", &logger)

	if acct.ReferredAt.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "Already entered a referral code.")
	}

	var body SubmitReferralCodeRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse request body.")
	}

	logger.Info().Msgf("Got referral code %s.", body.Code)
	referralCode := body.Code
	if !referralCodeRegex.MatchString(referralCode) {
		return fiber.NewError(fiber.StatusBadRequest, "Referral code must consist of 6 digits and upper-case letters.")
	}

	refAcct, err := models.Accounts(
		models.AccountWhere.ReferralCode.EQ(referralCode),
		qm.Load(models.AccountRels.Wallet),
	).One(c.Context(), tx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fiber.NewError(fiber.StatusBadRequest, "No user with that referral code found.")
		}
		return err
	}

	referrer := refAcct.R.Wallet
	if referrer == nil {
		return fmt.Errorf("referring user %s has no wallet", refAcct.ID)
	}

	referree := acct.R.Wallet
	if referree == nil {
		return fmt.Errorf("referred user %s has no wallet", acct.ID)
	}

	if common.BytesToAddress(referree.Address) == common.BytesToAddress(referrer.Address) {
		return fiber.NewError(fiber.StatusBadRequest, "User and referrer have the same Ethereum address.")
	}

	// No circular referrals.
	if refAcct.ReferredBy.Valid && refAcct.ReferredBy.String == acct.ID {
		return fiber.NewError(fiber.StatusBadRequest, "Referrer was referred by this user.")
	}

	acct.ReferredBy = null.StringFrom(refAcct.ID)
	acct.ReferredAt = null.TimeFrom(time.Now())
	if _, err := acct.Update(c.Context(), tx, boil.Whitelist(models.AccountColumns.ReferredBy, models.AccountColumns.ReferredAt, models.AccountColumns.UpdatedAt)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.JSON(StandardRes{
		Message: "Referral code successfully submitted.",
	})
}
