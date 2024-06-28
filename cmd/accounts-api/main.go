package main

import (
	"accounts-api/internal/config"
	"accounts-api/internal/services"
	"accounts-api/models"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/DIMO-Network/shared"
	"github.com/DIMO-Network/shared/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/segmentio/ksuid"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"

	"accounts-api/internal/controller"

	"github.com/gocarina/gocsv"
	"github.com/goccy/go-json"
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/swagger"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

// @title DIMO Accounts API
// @version 1.0
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("app", "accounts-api").Logger()

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && len(s.Value) == 40 {
				logger = logger.With().Str("commit", s.Value[:7]).Logger()
				break
			}
		}
	}

	settings, err := shared.LoadConfig[config.Settings]("settings.yaml")
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load settings")
	}

	dbs := db.NewDbConnectionFromSettings(ctx, &settings.DB, true)
	dbs.WaitForDB(logger)

	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}
	switch arg {
	case "migrate": // run migrations and complete
		command := "up"
		if len(os.Args) > 2 {
			command = os.Args[2]
			if command == "down-to" || command == "up-to" {
				command = command + " " + os.Args[3]
			}
		}
		if err := migrateDatabase(ctx, logger, &settings.DB, command, "migrations"); err != nil {
			logger.Fatal().Err(err).Msg("Failed to migrate datbase.")
		}

		return
	case "upload": // upload data from users db
		failures := 0
		successes := 0
		if len(os.Args) != 3 {
			logger.Fatal().Msg("Usage: go run ./cmd/accounts-api upload <path>")
		}

		filepath := os.Args[2]
		f, err := os.Open(filepath)
		if err != nil {
			logger.Fatal().Err(err).Msg("Unable to read input file " + filepath)
		}
		defer f.Close()

		users := []*User{}
		uploadsFailed := []User{}
		if err := gocsv.UnmarshalFile(f, &users); err != nil {
			logger.Fatal().Err(err).Msg("Unable to parse file as CSV for " + filepath)
		}

		refCodeMap := map[string]string{}
		timeFormat := "2006-01-0215:04:05+00"
		for idx, user := range users {
			refCodeMap[user.UserID] = user.ReferralCode
			created, _ := time.Parse(timeFormat, strings.Replace(users[idx].CreatedAt, " ", "", -1))
			users[idx].CreateAtTime = created
		}

		sort.Slice(users, func(i, j int) bool {
			return users[i].CreateAtTime.Before(users[j].CreateAtTime)
		})

		for idx, user := range users {
			if failures == idx+1 {
				return
			}

			createdAt, err := time.Parse(timeFormat, strings.Replace(user.CreatedAt, " ", "", -1))
			if err != nil {
				logger.Error().Err(err).Str("created_at", user.CreatedAt).Msg("Failed to parse created_at")
				failures++
				continue
			}

			if user.EthereumAddress == "" && user.Email == "" {
				uploadsFailed = append(uploadsFailed, *user)
				uploadsFailed[len(uploadsFailed)-1].ErrorReason = "No email or ethereum address"
				failures++
				continue
			}

			acct := models.Account{
				ID:           ksuid.New().String(),
				CustomerIoID: null.StringFrom(user.UserID),
				CreatedAt:    createdAt,
				UpdatedAt:    createdAt,
			}

			if user.ReferralCode != "" {
				acct.ReferralCode = null.StringFrom(user.ReferralCode)
			}
			if user.CountryCode != "" {
				acct.CountryCode = null.StringFrom(user.CountryCode)
			}

			if user.ReferredAt != "" && user.ReferringUserID != "" {
				referredAt, err := time.Parse(timeFormat, strings.Replace(user.ReferredAt, " ", "", -1))
				if err != nil {
					logger.Error().Err(err).Str("referred_at", user.ReferredAt).Msg("Failed to parse referred_at")
					failures++
					continue
				}

				acct.ReferredAt = null.TimeFrom(referredAt)
				acct.ReferredBy = null.StringFrom(refCodeMap[user.ReferringUserID])
			}

			if user.AgreedTOSAt != "" {
				acceptedTOS, err := time.Parse(timeFormat, strings.Replace(user.AgreedTOSAt, " ", "", -1))
				if err != nil {
					logger.Error().Err(err).Str("accepted_tos_at", user.AgreedTOSAt).Msg("Failed to parse accepted_tos_at")
					failures++
					continue
				}

				acct.AcceptedTosAt = null.TimeFrom(acceptedTOS)
			}

			if err := acct.Insert(ctx, dbs.DBS().Writer, boil.Infer()); err != nil {
				uploadsFailed = append(uploadsFailed, *user)
				uploadsFailed[len(uploadsFailed)-1].ErrorReason = fmt.Sprintf("Failed to insert account: %v", err.Error())
				failures++
				continue
			}

			if user.Email != "" {
				email := models.Email{
					AccountID:    acct.ID,
					EmailAddress: user.Email,
					Confirmed:    user.EmailConfirmed,
				}

				if user.EmailConfirmed && user.EmailConfirmationKey != "" {
					confirmationSent, err := time.Parse(timeFormat, strings.Replace(user.EmailConfirmationSentAt, " ", "", -1))
					if err != nil {
						uploadsFailed = append(uploadsFailed, *user)
						uploadsFailed[len(uploadsFailed)-1].ErrorReason = fmt.Sprintf("Failed to parse email confirmation sent time: %v", err.Error())
						failures++
						continue
					}

					email.ConfirmationSentAt = null.TimeFrom(confirmationSent)
					email.ConfirmationCode = null.StringFrom(user.EmailConfirmationKey)
				}

				if err := email.Insert(ctx, dbs.DBS().Writer, boil.Infer()); err != nil {
					uploadsFailed = append(uploadsFailed, *user)
					uploadsFailed[len(uploadsFailed)-1].ErrorReason = fmt.Sprintf("Failed to insert email address for user: %v", err.Error())
					failures++
					continue
				}
			}

			if user.EthereumAddress != "" && user.EthereumConfirmed {
				mixAddr, err := common.NewMixedcaseAddressFromString(common.HexToAddress(strings.Replace(user.EthereumAddress, `\x`, "0x", -1)).Hex())
				if err != nil {
					uploadsFailed = append(uploadsFailed, *user)
					uploadsFailed[len(uploadsFailed)-1].ErrorReason = fmt.Sprintf("Failed to parse eth addr for user: %v", err.Error())
					failures++
					continue
				}

				if !mixAddr.ValidChecksum() {
					uploadsFailed = append(uploadsFailed, *user)
					uploadsFailed[len(uploadsFailed)-1].ErrorReason = fmt.Sprintf("valid checksum failed for addr")
					failures++
					continue
				}

				wallet := &models.Wallet{
					AccountID:       acct.ID,
					EthereumAddress: mixAddr.Address().Bytes(),
					Provider:        null.StringFrom("web3"),
				}

				if err := wallet.Insert(ctx, dbs.DBS().Writer, boil.Infer()); err != nil {
					uploadsFailed = append(uploadsFailed, *user)
					uploadsFailed[len(uploadsFailed)-1].ErrorReason = fmt.Sprintf("Failed to insert wallet for user: %v", err.Error())
					failures++
					continue
				}

			}
			successes++
		}

		fmt.Println(fmt.Sprintf("Failed to upload %d users", len(uploadsFailed)))
		file2, err := os.Create("upload_failures.csv")
		if err != nil {
			panic(err)
		}
		defer file2.Close()

		writer := csv.NewWriter(file2)
		defer writer.Flush()
		headers := []string{
			"id", "email_address", "email_confirmed", "email_confirmation_sent_at",
			"email_confirmation_key", "created_at", "country_code", "ethereum_address",
			"agreed_tos_at", "auth_provider_id", "ethereum_confirmed", "in_app_wallet",
			"referral_code", "referred_at", "referring_user_id", "error_reason",
		}

		writer.Write(headers)
		for _, u := range uploadsFailed {
			row := []string{
				u.UserID, u.Email, strconv.FormatBool(u.EmailConfirmed), u.EmailConfirmationSentAt,
				u.EmailConfirmationKey, u.CreatedAt, u.CountryCode, u.EthereumAddress,
				u.AgreedTOSAt, u.AuthProviderID, strconv.FormatBool(u.EthereumConfirmed), strconv.FormatBool(u.InAppWallet),
				u.ReferralCode, u.ReferredAt, u.ReferringUserID, u.ErrorReason,
			}
			writer.Write(row)
		}
		return
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return errorHandler(c, err, &logger, settings.IsProduction())
		},
		DisableStartupMessage: true,
		ReadBufferSize:        16000,
		BodyLimit:             10 * 1024 * 1024,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
	})

	app.Get("/", healthCheck)

	go func() {
		monApp := fiber.New(fiber.Config{DisableStartupMessage: true})
		monApp.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
		if err := monApp.Listen(":" + settings.MonitoringPort); err != nil {
			logger.Fatal().Err(err).Str("port", settings.MonitoringPort).Msg("Failed to start monitoring web server.")
		}
	}()

	app.Get("/v1/swagger/*", swagger.HandlerDefault)

	v1 := app.Group("/v1/account", jwtware.New(
		jwtware.Config{
			JWKSetURLs: []string{settings.JWTKeySetURL},
			Claims:     &controller.AccountClaims{},
		},
	))

	idSvc := services.NewIdentityService(&settings)
	emailSvc := services.NewEmailService(&settings)
	accountController, err := controller.NewAccountController(ctx, dbs, idSvc, emailSvc, &settings, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to start account controller")
	}

	//create account based on 0x or email
	v1.Post("/", accountController.CreateUserAccount)

	//fetch account information based on whether the 0x or email links to an existing account
	//search is performed through wallets or emails table, whichever way you came in
	v1.Get("/", accountController.GetUserAccount)

	//update account other data(region,etc)
	v1.Put("/update", accountController.UpdateUser)

	//delete account and all associated links, cascade
	v1.Delete("/", accountController.DeleteUser)

	//agree to terms of service, can only be called after both email and wallet are linked
	v1.Post("/agree-tos", accountController.AgreeTOS)

	//agree to terms of service, can only be called after both email and wallet are linked
	v1.Post("/referral/submit", accountController.SubmitReferralCode)

	//link a wallet to the account, required a signed JWT from auth server
	v1.Post("/link/wallet/token", accountController.LinkWalletToken)

	//link a google account to the account, required a signed JWT from auth server
	v1.Post("/link/email/token", accountController.LinkEmailToken)

	//link some other email to the account, no JWT can be provider, so code is sent.
	v1.Post("/link/email", accountController.LinkEmail)

	//confirm the email code
	v1.Post("/link/email/confirm", accountController.ConfirmEmail)

	logger.Info().Msg("Server started on port " + settings.Port)

	// Start Server
	if err := app.Listen(":" + settings.Port); err != nil {
		logger.Fatal().Err(err).Send()
	}

}

