package controller

import (
	"accounts-api/models"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// GetOrCreateUserAccount godoc
// @Summary Get attributes for the authenticated user.
// @Produce json
// @Success 200 {object} controllers.UserResponse
// @Failure 403 {object} controllers.ErrorResponse
// @Security BearerAuth
// @Router /v1/user [get]
func (d *Controller) GetOrCreateUserAccount(c *fiber.Ctx) error {
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
		d.log.Err(err).Msg("failed to get user account")
		return err
	}

	var body UserUpdateRequest
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	if body.CountryCode.Defined {
		if body.CountryCode.Value.Valid && !inSorted(d.countryCodes, body.CountryCode.Value.String) {
			return errors.New("invalid country code")
		}
		acct.CountryCode = body.CountryCode.Value
	}

	// TODO AE: are we still allowing this type of update or will the user have to go through the Link endpoint?
	// assuming we are no longer allowing someone to link an eth addr through this endpoint
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

		if _, err := email.Update(c.Context(), tx, boil.Infer()); err != nil {
			return err
		}
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

	acct, err := models.Accounts(
		models.AccountWhere.DexID.EQ(userAccount.DexID),
		qm.Load(models.AccountRels.Wallet)).One(c.Context(), tx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return err
		}
		return err
	}

	if acct.R.Wallet != nil {
		if ownedVehicles, err := d.identityService.VehiclesOwned(c.Context(), common.BytesToAddress(acct.R.Wallet.EthereumAddress)); err != nil {
			return err
		} else if ownedVehicles {
			return fmt.Errorf("user must burn on-chain vehicles before deleting account")
		}
	}

	if _, err := acct.Delete(c.Context(), tx); err != nil {
		fmt.Println("Error: ", err)
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

	acct.AgreedTosAt = null.TimeFrom(time.Now())

	if _, err := acct.Update(c.Context(), tx, boil.Infer()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
