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

	pb "github.com/DIMO-Network/devices-api/pkg/grpc"
	"github.com/DIMO-Network/shared/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/volatiletech/null/v8"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
	devicesClient   services.DeviceService
	amClient        pb.AftermarketDeviceServiceClient
}

type Account struct {
	DexID           string
	EthereumAddress common.Address
	EmailAddress    string
	ProviderID      string
}

func NewAccountController(settings *config.Settings, dbs db.Store, eventService services.EventService, logger *zerolog.Logger) Controller {
	// rand.New(rand.NewSource(time.Now().UnixNano()))
	var countryCodes []string
	if err := json.Unmarshal(rawCountryCodes, &countryCodes); err != nil {
		panic(err)
	}
	t := template.Must(template.New("confirmation_email").Parse(rawConfirmationEmail))

	gc, err := grpc.Dial(settings.DevicesAPIGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	dc := pb.NewUserDeviceServiceClient(gc)

	amc := pb.NewAftermarketDeviceServiceClient(gc)

	return Controller{
		Settings:        settings,
		dbs:             dbs,
		log:             logger,
		allowedLateness: 5 * time.Minute,
		countryCodes:    countryCodes,
		emailTemplate:   t,
		eventService:    eventService,
		devicesClient:   dc,
		amClient:        amc,
	}
}

func getUserAccountInfos(c *fiber.Ctx) (*Account, error) {
	var acct Account
	if token, ok := c.Locals("user").(*jwt.Token); ok {
		claims := token.Claims.(jwt.MapClaims)

		if acctID, ok := claims["sub"].(string); ok {
			acct.DexID = acctID
		}

		if provider, ok := claims["provider_id"].(string); ok {
			acct.ProviderID = provider
		}

		if addr, ok := claims["ethereum_address"].(string); ok {
			acct.EthereumAddress = common.HexToAddress(addr)
		}

		if eml, ok := claims["email_address"].(string); ok {
			if emailPattern.MatchString(eml) {
				acct.EmailAddress = eml
			}
		}

		return &acct, nil
	}
	return nil, fmt.Errorf("failed to parse user account infos")
}

func (d *Controller) getOrCreateUserAccount(c *fiber.Ctx) (*models.Wallet, error) {
	userAccount, err := getUserAccountInfos(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return nil, err
	}

	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint

	var acct *models.Wallet
	acct, err = d.getUserAccount(c.Context(), userAccount, tx)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	if acct == nil {
		acct, err = d.createUser(c.Context(), userAccount, tx)
		if err != nil {
			return nil, err
		}
	}

	return acct, nil
}

func (d *Controller) getUserAccount(ctx context.Context, userAccount *Account, tx *sql.Tx) (*models.Wallet, error) {
	var userWallet *models.Wallet
	var err error

	// AE: we can check if eth addr and email or valid and search on whichever is...
	// but it seems like we're expecting eth addr in every jwt?
	userWallet, err = models.Wallets(
		models.WalletWhere.EthereumAddress.EQ(userAccount.EthereumAddress.Bytes()),
		qm.Load(models.WalletRels.Account),
		qm.Load(qm.Rels(models.WalletRels.Account, models.AccountRels.Email)),
	).One(ctx, tx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if userWallet == nil {
		userWallet, err = d.createUser(ctx, userAccount, tx)
		if err != nil {
			return nil, err
		}
	}

	return userWallet, nil
}

func (d *Controller) createUser(ctx context.Context, userAccount *Account, tx *sql.Tx) (*models.Wallet, error) {
	referralCode, err := d.GenerateReferralCode(ctx)
	acct := models.Account{
		ID:           userAccount.DexID,
		ReferralCode: null.StringFrom(referralCode),
	}
	if err := acct.Insert(ctx, tx, boil.Infer()); err != nil {
		return nil, err
	}

	// AE Confirm: we expect to always see eth address in jwt
	mixAddr, err := common.NewMixedcaseAddressFromString(userAccount.EthereumAddress.Hex())
	if err != nil {
		return nil, fmt.Errorf("invalid ethereum_address %s", userAccount.EthereumAddress.Hex())
	}
	if !mixAddr.ValidChecksum() {
		d.log.Warn().Msgf("ethereum_address %s in ID token is not checksummed", userAccount.EthereumAddress.Hex())
	}

	wallet := &models.Wallet{
		AccountID:       userAccount.DexID,
		EthereumAddress: mixAddr.Address().Bytes(),
		Confirmed:       true, // ??
		Provider:        null.StringFrom(userAccount.ProviderID),
	}

	if err := wallet.Insert(ctx, tx, boil.Infer()); err != nil {
		return nil, fmt.Errorf("failed to insert wallet: %w", err)
	}

	var email models.Email
	if userAccount.EmailAddress != "" {
		email = models.Email{
			AccountID:    userAccount.DexID,
			EmailAddress: userAccount.EmailAddress,
			Confirmed:    false,
		}

		if err := email.Insert(ctx, tx, boil.Infer()); err != nil {
			return nil, fmt.Errorf("failed to insert email: %w", err)
		}
	}

	// do we want to change this event to include the eth addr?
	msg := UserCreationEventData{
		Timestamp: time.Now(),
		UserID:    userAccount.DexID,
		Method:    userAccount.ProviderID,
	}
	if err := d.eventService.Emit(&services.Event{
		Type:    UserCreationEventType,
		Subject: userAccount.DexID,
		Source:  "accounts-api",
		Data:    msg,
	}); err != nil {
		d.log.Err(err).Msg("Failed sending user creation event")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return wallet, nil
}

func (d *Controller) formatUserAcctResponse(ctx context.Context, userWallet *models.Wallet) (*UserResponse, error) {
	userResp := &UserResponse{
		ID: userWallet.AccountID,
	}

	acct := userWallet.R.Account
	if acct == nil {
		return nil, fmt.Errorf("account not found for wallet %s", userWallet.AccountID)
	}

	if acct.ReferredBy.Valid {
		userResp.ReferredBy = acct.ReferredBy.String
		userResp.ReferredAt = acct.ReferredAt.Time
	}

	if acct.R.Email != nil {
		userResp.Email.Address = acct.R.Email.EmailAddress
		userResp.Email.Confirmed = acct.R.Email.Confirmed
		userResp.Email.ConfirmationSentAt = acct.R.Email.ConfirmationSent.Time
	}

	if acct.R.Wallet != nil {
		userResp.Web3.Address = common.BytesToAddress(acct.R.Wallet.EthereumAddress)
		userResp.Web3.Confirmed = acct.R.Wallet.Confirmed
		userResp.Web3.Provider = acct.R.Wallet.Provider.String

		devices, err := d.devicesClient.ListUserDevicesForUser(ctx, &pb.ListUserDevicesForUserRequest{UserId: userResp.ID})
		if err != nil {
			return nil, fmt.Errorf("couldn't retrieve user's vehicles: %w", err)
		}

		for _, amd := range devices.UserDevices {
			if amd.TokenId != nil {
				userResp.Web3.Used = true
				break
			}
		}

		if !userResp.Web3.Used {
			ams, err := d.amClient.ListAftermarketDevicesForUser(ctx, &pb.ListAftermarketDevicesForUserRequest{UserId: userResp.ID})
			if err != nil {
				return nil, fmt.Errorf("couldn't retrieve user's aftermarket devices: %w", err)
			}

			for _, am := range ams.AftermarketDevices {
				if len(am.OwnerAddress) == 20 {
					userResp.Web3.Used = true
					break
				}
			}
		}

	}

	return userResp, nil
}

type ErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
}

func errorResponseHandler(c *fiber.Ctx, err error, status int) error {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return c.Status(status).JSON(ErrorResponse{msg})
}

func inSorted(v []string, x string) bool {
	i := sort.SearchStrings(v, x)
	return i < len(v) && v[i] == x
}
