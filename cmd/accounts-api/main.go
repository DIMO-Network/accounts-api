package main

import (
	"accounts-api/internal/config"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strconv"

	"github.com/DIMO-Network/shared"
	"github.com/DIMO-Network/shared/db"

	"accounts-api/internal/controller"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/swagger"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

func main() {
	gitSha1 := os.Getenv("GIT_SHA1")
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "accounts-api").
		Str("git-sha1", gitSha1).
		Logger()

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

	v1 := app.Group("/v1/account", jwtware.New(jwtware.Config{
		JWKSetURLs: []string{settings.JWTKeySetURL},
	}))

	accountController := controller.NewAccountController(&settings, dbs, nil, &logger)

	//get or create account based on whether the 0x or email links to an existing account
	//search is performed through wallets or emails table, whichever way you came in
	// ^ ae- aren't we assuming people will only be coming in via wallet?
	v1.Get("/", accountController.GetUserAccount)

	//update account other data(region,etc)
	v1.Put("/", accountController.UpdateUser)
	//delete account and all associated links, cascade
	v1.Delete("/", accountController.DeleteUser)
	//agree to terms of service, can only be called after both email and wallet are linked
	v1.Post("/agree-tos", accountController.AgreeTOS)
	//agree to terms of service, can only be called after both email and wallet are linked
	v1.Post("/referral/submit", accountController.SubmitReferralCode)
	//link a wallet to the account, required a signed JWT from auth server
	v1.Post("/link/wallet/token", accountController.GenerateEthereumChallenge)
	//link a google account to the account, required a signed JWT from auth server
	v1.Post("/link/email/token", accountController.GenerateEthereumChallenge)
	//link some other email to the account, no JWT can be provider, so code is sent.
	v1.Post("/link/email", accountController.GenerateEthereumChallenge)
	//confirm the email code
	v1.Post("/link/email/confirm", accountController.GenerateEthereumChallenge)

	logger.Info().Msg("Server started on port " + settings.Port)

	// go startGRPCServer(settings, dbs, &logger)

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

	_, err = db.Exec("CREATE SCHEMA IF NOT EXISTS users_api;")
	if err != nil {
		return err
	}
	goose.SetTableName("users_api.migrations")
	return goose.RunContext(ctx, command, db, migrationsDir)
}

// TODO
func startGRPCServer(settings *config.Settings, dbs db.Store, logger *zerolog.Logger) {
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

type ErrorRes struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const skipErrorLogKey = "skipErrorLog"

func SkipErrorLog(c *fiber.Ctx) {
	c.Locals(skipErrorLogKey, true)
}

func GetLogger(c *fiber.Ctx, d *zerolog.Logger) *zerolog.Logger {
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
	logger = GetLogger(c, logger)

	code := fiber.StatusInternalServerError // Default 500 statuscode

	var e *fiber.Error
	isFiberErr := errors.As(err, &e)
	if isFiberErr {
		// Override status code if fiber.Error type
		code = e.Code
	}
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	codeStr := strconv.Itoa(code)

	if c.Locals(skipErrorLogKey) != true {
		logger.Err(err).Str("httpStatusCode", codeStr).
			Str("httpMethod", c.Method()).
			Str("httpPath", c.Path()).
			Msg("caught an error from http request")
	}
	// return an opaque error if we're in a higher level environment and we haven't specified an fiber type err.
	if !isFiberErr && isProduction {
		err = fiber.NewError(fiber.StatusInternalServerError, "Internal error")
	}

	return c.Status(code).JSON(ErrorRes{
		Code:    code,
		Message: err.Error(),
	})
}
