package controller

import (
	"accounts-api/models"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

// CreateUserAccount godoc
// @Summary Create an account based on email or 0x address.
// @Produce json
// @Success 201 {object} controller.UserResponse
// @Failure 403 {object} controller.BasicResponse
// @Security BearerAuth
// @Router /v1/account [post]
func (d *Controller) CreateUserAccount(c *fiber.Ctx) error {
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

	if err := d.createUser(c.Context(), userAccount, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		d.log.Err(err).Msg("failed to commit create user account tx")
		return err
	}

	acct, err := d.getUserAccount(c.Context(), userAccount, tx)
	if err != nil {
		return err
	}

	formattedAcct, err := d.formatUserAcctResponse(acct, acct.R.Wallet, acct.R.Email)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(formattedAcct)
}

// GetUserAccount godoc
// @Summary Get attributes for the authenticated user.
// @Produce json
// @Success 200 {object} controller.UserResponse
// @Failure 403 {object} controller.BasicResponse
// @Security BearerAuth
// @Router /v1/account [get]
func (d *Controller) GetUserAccount(c *fiber.Ctx) error {
	userAccount, err := getUserAccountClaims(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	acct, err := d.getUserAccount(c.Context(), userAccount, d.dbs.DBS().Reader)
	if err != nil {
		return err
	}

	formattedAcct, err := d.formatUserAcctResponse(acct, acct.R.Wallet, acct.R.Email)
	if err != nil {
		return err
	}

	return c.JSON(formattedAcct)
}

// UpdateUser godoc
// @Summary Modify attributes for the authenticated user
// @Accept json
// @Produce json
// @Param userUpdateRequest body controller.UserUpdateRequest true "New field values"
// @Success 200 {object} controller.UserResponse
// @Success 400 {object} controller.BasicResponse
// @Failure 403 {object} controller.BasicResponse
// @Router /v1/account [put]
func (d *Controller) UpdateUser(c *fiber.Ctx) error {
	userAccount, err := getUserAccountClaims(c)
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
		if !slices.Contains(d.countryCodes, body.CountryCode) {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Unrecognized country code %q.", body.CountryCode))
		}

		acct.CountryCode = null.StringFrom(body.CountryCode)

		// This will get more awkward if there are ever more fields.
		if _, err := acct.Update(c.Context(), d.dbs.DBS().Reader, boil.Whitelist(models.AccountColumns.CountryCode, models.AccountColumns.UpdatedAt)); err != nil {
			return err
		}
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
// @Failure 400 {object} controller.BasicResponse
// @Failure 403 {object} controller.BasicResponse
// @Failure 409 {object} controller.BasicResponse "Returned if the user still has devices."
// @Router /v1/account [delete]
func (d *Controller) DeleteUser(c *fiber.Ctx) error {
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
		if errors.Is(err, sql.ErrNoRows) {
			return err
		}
		return err
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

// AcceptTOS godoc
// @Summary Agree to the current terms of service.
// @Success 204
// @Failure 400 {object} controller.BasicResponse
// @Router /v1/account/accept-tos [post]
func (d *Controller) AcceptTOS(c *fiber.Ctx) error {
	userAccount, err := getUserAccountClaims(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	acct, err := d.getUserAccount(c.Context(), userAccount, d.dbs.DBS().Reader)
	if err != nil {
		return err
	}

	// TODO(elffjs): Make a note if they already agreed.
	acct.AcceptedTosAt = null.TimeFrom(time.Now())

	if _, err := acct.Update(c.Context(), d.dbs.DBS().Reader, boil.Whitelist(models.AccountColumns.AcceptedTosAt, models.AccountColumns.UpdatedAt)); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}
