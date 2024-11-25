package controller

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"time"

	"github.com/DIMO-Network/accounts-api/internal/config"
	"github.com/DIMO-Network/accounts-api/internal/services"
	"github.com/DIMO-Network/accounts-api/models"

	"github.com/DIMO-Network/shared/db"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/segmentio/ksuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

//go:embed resources/confirmation_email.html
var rawConfirmationEmail string

// Sorted JSON array of valid ISO 3116-1 apha-3 codes
//
//go:embed resources/country_codes.json
var rawCountryCodes []byte

type CIOClient interface {
	SetEmail(id, email string) error
	SetWallet(id string, wallet common.Address) error
}

type Controller struct {
	dbs             db.Store
	log             *zerolog.Logger
	allowedLateness time.Duration
	countryCodes    []string
	emailService    services.EmailService
	cioService      CIOClient
	jwkResource     keyfunc.Keyfunc
	emailTemplate   *template.Template
}

type AccountClaims struct {
	EmailAddress    *string         `json:"email,omitempty"`
	ProviderID      *string         `json:"provider_id,omitempty"`
	EthereumAddress *common.Address `json:"ethereum_address,omitempty"`
	jwt.RegisteredClaims
}

func NewAccountController(ctx context.Context, dbs db.Store, emlSvc services.EmailService, cioSvc CIOClient, settings *config.Settings, logger *zerolog.Logger) (*Controller, error) {
	var countryCodes []string
	if err := json.Unmarshal(rawCountryCodes, &countryCodes); err != nil {
		return nil, err
	}

	jwkResource, err := keyfunc.NewDefaultCtx(ctx, []string{settings.JWTKeySetURL}) // Context is used to end the refresh goroutine.
	if err != nil {
		log.Fatalf("Failed to create a keyfunc.Keyfunc from the server's URL.\nError: %s", err)
	}

	dur, err := time.ParseDuration(settings.EmailCodeDuration)
	if err != nil {
		return nil, err
	} else if dur <= 0 {
		return nil, fmt.Errorf("email confirmation code duration %s is non-positive", dur)
	}

	return &Controller{
		dbs:             dbs,
		log:             logger,
		allowedLateness: dur,
		countryCodes:    countryCodes,
		emailService:    emlSvc,
		cioService:      cioSvc,
		jwkResource:     jwkResource,
		emailTemplate:   template.Must(template.New("confirmation_email").Parse(rawConfirmationEmail)),
	}, nil
}

func getUserAccountClaims(c *fiber.Ctx) (*AccountClaims, error) {
	token, ok := c.Locals("user").(*jwt.Token)
	if !ok {
		return nil, errors.New("failed to get user token")
	}

	infos := token.Claims.(*AccountClaims)

	validEthAddr := infos.EthereumAddress != nil
	validEmlAddr := infos.EmailAddress != nil

	if validEthAddr && validEmlAddr {
		return nil, errors.New("unexpected: both email and wallet present in token")
	}

	if validEmlAddr {
		if !emailPattern.MatchString(*infos.EmailAddress) {
			validEmlAddr = false
		}
	}

	if !validEthAddr && !validEmlAddr {
		return nil, errors.New("invalid user token")
	}

	return infos, nil
}

func (d *Controller) getUserAccount(ctx context.Context, userAccount *AccountClaims, exec boil.ContextExecutor) (*models.Account, error) {
	switch {
	case userAccount.EmailAddress != nil:
		normalEmail := normalizeEmail(*userAccount.EmailAddress)
		email, err := models.Emails(
			models.EmailWhere.Address.EQ(normalEmail),
			qm.Load(qm.Rels(models.EmailRels.Account, models.AccountRels.Wallet)),
			qm.Load(qm.Rels(models.EmailRels.Account, models.AccountRels.ReferredByAccount, models.AccountRels.Wallet)),
		).One(ctx, exec)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("No account found with email %s.", normalEmail))
			}
			return nil, err
		}
		return email.R.Account, nil
	case userAccount.EthereumAddress != nil:
		wallet, err := models.Wallets(
			models.WalletWhere.Address.EQ(userAccount.EthereumAddress.Bytes()),
			qm.Load(qm.Rels(models.WalletRels.Account, models.AccountRels.Email)),
			qm.Load(qm.Rels(models.WalletRels.Account, models.AccountRels.ReferredByAccount, models.AccountRels.Wallet)),
		).One(ctx, exec)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("No account found with wallet %s.", *userAccount.EthereumAddress))
			}
			return nil, err
		}
		return wallet.R.Account, nil
	default:
		return nil, errors.New("no email or wallet in token")
	}
}

