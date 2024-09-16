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
	"github.com/volatiletech/null/v8"

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
	identityService services.IdentityService
	emailService    services.EmailService
	cioService      services.CustomerIoService
	jwkResource     keyfunc.Keyfunc
	emailTemplate   *template.Template
}

type AccountClaims struct {
	EmailAddress    *string         `json:"email,omitempty"`
	ProviderID      *string         `json:"provider_id,omitempty"`
	EthereumAddress *common.Address `json:"ethereum_address,omitempty"`
	jwt.RegisteredClaims
}

func NewAccountController(ctx context.Context, dbs db.Store, idSvc services.IdentityService, emlSvc services.EmailService, cioSvc services.CustomerIoService, settings *config.Settings, logger *zerolog.Logger) (*Controller, error) {
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
		identityService: idSvc,
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

	if validEthAddr {
		if mixAddr, err := common.NewMixedcaseAddressFromString((*infos.EthereumAddress).Hex()); err != nil {
			validEthAddr = false
		} else if !mixAddr.ValidChecksum() {
			validEthAddr = false
		}
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
	queryMods := []qm.QueryMod{}

	if userAccount.EmailAddress != nil {
		queryMods = append(queryMods,
			qm.InnerJoin("accounts_api.emails e on e.account_id = accounts.id"),
			qm.Where("e.email_address = ?", *userAccount.EmailAddress),
			qm.Load(models.AccountRels.Email),
			qm.Load(models.AccountRels.Wallet),
		)
	}

	if userAccount.EthereumAddress != nil {
		queryMods = append(queryMods,
			qm.InnerJoin("accounts_api.wallets w on w.account_id = accounts.id"),
			qm.Where("w.ethereum_address = ?", userAccount.EthereumAddress.Bytes()),
			qm.Load(models.AccountRels.Email),
			qm.Load(models.AccountRels.Wallet),
		)
	}

	return models.Accounts(queryMods...,
	).One(ctx, exec)
}

func (d *Controller) createUser(ctx context.Context, userAccount *AccountClaims, tx *sql.Tx) error {
	referralCode, err := d.GenerateReferralCode(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate referral code: %w", err)
	}

	acct := models.Account{
		ID:           ksuid.New().String(), // this is also the cio id
		ReferralCode: null.StringFrom(referralCode),
	}

	if err := acct.Insert(ctx, tx, boil.Infer()); err != nil {
		return err
	}

	var cioWallet *common.Address
	var cioEmail *string
	switch *userAccount.ProviderID {
	case "web3":
		wallet := &models.Wallet{
			AccountID:       acct.ID,
			EthereumAddress: (*userAccount.EthereumAddress).Bytes(),
			Provider:        null.StringFrom(*userAccount.ProviderID),
		}

		if err := wallet.Insert(ctx, tx, boil.Infer()); err != nil {
			return fmt.Errorf("failed to insert wallet: %w", err)
		}

		cioWallet = userAccount.EthereumAddress
	case "apple", "google":
		email := models.Email{
			AccountID:    acct.ID,
			EmailAddress: *userAccount.EmailAddress,
			Confirmed:    true,
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
		ID:           acct.ID,
		CreatedAt:    acct.CreatedAt,
		ReferralCode: acct.ReferralCode.String,
		ReferredBy:   acct.ReferredBy.String,
		ReferredAt:   acct.ReferredAt.Time,
		AgreedTOSAt:  acct.AcceptedTosAt.Time,
		CountryCode:  acct.CountryCode.String,
		UpdatedAt:    acct.UpdatedAt,
	}

	if acct.ReferredBy.Valid {
		userResp.ReferredBy = acct.ReferredBy.String
		userResp.ReferredAt = acct.ReferredAt.Time
	}

	if email != nil {
		userResp.Email = &UserResponseEmail{
			Address:            email.EmailAddress,
			Confirmed:          email.Confirmed,
			ConfirmationSentAt: email.ConfirmationSentAt.Time,
		}
	}

	if wallet != nil {
		userResp.Web3 = &UserResponseWeb3{
			Address:  common.BytesToAddress(wallet.EthereumAddress),
			Provider: wallet.Provider.String,
		}
	}

	return userResp, nil
}
