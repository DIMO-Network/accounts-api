package controller

import (
	"accounts-api/models"
	"database/sql"
	"errors"
	"fmt"
	"time"

	pb "github.com/DIMO-Network/devices-api/pkg/grpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// GetUserAccount godoc
// @Summary Get attributes for the authenticated user.
// @Produce json
// @Success 200 {object} controllers.UserResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Security BearerAuth
// @Router /v1/user [get]
func (d *Controller) GetUserAccount(c *fiber.Ctx) error {
	acct, err := d.getOrCreateUserAccount(c)
	if err != nil {
		return err
	}

	userResp, err := d.formatUserAcctResponse(c.Context(), acct)
	if err != nil {
		return err
	}

	return c.JSON(userResp)
}

// UpdateUser godoc
// @Summary Modify attributes for the authenticated user
// @Accept json
// @Produce json
// @Param userUpdateRequest body controllers.UserUpdateRequest true "New field values"
// @Success 200 {object} controllers.UserResponse
// @Success 400 {object} controllers.ErrorResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Router /v1/user [put]
func (d *Controller) UpdateUser(c *fiber.Ctx) error {
	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint

	acct, err := getUserAccount(c, tx)
	if err != nil {
		d.log.Err(err).Msg("failed to get user account")
		return err
	}

	var body UserUpdateRequest
	if err := c.BodyParser(&body); err != nil {
		return errorResponseHandler(c, err, fiber.StatusBadRequest)
	}

	if body.CountryCode.Defined {
		if body.CountryCode.Value.Valid && !inSorted(d.countryCodes, body.CountryCode.Value.String) {
			return errorResponseHandler(c, fmt.Errorf("invalid country code"), fiber.StatusBadRequest)
		}
		acct.CountryCode = body.CountryCode.Value
	}

	if body.Email.Address.Defined {
		var email models.Email
		if acct.R.Email == nil {
			email.AccountID = acct.ID
			email.Confirmed = false
		}

		if !emailPattern.MatchString(body.Email.Address.Value.String) {
			return err
		}

		email.EmailAddress = body.Email.Address.Value.String

		if err := email.Upsert(c.Context(), d.dbs.DBS().Writer, true, []string{models.EmailColumns.EmailAddress}, boil.Infer(), boil.Infer()); err != nil {
			return err
		}
	}

	if body.Web3.Address.Defined {
		var wallet models.Wallet
		if acct.R.Wallet == nil {
			wallet.AccountID = acct.ID
			wallet.Confirmed = false
			wallet.Provider = null.StringFrom("Other")
		}

		if !body.Web3.Address.Value.Valid {
			return err
		}

		mixAddr, err := common.NewMixedcaseAddressFromString(body.Web3.Address.Value.String)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Invalid Ethereum address %s.", mixAddr))
		}

		wallet.EthereumAddress = mixAddr.Address().Bytes()
		if body.Web3.InApp {
			wallet.Provider = null.StringFrom("InApp")
		}

		if err := wallet.Upsert(c.Context(), d.dbs.DBS().Writer, true, []string{models.WalletColumns.EthereumAddress}, boil.Infer(), boil.Infer()); err != nil {
			return err
		}
	}

	if _, err := acct.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return err
	}

	userResp, err := d.formatUserAcctResponse(c.Context(), acct)
	if err != nil {
		return err

	}

	return c.JSON(userResp)
}

// DeleteUser godoc
// @Summary Delete the authenticated user. Fails if the user has any devices.
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Failure 409 {object} controllers.ErrorResponse "Returned if the user still has devices."
// @Router /v1/user [delete]
func (d *Controller) DeleteUser(c *fiber.Ctx) error {
	userAccount, err := getUserAccountInfos(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint

	acct, err := models.FindAccount(c.Context(), tx, userAccount.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errorResponseHandler(c, err, fiber.StatusBadRequest)
		}
		return err
	}

	// TODO-- should this now be based on eth address?
	dr, err := d.devicesClient.ListUserDevicesForUser(c.Context(), &pb.ListUserDevicesForUserRequest{UserId: acct.ID})
	if err != nil {
		return err
	}

	if l := len(dr.UserDevices); l > 0 {
		return errorResponseHandler(c, fmt.Errorf("user must delete %d devices first", l), fiber.StatusConflict)
	}

	if _, err := acct.Delete(c.Context(), d.dbs.DBS().Writer); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	d.log.Info().Str("userId", acct.ID).Msg("Deleted user.")

	return c.SendStatus(fiber.StatusNoContent)
}

// AgreeTOS godoc
// @Summary Agree to the current terms of service
// @Success 204
// @Failure 400 {object} controllers.ErrorResponse
// @Router /v1/user/agree-tos [post]
func (d *Controller) AgreeTOS(c *fiber.Ctx) error {
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

	acct.AgreedTosAt = null.TimeFrom(time.Now())

	if _, err := acct.Update(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
