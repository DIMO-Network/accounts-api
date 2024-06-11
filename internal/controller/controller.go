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
	eventService    services.EventService
	identityService services.IdentityService
	emailService    services.EmailService
	jwkResource     keyfunc.Keyfunc
	emailTemplate   *template.Template
}

type Account struct {
	DexID           string
	EthereumAddress common.Address
	EmailAddress    string
	ProviderID      string
}

func NewAccountController(ctx context.Context, dbs db.Store, eventService services.EventService, idSvc services.IdentityService, emlSvc services.EmailService, settings *config.Settings, logger *zerolog.Logger) (*Controller, error) {
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
		eventService:    eventService,
		emailService:    emlSvc,
		identityService: idSvc,
		jwkResource:     jwkResource,
		emailTemplate:   template.Must(template.New("confirmation_email").Parse(rawConfirmationEmail)),
	}, nil
}

func getuserAccountInfosToken(c *fiber.Ctx) (*Account, error) {
	token, ok := c.Locals("user").(*jwt.Token)
	if !ok {
		return nil, errors.New("failed to get user token")
	}

	infos := getUserAccountInfos(token.Claims.(jwt.MapClaims))
	if infos.DexID == "" && infos.ProviderID == "" && (infos.EthereumAddress.Hex() == "" || infos.EmailAddress == "") {
		return nil, errors.New("failed to parse user account infos")
	}

	return infos, nil
}

func getUserAccountInfos(claims jwt.MapClaims) *Account {
	var acct Account
	if acctID, ok := claims["sub"].(string); ok {
		acct.DexID = acctID
	}

	if provider, ok := claims["provider_id"].(string); ok {
		acct.ProviderID = provider
	}

	if addr, ok := claims["ethereum_address"].(string); ok {
		acct.EthereumAddress = common.HexToAddress(addr)
	}

	if eml, ok := claims["email"].(string); ok {
		if emailPattern.MatchString(eml) {
			acct.EmailAddress = eml
		}
	}

	return &acct
}

func (d *Controller) getOrCreateUserAccount(c *fiber.Ctx) (*models.Account, error) {
	userAccount, err := getuserAccountInfosToken(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return nil, err
	}

	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint

	if exists, err := models.Accounts(
		models.AccountWhere.DexID.EQ(userAccount.DexID),
	).Exists(c.Context(), tx); err != nil {
		return nil, err
	} else if !exists {
		if err := d.createUser(c.Context(), userAccount, tx); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	acct, err := d.getUserAccount(c.Context(), userAccount, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user account: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit get or create user account tx: %w", err)
	}

	return acct, nil
}

func (d *Controller) getUserAccount(ctx context.Context, userAccount *Account, exec boil.ContextExecutor) (*models.Account, error) {
	return models.Accounts(
		models.AccountWhere.DexID.EQ(userAccount.DexID),
		qm.Load(models.AccountRels.Wallet),
		qm.Load(models.AccountRels.Email),
	).One(ctx, exec)
}

func (d *Controller) createUser(ctx context.Context, userAccount *Account, tx *sql.Tx) error {
	referralCode, err := d.GenerateReferralCode(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate referral code: %w", err)
	}

	acct := models.Account{
		ID:           ksuid.New().String(),
		DexID:        userAccount.DexID,
		ReferralCode: null.StringFrom(referralCode),
	}

	if err := acct.Insert(ctx, tx, boil.Infer()); err != nil {
		return err
	}

	switch userAccount.ProviderID {
	case "web3":
		mixAddr, err := common.NewMixedcaseAddressFromString(userAccount.EthereumAddress.Hex())
		if err != nil {
			return fmt.Errorf("invalid ethereum_address %s", userAccount.EthereumAddress.Hex())
		}
		if !mixAddr.ValidChecksum() {
			d.log.Warn().Msgf("ethereum_address %s in ID token is not checksummed", userAccount.EthereumAddress.Hex())
		}

		wallet := &models.Wallet{
			AccountID:       acct.ID,
			DexID:           userAccount.DexID,
			EthereumAddress: mixAddr.Address().Bytes(),
			Confirmed:       true,
			// TODO AE: where are we getting the provider from? how is this passed?
		}

		if err := wallet.Insert(ctx, tx, boil.Infer()); err != nil {
			return fmt.Errorf("failed to insert wallet: %w", err)
		}
	case "apple", "google":
		email := models.Email{
			AccountID:    acct.ID,
			DexID:        userAccount.DexID,
			EmailAddress: userAccount.EmailAddress,
			Confirmed:    true,
		}

		if err := email.Insert(ctx, tx, boil.Infer()); err != nil {
			return fmt.Errorf("failed to insert email: %w", err)
		}
	}

	msg := UserCreationEventData{
		Timestamp: time.Now(),
		UserID:    userAccount.DexID,
		Method:    userAccount.ProviderID,
	}
	if err := d.eventService.Emit(&services.Event{
		Type:    UserCreationEventType,
		Subject: acct.ID,
		Source:  "accounts-api",
		Data:    msg,
	}); err != nil {
		d.log.Err(err).Msg("Failed sending account creation event")
	}

	return nil
}

func (d *Controller) formatUserAcctResponse(ctx context.Context, acct *models.Account, wallet *models.Wallet, email *models.Email) (*UserResponse, error) {
	userResp := &UserResponse{
		ID:           acct.ID,
		CreatedAt:    acct.CreatedAt,
		ReferralCode: acct.ReferralCode.String,
		ReferredBy:   acct.ReferredBy.String,
		ReferredAt:   acct.ReferredAt.Time,
		AgreedTOSAt:  acct.AgreedTosAt.Time,
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
			ConfirmationSentAt: email.ConfirmationSent.Time,
		}
	}

	if wallet != nil {
		userResp.Web3 = &UserResponseWeb3{
			Address:   common.BytesToAddress(wallet.EthereumAddress),
			Confirmed: wallet.Confirmed,
			Provider:  wallet.Provider.String,
		}

		if web3Used, err := d.identityService.VehiclesOwned(ctx, userResp.Web3.Address); err != nil {
			return nil, fmt.Errorf("couldn't retrieve user vehicles: %w", err)
		} else if web3Used {
			userResp.Web3.Used = true
		}

		if !userResp.Web3.Used {
			if web3Used, err := d.identityService.AftermarketDevicesOwned(ctx, userResp.Web3.Address); err != nil {
				return nil, fmt.Errorf("couldn't retrieve user aftermarket devices: %w", err)
			} else if web3Used {
				userResp.Web3.Used = true
			}
		}
	}

	return userResp, nil
}
