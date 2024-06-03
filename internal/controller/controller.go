package controller

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"sort"
	"time"

	"accounts-api/internal/config"
	"accounts-api/internal/services"
	"accounts-api/models"

	"github.com/DIMO-Network/shared/db"
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

// Sorted JSON array of valid ISO 3116-1 apha-3 codes
//
//go:embed resources/country_codes.json
var rawCountryCodes []byte

//go:embed resources/confirmation_email.html
var rawConfirmationEmail string

type Controller struct {
	Settings        *config.Settings
	dbs             db.Store
	log             *zerolog.Logger
	allowedLateness time.Duration
	countryCodes    []string
	emailTemplate   *template.Template
	eventService    services.EventService
	identityService services.IdentityService
}

type Account struct {
	DexID           string
	EthereumAddress common.Address
	EmailAddress    string
	ProviderID      string
}

func NewAccountController(settings *config.Settings, dbs db.Store, eventService services.EventService, idSvc services.IdentityService, logger *zerolog.Logger) (*Controller, error) {
	var countryCodes []string
	if err := json.Unmarshal(rawCountryCodes, &countryCodes); err != nil {
		return nil, err
	}

	return &Controller{
		Settings:        settings,
		dbs:             dbs,
		log:             logger,
		allowedLateness: settings.AllowableEmailConfirmationLateness * time.Minute,
		countryCodes:    countryCodes,
		emailTemplate:   template.Must(template.New("confirmation_email").Parse(rawConfirmationEmail)),
		eventService:    eventService,
		identityService: idSvc,
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

func (d *Controller) getUserAccount(ctx context.Context, userAccount *Account, tx *sql.Tx) (*models.Account, error) {
	return models.Accounts(
		models.AccountWhere.DexID.EQ(userAccount.DexID),
		qm.Load(models.AccountRels.Wallet),
		qm.Load(models.AccountRels.Email),
	).One(ctx, tx)
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
			Provider:        null.StringFrom("Turnkey"), // where are we getting this from? what are valid options?
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

func (d *Controller) formatUserAcctResponse(ctx context.Context, userAccount *models.Account) (*UserResponse, error) {
	userResp := &UserResponse{
		ID: userAccount.ID,
	}

	if userAccount.ReferredBy.Valid {
		userResp.ReferredBy = userAccount.ReferredBy.String
		userResp.ReferredAt = userAccount.ReferredAt.Time
	}

	if userAccount.R.Email != nil {
		userResp.Email = &UserResponseEmail{
			Address:            userAccount.R.Email.EmailAddress,
			Confirmed:          userAccount.R.Email.Confirmed,
			ConfirmationSentAt: userAccount.R.Email.ConfirmationSent.Time,
		}
	}

	if userAccount.R.Wallet != nil {
		userResp.Web3 = &UserResponseWeb3{
			Address:   common.BytesToAddress(userAccount.R.Wallet.EthereumAddress),
			Confirmed: userAccount.R.Wallet.Confirmed,
			Provider:  userAccount.R.Wallet.Provider.String,
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

func inSorted(v []string, x string) bool {
	i := sort.SearchStrings(v, x)
	return i < len(v) && v[i] == x
}