func migrateDatabase(ctx context.Context, _ zerolog.Logger, settings *db.Settings, command, migrationsDir string) error {
	db, err := sql.Open("postgres", settings.BuildConnectionString(true))
	if err != nil {
		return err
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return err
	}

	if command == "" {
		command = "up"
	}

	_, err = db.Exec("CREATE SCHEMA IF NOT EXISTS accounts_api;")
	if err != nil {
		return err
	}
	goose.SetTableName("accounts_api.migrations")
	return goose.RunContext(ctx, command, db, migrationsDir)
}

func healthCheck(c *fiber.Ctx) error {
	res := map[string]interface{}{
		"data": "Server is up and running",
	}

	err := c.JSON(res)

	if err != nil {
		return err
	}

	return nil
}

func getLogger(c *fiber.Ctx, d *zerolog.Logger) *zerolog.Logger {
	m := c.Locals("logger")
	if m == nil {
		return d
	}

	l, ok := m.(*zerolog.Logger)
	if !ok {
		return d
	}

	return l
}

// ErrorHandler custom handler to log recovered errors using our logger and return json instead of string
func errorHandler(c *fiber.Ctx, err error, logger *zerolog.Logger, isProduction bool) error {
	logger = getLogger(c, logger)

	code := fiber.StatusInternalServerError // Default 500 statuscode

	var e *fiber.Error
	isFiberErr := errors.As(err, &e)
	if isFiberErr {
		// Override status code if fiber.Error type
		code = e.Code
	}

	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	logger.Err(err).Int("httpStatusCode", code).
		Str("httpMethod", c.Method()).
		Str("httpPath", c.Path()).
		Msg("caught an error from http request")

	// return an opaque error if we're in a higher level environment and we haven't specified an fiber type err.
	if !isFiberErr && isProduction {
		err = fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	return c.Status(code).JSON(controller.ErrorRes{
		Code:    code,
		Message: err.Error(),
	})
}

type User struct {
	UserID                  string `csv:"id"`
	Email                   string `csv:"email_address"`
	EmailConfirmed          bool   `csv:"email_confirmed"`
	EmailConfirmationSentAt string `csv:"email_confirmation_sent_at"`
	EmailConfirmationKey    string `csv:"email_confirmation_key"`
	CreatedAt               string `csv:"created_at"`
	CountryCode             string `csv:"country_code"`
	EthereumAddress         string `csv:"ethereum_address"`
	AgreedTOSAt             string `csv:"agreed_tos_at"`
	AuthProviderID          string `csv:"auth_provider_id"`
	EthereumConfirmed       bool   `csv:"ethereum_confirmed"`
	InAppWallet             bool   `csv:"in_app_wallet"`
	ReferralCode            string `csv:"referral_code"`
	ReferredAt              string `csv:"referred_at"`
	ReferringUserID         string `csv:"referring_user_id"`
	ErrorReason             string `json:"error_reason" csv:"error_reason"`
	CreateAtTime            time.Time
}
