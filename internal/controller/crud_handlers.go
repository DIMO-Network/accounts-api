package controller

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/DIMO-Network/accounts-api/models"
	"github.com/gofiber/fiber/v2"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

// CreateAccount godoc
// @Summary Create user account using an auth token in the header.
// @Produce json
// @Success 201 {object} controller.UserResponse
// @Failure 400 {object} controller.ErrorRes
// @Security BearerAuth
// @Router /v1/account [post]
func (d *Controller) CreateAccount(c *fiber.Ctx) error {
	userAccount, err := getUserAccountClaims(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return err
	}

	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), &sql.TxOptions{Isolation: sql.LevelSerializable})
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

	acct, err := d.getUserAccount(c.Context(), userAccount, d.dbs.DBS().Reader)
	if err != nil {
		return err
	}

	if userAccount.EmailAddress != nil {
		d.log.Info().Str("account", acct.ID).Msgf("Created account with email %s.", *userAccount.EmailAddress)
	} else if userAccount.EthereumAddress != nil {
		d.log.Info().Str("account", acct.ID).Msgf("Created account with wallet %s.", *userAccount.EthereumAddress)
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
// @Failure 403 {object} controller.ErrorRes
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
		if errors.Is(err, sql.ErrNoRows) {
			// TODO(elffjs): Make this more precise.
			return fiber.NewError(fiber.StatusNotFound, "No account found with this email or wallet.")
		}
		return err
	}

	formattedAcct, err := d.formatUserAcctResponse(acct, acct.R.Wallet, acct.R.Email)
	if err != nil {
		return err
	}

	return c.JSON(formattedAcct)
}

var countryCodePattern = regexp.MustCompile("^[A-Z]{3}$")

// UpdateUser godoc
// @Summary Modify attributes for the authenticated user
// @Accept json
// @Produce json
// @Param userUpdateRequest body controller.UserUpdateRequest true "New field values"
// @Success 200 {object} controller.UserResponse
// @Success 400 {object} controller.ErrorRes
// @Failure 403 {object} controller.ErrorRes
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
		if !countryCodePattern.MatchString(body.CountryCode) {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Unrecognized country code %q. Country codes consist of three capital letters.", body.CountryCode))
		}

		if !slices.Contains(d.countryCodes, body.CountryCode) {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Unrecognized country code %q.", body.CountryCode))
		}

		if !acct.CountryCode.Valid || acct.CountryCode.String != body.CountryCode {
			acct.CountryCode = null.StringFrom(body.CountryCode)

			if _, err := acct.Update(c.Context(), d.dbs.DBS().Reader, boil.Whitelist(models.AccountColumns.CountryCode, models.AccountColumns.UpdatedAt)); err != nil {
				return err
			}

			d.log.Info().Str("account", acct.ID).Msgf("Updated country to %s.", body.CountryCode)
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
// @Success 200 {object} controller.StandardRes
// @Failure 400 {object} controller.ErrorRes
// @Failure 403 {object} controller.ErrorRes
// @Failure 409 {object} controller.ErrorRes "Returned if the user still has devices."
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
	return c.JSON(StandardRes{
		Message: fmt.Sprintf("Deleted account %s.", acct.ID),
	})
}

// AcceptTOS godoc
// @Summary Agree to the current terms of service
// @Success 200 {object} controller.StandardRes
// @Failure 400 {object} controller.ErrorRes
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

	if acct.AcceptedTosAt.Valid {
		return c.JSON(StandardRes{
			Message: fmt.Sprintf("Already accepted the terms of service at %s.", acct.AcceptedTosAt.Time),
		})
	}

	accTime := time.Now()

	acct.AcceptedTosAt = null.TimeFrom(accTime)

	if _, err := acct.Update(c.Context(), d.dbs.DBS().Reader, boil.Whitelist(models.AccountColumns.AcceptedTosAt, models.AccountColumns.UpdatedAt)); err != nil {
		return err
	}

	return c.JSON(StandardRes{
		Message: "Accepted the terms of service.",
	})
}
