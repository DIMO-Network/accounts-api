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

	"github.com/golang-jwt/jwt/v5"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func (d *Controller) getOrCreateUserAccount(c *fiber.Ctx) (*models.Account, error) {
	userAccount, err := getUserAccountInfos(c)
	if err != nil {
		d.log.Err(err).Msg("failed to parse user")
		return nil, err
	}

	acct, err := models.Accounts(
		models.AccountWhere.ID.EQ(userAccount.ID),
		qm.Load(models.AccountRels.Email),
		qm.Load(models.AccountRels.Wallet),
	).One(c.Context(), d.dbs.DBS().Reader)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	if acct == nil {
		acct, err = d.createUser(c, userAccount)
		if err != nil {
			return nil, err
		}
	}

	return acct, nil
}

func (d *Controller) createUser(c *fiber.Ctx, userAccount *Account) (*models.Account, error) {
	tx, err := d.dbs.DBS().Writer.BeginTx(c.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint

	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	providerID, ok := getStringClaim(claims, "provider_id")
	if !ok {
		return nil, errors.New("no provider_id claim in ID token")
	}

	referralCode, err := d.GenerateReferralCode(c.Context())

	acct := &models.Account{
		ID:           userAccount.ID,
		ReferralCode: null.StringFrom(referralCode),
	}
	if err := acct.Insert(c.Context(), tx, boil.Infer()); err != nil {
		return nil, err
	}

	var email models.Email
	var wallet models.Wallet
	switch providerID {
	case "apple", "google":
		emailAddr, ok := getStringClaim(claims, "email")
		if !ok {
			return nil, fmt.Errorf("provider %s but no email claim in ID token", providerID)
		}
		if !emailPattern.MatchString(emailAddr) {
			return nil, fmt.Errorf("invalid email address %s", email)
		}

		email.AccountID = userAccount.ID
		email.EmailAddress = emailAddr
		email.Confirmed = true

		if err := email.Insert(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
			return nil, fmt.Errorf("failed to insert email: %w", err)
		}

	case "web3":
		ethereum, ok := getStringClaim(claims, "ethereum_address")
		if !ok {
			return nil, fmt.Errorf("provider %s but no ethereum_address claim in ID token", providerID)
		}

		mixAddr, err := common.NewMixedcaseAddressFromString(ethereum)
		if err != nil {
			return nil, fmt.Errorf("invalid ethereum_address %s", ethereum)
		}
		if !mixAddr.ValidChecksum() {
			d.log.Warn().Msgf("ethereum_address %s in ID token is not checksummed", ethereum)
		}

		wallet.AccountID = userAccount.ID
		wallet.EthereumAddress = mixAddr.Address().Bytes()
		wallet.Confirmed = true
		wallet.Provider = null.StringFrom("Other")

		if err := wallet.Insert(c.Context(), d.dbs.DBS().Writer, boil.Infer()); err != nil {
			return nil, fmt.Errorf("failed to insert wallet: %w", err)
		}

	default:
		return nil, fmt.Errorf("unrecognized provider_id %s", providerID)
	}

	msg := UserCreationEventData{
		Timestamp: time.Now(),
		UserID:    userAccount.ID,
		Method:    providerID,
	}
	err = d.eventService.Emit(&services.Event{
		Type:    UserCreationEventType,
		Subject: userAccount.ID,
		Source:  "accounts-api",
		Data:    msg,
	})
	if err != nil {
		d.log.Err(err).Msg("Failed sending user creation event")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	acct.R.Email = &email
	acct.R.Wallet = &wallet

	return acct, nil
}

func (d *Controller) formatUserAcctResponse(ctx context.Context, acct *models.Account) (*UserResponse, error) {
	userResp := &UserResponse{
		ID:           acct.ID,
		CreatedAt:    acct.CreatedAt,
		CountryCode:  acct.CountryCode.String,
		AgreedTOSAt:  acct.AgreedTosAt.Time,
		ReferralCode: acct.ReferralCode.String,
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

type Account struct {
	ID              string
	EthereumAddress common.Address
}

// TODO: Is "sub" now expected to be the account ID?
func getUserAccountInfos(c *fiber.Ctx) (*Account, error) {
	var acct Account
	if token, ok := c.Locals("user").(*jwt.Token); ok {
		claims := token.Claims.(jwt.MapClaims)

		// are we sure eth addr will always be here going forward?
		if acctID, ok := claims["sub"].(string); ok {
			acct.ID = acctID
		}

		if addr, ok := claims["ethereum_address"].(string); ok {
			acct.EthereumAddress = common.HexToAddress(addr)
		}
		return &acct, nil
	}
	return nil, fmt.Errorf("failed to parse user account infos")
}

func getUserAccount(c *fiber.Ctx, tx *sql.Tx) (*models.Account, error) {
	userAccount, err := getUserAccountInfos(c)
	if err != nil {
		return nil, err
	}

	acct, err := models.Accounts(
		models.AccountWhere.ID.EQ(userAccount.ID),
		qm.Load(models.AccountRels.Email),
		qm.Load(models.AccountRels.Wallet),
	).One(c.Context(), tx)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	return acct, nil
}

func inSorted(v []string, x string) bool {
	i := sort.SearchStrings(v, x)
	return i < len(v) && v[i] == x
}

func getStringClaim(claims jwt.MapClaims, key string) (value string, ok bool) {
	if rawValue, ok := claims[key]; ok {
		if value, ok := rawValue.(string); ok {
			return value, true
		}
	}
	return "", false
}
