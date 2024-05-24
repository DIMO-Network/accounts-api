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

	pb "github.com/DIMO-Network/devices-api/pkg/grpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func (d *Controller) GenerateReferralCode(ctx context.Context) (string, error) {
	alphabet := []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 12)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}

	for {
		// Generate a random 12-character code
		codeB := make([]rune, 12)
		for i := range b {
			b[i] = alphabet[rand.Intn(len(alphabet))]
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
			return err
		}
	}

	if acct.ReferredBy.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "cannot accept more than one referral code per user ")
	}

	if acct.ReferredAt.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "Already entered a referral code.")
	}

	dr, err := d.devicesClient.ListUserDevicesForUser(c.Context(), &pb.ListUserDevicesForUserRequest{UserId: acct.ID})
	if err != nil {
		return err
	}

	if len(dr.UserDevices) != 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Can't enter a referral code after adding vehicles.")
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

	if acct.ReferralCode.Valid && acct.ReferralCode.String == referralCode {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot refer self.")
	}

	referrer, err := models.Accounts(
		models.AccountWhere.ReferralCode.EQ(null.StringFrom(referralCode)),
		qm.Load(models.AccountRels.Wallet),
	).One(c.Context(), d.dbs.DBS().Reader)
	if err != nil {
		if err == sql.ErrNoRows {
			return fiber.NewError(fiber.StatusBadRequest, "No user with that referral code found.")
		}
		return err
	}

	refWallet := referrer.R.Wallet
	if refWallet == nil {
		return fmt.Errorf("referring user %s has no wallet", referrer.ID)
	}

	wallet := acct.R.Wallet
	if wallet == nil {
		return fmt.Errorf("referred user %s has no wallet", acct.ID)
	}

	if common.BytesToAddress(wallet.EthereumAddress) == common.BytesToAddress(refWallet.EthereumAddress) {
		return fiber.NewError(fiber.StatusBadRequest, "User and referrer have the same Ethereum address.")
	}

	// No circular referrals.
	if referrer.ReferredBy.Valid && referrer.ReferredBy.String == acct.ReferralCode.String {
		return fiber.NewError(fiber.StatusBadRequest, "Referrer was referred by this user.")
	}

	acct.ReferredBy = null.StringFrom(referrer.ReferralCode.String)
	acct.ReferredAt = null.TimeFrom(time.Now())
	if _, err := acct.Update(c.Context(), d.dbs.DBS().Writer, boil.Whitelist(models.AccountColumns.ReferredBy, models.AccountColumns.ReferredAt)); err != nil {
		return err
	}

	return c.JSON(SubmitReferralCodeResponse{
		Message: "Referral code used.",
	})
}