func (d *Controller) createUser(ctx context.Context, userAccount *AccountClaims, tx *sql.Tx) error {
	if userAccount.EthereumAddress != nil {
		conflict, err := models.WalletExists(ctx, tx, userAccount.EthereumAddress.Bytes())
		if err != nil {
			return err
		}

		if conflict {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Wallet %s is already linked to an account.", *userAccount.EthereumAddress))
		}
	} else if userAccount.EmailAddress != nil {
		conflict, err := models.EmailExists(ctx, tx, *userAccount.EmailAddress)
		if err != nil {
			return err
		}

		if conflict {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("Email %s is already linked to an account.", *userAccount.EmailAddress))
		}
	}

	referralCode, err := d.GenerateReferralCode(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate referral code: %w", err)
	}

	acct := models.Account{
		ID:           ksuid.New().String(), // this is also the cio id
		ReferralCode: referralCode,
	}

	if err := acct.Insert(ctx, tx, boil.Infer()); err != nil {
		return err
	}

	if userAccount.EthereumAddress != nil {
		wallet := &models.Wallet{
			AccountID: acct.ID,
			Address:   userAccount.EthereumAddress.Bytes(),
		}

		if err := wallet.Insert(ctx, tx, boil.Infer()); err != nil {
			return fmt.Errorf("failed to insert wallet: %w", err)
		}

		if err := d.cioService.SetWallet(acct.ID, *userAccount.EthereumAddress); err != nil {
			d.log.Err(err).Msg("Error sending wallet information to Customer.io.")
		}
	} else if userAccount.EmailAddress != nil {
		normalEmail := normalizeEmail(*userAccount.EmailAddress)

		email := models.Email{
			AccountID:   acct.ID,
			Address:     normalEmail,
			ConfirmedAt: null.TimeFrom(time.Now()),
		}

		if err := email.Insert(ctx, tx, boil.Infer()); err != nil {
			return fmt.Errorf("failed to insert email: %w", err)
		}

		if err := d.cioService.SetEmail(acct.ID, normalEmail); err != nil {
			d.log.Err(err).Msg("Error sending email information to Customer.io.")
		}
	}

	return nil
}

func (d *Controller) formatUserAcctResponse(acct *models.Account, wallet *models.Wallet, email *models.Email) (*UserResponse, error) {
	userResp := &UserResponse{
		ID:            acct.ID,
		CreatedAt:     acct.CreatedAt,
		AcceptedTOSAt: acct.AcceptedTosAt.Ptr(),
		CountryCode:   acct.CountryCode.Ptr(),
		UpdatedAt:     acct.UpdatedAt,
	}

	if email != nil {
		userResp.Email = &UserResponseEmail{
			Address:     email.Address,
			ConfirmedAt: email.ConfirmedAt.Ptr(),
		}
	}

	if wallet != nil {
		userResp.Wallet = &UserResponseWallet{
			Address: common.BytesToAddress(wallet.Address).Hex(),
		}

		var referredBy *string
		if acct.R.ReferredByAccount != nil && acct.R.ReferredByAccount.R.Wallet != nil {
			a := common.BytesToAddress(acct.R.ReferredByAccount.R.Wallet.Address).Hex()
			referredBy = &a
		}

		userResp.Referral = &UserResponseReferral{
			Code:       acct.ReferralCode,
			ReferredAt: acct.ReferredAt.Ptr(),
			ReferredBy: referredBy,
		}
	}

	return userResp, nil
}
