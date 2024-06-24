package controller

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

// GetOrCreateUserAccount godoc
// @Summary Get attributes for the authenticated user.
// @Produce json
// @Success 200 {object} controller.UserResponse
// @Failure 403 {object} controller.ErrorRes
// @Security BearerAuth
// @Router /v1/accounts [get]
func (d *Controller) GetOrCreateUserAccount(c *fiber.Ctx) error {
	acct, err := d.getOrCreateUserAccount(c)
	if err != nil {
		return err
	}

	userResp, err := d.formatUserAcctResponse(acct, acct.R.Wallet, acct.R.Email)
	if err != nil {
		return err
	}

	return c.JSON(userResp)
}

// UpdateUser godoc
// @Summary Modify attributes for the authenticated user
// @Accept json
// @Produce json
// @Param userUpdateRequest body controller.UserUpdateRequest true "New field values"
// @Success 200 {object} controller.UserResponse
// @Success 400 {object} controller.ErrorRes
// @Failure 403 {object} controller.ErrorRes
// @Router /v1/accounts [put]
func (d *Controller) UpdateUser(c *fiber.Ctx) error {
	userAccount, err := getuserAccountInfosToken(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	acct, err := d.getUserAccount(c.Context(), userAccount, d.dbs.DBS().Reader)
	if err != nil {
		d.log.Err(err).Msg("failed to get user account")
		return err
	}

	var body UserUpdateRequest
	if err := c.BodyParser(&body); err != nil {
		return err
	}

	if body.CountryCode != "" {
		acct.CountryCode = null.StringFrom(body.CountryCode)
	}

	if _, err := acct.Update(c.Context(), d.dbs.DBS().Reader, boil.Infer()); err != nil {
		return err
	}

	userResp, err := d.formatUserAcctResponse(acct, acct.R.Wallet, acct.R.Email)
	if err != nil {
		return err
	}

	return c.JSON(userResp)
}

// DeleteUser godoc
// @Summary Delete the authenticated user. Fails if the user has any devices.
// @Success 204
// @Failure 400 {object} controller.ErrorRes
// @Failure 403 {object} controller.ErrorRes
// @Failure 409 {object} controller.ErrorRes "Returned if the user still has devices."
// @Router /v1/accounts [delete]
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

	acct, err := d.getUserAccount(c.Context(), userAccount, tx)
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
// @Failure 400 {object} controller.ErrorRes
// @Router /v1/accounts/agree-tos [post]
func (d *Controller) AgreeTOS(c *fiber.Ctx) error {
	userAccount, err := getuserAccountInfosToken(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	acct, err := d.getUserAccount(c.Context(), userAccount, d.dbs.DBS().Reader)
	if err != nil {
		return err
	}

	acct.AgreedTosAt = null.TimeFrom(time.Now())

	if _, err := acct.Update(c.Context(), d.dbs.DBS().Reader, boil.Infer()); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
