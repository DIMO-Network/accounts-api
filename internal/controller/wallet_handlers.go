package controller

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"math/big"
	"time"

	"accounts-api/models"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"

	"crypto/rand"

	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// GenerateEthereumChallenge godoc
// @Summary Generate a challenge message for the user to sign.
// @Success 200 {object} controllers.ChallengeResponse
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/user/web3/challenge/generate [post]
func (d *Controller) GenerateEthereumChallenge(c *fiber.Ctx) error {
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

	userEth := acct.R.Wallet
	if userEth == nil {
		return fiber.NewError(fiber.StatusBadRequest, "No ethereum address to confirm.")
	}

	if userEth.Confirmed {
		return fiber.NewError(fiber.StatusBadRequest, "Ethereum address already confirmed.")
	}

	nonce, err := generateNonce()
	if err != nil {
		d.log.Err(err).Str("userId", userAccount.ID).Msg("Failed to generate nonce.")
		return opaqueInternalError
	}

	challenge := fmt.Sprintf("%s is asking you to please verify ownership of the address %s by signing this random string: %s", c.Hostname(), common.Bytes2Hex(userEth.EthereumAddress), nonce)

	userEth.ConfirmationSent = null.TimeFrom(time.Now())
	userEth.Challenge = null.StringFrom(challenge)

	if _, err := userEth.Update(c.Context(), d.dbs.DBS().Reader, boil.Infer()); err != nil {
		d.log.Err(err).Str("userId", userAccount.ID).Msg("Failed to update database record with new challenge.")
		return opaqueInternalError
	}

	return c.JSON(
		ChallengeResponse{
			Challenge: challenge,
			ExpiresAt: userEth.ConfirmationSent.Time.Add(d.allowedLateness),
		},
	)
}

// SubmitEthereumChallenge godoc
// @Summary Confirm ownership of an ethereum address by submitting a signature
// @Param confirmEthereumRequest body controllers.ConfirmEthereumRequest true "Signed challenge message"
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 500 {object} controllers.ErrorResponse
// @Router /v1/user/web3/challenge/submit [post]
func (d *Controller) SubmitEthereumChallenge(c *fiber.Ctx) error {
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

	userEth := acct.R.Wallet
	if userEth == nil {
		return fiber.NewError(fiber.StatusBadRequest, "No ethereum address to confirm.")
	}

	if userEth.Confirmed {
		return fiber.NewError(fiber.StatusBadRequest, "Ethereum address already confirmed.")
	}

	if !userEth.ConfirmationSent.Valid || !userEth.Challenge.Valid {
		return fiber.NewError(fiber.StatusBadRequest, "failed to find valid ethereum challenge")
	}

	if time.Since(userEth.ConfirmationSent.Time) > d.allowedLateness {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("ethereum challenge expired at %s", userEth.ConfirmationSent.Time))
	}

	submitBody := new(ConfirmEthereumRequest)

	if err := c.BodyParser(submitBody); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	addrb := common.BytesToAddress(userEth.EthereumAddress)

	signb, err := hexutil.Decode(submitBody.Signature)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not decode hex signature")
	}

	// This is the v parameter in the signature. Per the yellow paper, 27 means even and 28
	// means odd; it is our responsibility to shift it before passing it to crypto functions.
	switch signb[64] {
	case 0, 1:
		// This is not standard, but it seems to be what Ledger does.
		break
	case 27, 28:
		signb[64] -= 27
	default:
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("invalid v parameter %d", signb[64]))
	}

	pubKey, err := crypto.SigToPub(signHash(userEth.EthereumAddress), signb)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "could not recover public key from signature")
	}

	// TODO(elffjs): Why can't we just use crypto.Ecrecover?
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	// These are byte arrays, not slices, so this is okay to do.
	if recoveredAddr != addrb {
		return fiber.NewError(fiber.StatusBadRequest, "given address and recovered address do not match")
	}

	referralCode, err := d.GenerateReferralCode(c.Context())
	if err != nil {
		d.log.Error().Err(err).Msg("error occurred creating referral code for user")
		return fiber.NewError(fiber.StatusInternalServerError, "internal error")
	}

	acct.ReferralCode = null.StringFrom(referralCode)
	userEth.Confirmed = true
	userEth.ConfirmationSent = null.Time{}
	userEth.Challenge = null.String{}
	if _, err := acct.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal error")
	}

	if _, err := userEth.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}

func generateNonce() (string, error) {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alphabetSize := big.NewInt(int64(len(alphabet)))
	b := make([]byte, 30)
	for i := range b {
		c, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[c.Int64()]
	}
	return string(b), nil
}
