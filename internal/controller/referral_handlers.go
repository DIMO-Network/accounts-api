package controller

import (
	"accounts-api/models"
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func (d *Controller) GenerateReferralCode(ctx context.Context) (string, error) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	alphabet := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	for {
		// Generate a random 12-character code
		codeB := make([]byte, 12)
		for i := range codeB {
			codeB[i] = alphabet[rand.Intn(len(alphabet))]
		}
		code := string(codeB)

		if exists, err := models.Accounts(
			models.AccountWhere.ReferralCode.EQ(null.StringFrom(code)),
		).Exists(ctx, d.dbs.DBS().Reader); err != nil {
			return "", err
		} else if !exists {
			return code, nil
		}
	}
}

// SubmitReferralCode godoc
// @Summary Takes the referral code, validates and stores it
// @Param submitReferralCodeRequest body controllers.SubmitReferralCodeRequest true "ReferralCode is the 6-digit, alphanumeric referral code from another user."
// @Success 200 {object} controllers.SubmitReferralCodeResponse
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/user/submit-referral-code [post]
func (d *Controller) SubmitReferralCode(c *fiber.Ctx) error {
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
		return fmt.Errorf("failed to get user account to submit referral code: %w", err)
	}

	if acct.ReferredBy.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "cannot accept more than one referral code per user")
	}

	if acct.ReferredAt.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "already entered a referral code.")
	}

	if acct.R.Wallet != nil {
		if devicesPaired, err := d.identityService.VehiclesOwned(c.Context(), common.BytesToAddress(acct.R.Wallet.EthereumAddress)); err != nil {
			return err
		} else if devicesPaired {
			return fiber.NewError(fiber.StatusBadRequest, "Can't enter a referral code after adding vehicles.")
		}
	}

	var body SubmitReferralCodeRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Couldn't parse request body.")
	}

	d.log.Info().Str("userId", acct.ID).Msgf("Got referral code %q.", body.ReferralCode)
	referralCode := strings.ToUpper(strings.TrimSpace(body.ReferralCode))
	if !referralCodeRegex.MatchString(referralCode) {
		return fiber.NewError(fiber.StatusBadRequest, "Referral code must be 6 characters and consist of digits and upper-case letters.")
	}

	refAcct, err := models.Accounts(
		models.AccountWhere.ReferralCode.EQ(null.StringFrom(referralCode)),
		qm.Load(models.AccountRels.Wallet),
	).One(c.Context(), tx)
	if err != nil {
		if err == sql.ErrNoRows {
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

	if common.BytesToAddress(referree.EthereumAddress) == common.BytesToAddress(referrer.EthereumAddress) {
		return fiber.NewError(fiber.StatusBadRequest, "User and referrer have the same Ethereum address.")
	}

	// No circular referrals.
	if refAcct.ReferredBy.Valid && refAcct.ReferredBy.String == acct.ReferralCode.String {
		return fiber.NewError(fiber.StatusBadRequest, "Referrer was referred by this user.")
	}

	acct.ReferredBy = null.StringFrom(refAcct.ReferralCode.String)
	acct.ReferredAt = null.TimeFrom(time.Now())
	if _, err := acct.Update(c.Context(), tx, boil.Whitelist(models.AccountColumns.ReferredBy, models.AccountColumns.ReferredAt)); err != nil {
		return err
	}

	return c.JSON(SubmitReferralCodeResponse{
		Message: "Referral code used.",
	})
}
