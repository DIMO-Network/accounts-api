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

	"accounts-api/internal/config"
	"accounts-api/internal/services"
	"accounts-api/models"

	"github.com/DIMO-Network/shared/db"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/segmentio/ksuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

//go:embed resources/confirmation_email.html
var rawConfirmationEmail string

// Sorted JSON array of valid ISO 3116-1 apha-3 codes
//
//go:embed resources/country_codes.json
var rawCountryCodes []byte

type Controller struct {
	dbs             db.Store
	log             *zerolog.Logger
	allowedLateness time.Duration
	countryCodes    []string
	emailService    services.EmailService
	cioService      services.CustomerIoService
	jwkResource     keyfunc.Keyfunc
	emailTemplate   *template.Template
}

type AccountClaims struct {
	jwt.RegisteredClaims
	EmailAddress    *string         `json:"email,omitempty"`
	ProviderID      *string         `json:"provider_id,omitempty"`
	EthereumAddress *common.Address `json:"ethereum_address,omitempty"`
}

func NewAccountController(ctx context.Context, dbs db.Store, emlSvc services.EmailService, cioSvc services.CustomerIoService, settings *config.Settings, logger *zerolog.Logger) (*Controller, error) {
	var countryCodes []string
	if err := json.Unmarshal(rawCountryCodes, &countryCodes); err != nil {
		return nil, err
	}

	jwkResource, err := keyfunc.NewDefaultCtx(ctx, []string{settings.JWTKeySetURL}) // Context is used to end the refresh goroutine.
	if err != nil {
		log.Fatalf("Failed to create a keyfunc.Keyfunc from the server's URL.\nError: %s", err)
	}

	return &Controller{
		dbs:             dbs,
		log:             logger,
		allowedLateness: settings.AllowableEmailConfirmationLateness * time.Minute,
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

	if infos.EthereumAddress == nil && infos.EmailAddress != nil {
		return nil, errors.New("neither ethereum address nor email address present")
	}

	if infos.EmailAddress != nil {
		if !emailPattern.MatchString(*infos.EmailAddress) {
			return nil, errors.New("invalid email address")
		}
	}
	if infos.EthereumAddress != nil && infos.EmailAddress != nil {
		return nil, errors.New("token has both an ethereum address and an email address")
	}

	return infos, nil
}

func (d *Controller) getUserAccount(ctx context.Context, userAccount *AccountClaims, exec boil.ContextExecutor) (*models.Account, error) {
	if userAccount.EmailAddress != nil {
		email, err := models.Emails(
			models.EmailWhere.Address.EQ(*userAccount.EmailAddress),
			qm.Load(qm.Rels(models.EmailRels.Account, models.AccountRels.Wallet)),
		).One(ctx, exec)
		if err != nil {
			return nil, err
		}

		return email.R.Account, nil
	}

	if userAccount.EthereumAddress != nil {
		wallet, err := models.Wallets(
			models.WalletWhere.Address.EQ(userAccount.EthereumAddress.Bytes()),
			qm.Load(qm.Rels(models.EmailRels.Account, models.AccountRels.Email)),
		).One(ctx, exec)
		if err != nil {
			return nil, err
		}

		return wallet.R.Account, nil
	}

	return nil, errors.New("no email or wallet in authorization token")
}

func (d *Controller) createUser(ctx context.Context, userAccount *AccountClaims, tx *sql.Tx) error {
	referralCode, err := d.GenerateReferralCode(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate referral code: %w", err)
	}

	acct := models.Account{
		ID:           ksuid.New().String(),
		ReferralCode: referralCode,
	}

	if err := acct.Insert(ctx, tx, boil.Infer()); err != nil {
		return err
	}

	var cioWallet *common.Address
	var cioEmail *string
	switch *userAccount.ProviderID {
	case "web3":
		wallet := &models.Wallet{
			AccountID: acct.ID,
			Address:   userAccount.EthereumAddress.Bytes(),
		}

		if err := wallet.Insert(ctx, tx, boil.Infer()); err != nil {
			return fmt.Errorf("failed to insert wallet: %w", err)
		}

		cioWallet = userAccount.EthereumAddress
	case "apple", "google":
		email := models.Email{
			AccountID: acct.ID,
			Address:   *userAccount.EmailAddress,
			Confirmed: true,
		}

		if err := email.Insert(ctx, tx, boil.Infer()); err != nil {
			return fmt.Errorf("failed to insert email: %w", err)
		}

		cioEmail = userAccount.EmailAddress
	}

	if err := d.cioService.SendCustomerIoEvent(acct.ID, cioEmail, cioWallet); err != nil {
		return fmt.Errorf("failed to send customer.io event while creating user: %w", err)
	}

	return nil
}

func (d *Controller) formatUserAcctResponse(acct *models.Account, wallet *models.Wallet, email *models.Email) (*UserResponse, error) {
	userResp := &UserResponse{
		ID:            acct.ID,
		ReferredBy:    acct.ReferredBy.String,
		ReferredAt:    acct.ReferredAt.Time,
		AcceptedTOSAt: acct.AcceptedTosAt.Ptr(),
		CountryCode:   acct.CountryCode.String,
		CreatedAt:     acct.CreatedAt,
		UpdatedAt:     acct.UpdatedAt,
	}

	if acct.ReferredBy.Valid {
		userResp.ReferredBy = acct.ReferredBy.String
		userResp.ReferredAt = acct.ReferredAt.Time
	}

	if email != nil {
		userResp.Email = &UserResponseEmail{
			Address:            email.Address,
			Confirmed:          email.Confirmed,
			ConfirmationSentAt: email.ConfirmationSentAt.Ptr(),
		}
	}

	if wallet != nil {
		userResp.Web3 = &UserResponseWeb3{
			Address: common.BytesToAddress(wallet.Address).Hex(),
		}
		userResp.ReferralCode = acct.ReferralCode
	}

	return userResp, nil
}
