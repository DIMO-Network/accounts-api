package test

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"accounts-api/internal/config"
	"accounts-api/internal/services"
	"accounts-api/models"

	"github.com/DIMO-Network/shared/db"
	"github.com/docker/go-connections/nat"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"github.com/segmentio/ksuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

const testDbName = "accounts_api"

// StartContainerDatabase starts postgres container with default test settings, and migrates the db. Caller must terminate container.
func StartContainerDatabase(ctx context.Context, t *testing.T, migrationsDirRelPath string) (db.Store, testcontainers.Container) {
	settings := getTestDbSettings()
	pgPort := "5432/tcp"
	dbURL := func(_ string, port nat.Port) string {
		return fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", settings.DB.User, settings.DB.Password, port.Port(), settings.DB.Name)
	}

	cr := testcontainers.ContainerRequest{
		Image:        "postgres:12.9-alpine",
		Env:          map[string]string{"POSTGRES_USER": settings.DB.User, "POSTGRES_PASSWORD": settings.DB.Password, "POSTGRES_DB": settings.DB.Name},
		ExposedPorts: []string{pgPort},
		Cmd:          []string{"postgres", "-c", "fsync=off"},
		WaitingFor:   wait.ForSQL(nat.Port(pgPort), "postgres", dbURL),
	}

	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: cr,
		Started:          true,
	})
	if err != nil {
		return handleContainerStartErr(ctx, err, pgContainer, t)
	}
	mappedPort, err := pgContainer.MappedPort(ctx, nat.Port(pgPort))
	if err != nil {
		return handleContainerStartErr(ctx, errors.Wrap(err, "failed to get container external port"), pgContainer, t)
	}
	fmt.Printf("postgres container session %s ready and running at port: %s \n", pgContainer.SessionID(), mappedPort)
	//defer pgContainer.Terminate(ctx) // this should be done by the caller

	settings.DB.Port = mappedPort.Port()
	pdb := db.NewDbConnectionForTest(ctx, &settings.DB, false)
	for !pdb.IsReady() {
		time.Sleep(500 * time.Millisecond)
	}
	// can't connect to db, dsn=user=postgres password=postgres dbname=accounts_api host=localhost port=49395 sslmode=disable search_path=accounts_api, err=EOF
	// error happens when calling here
	_, err = pdb.DBS().Writer.Exec(`
		grant usage on schema public to public;
		grant create on schema public to public;
		CREATE SCHEMA IF NOT EXISTS accounts_api;
		ALTER USER postgres SET search_path = accounts_api, public;
		SET search_path = accounts_api, public;
		`)
	if err != nil {
		return handleContainerStartErr(ctx, errors.Wrapf(err, "failed to apply schema. session: %s, port: %s",
			pgContainer.SessionID(), mappedPort.Port()), pgContainer, t)
	}
	// add truncate tables func
	_, err = pdb.DBS().Writer.Exec(`
CREATE OR REPLACE FUNCTION truncate_tables() RETURNS void AS $$
DECLARE
    statements CURSOR FOR
        SELECT tablename FROM pg_tables
        WHERE schemaname = 'accounts_api' and tablename != 'migrations';
BEGIN
    FOR stmt IN statements LOOP
        EXECUTE 'TRUNCATE TABLE ' || quote_ident(stmt.tablename) || ' CASCADE;';
    END LOOP;
END;
$$ LANGUAGE plpgsql;
`)
	if err != nil {
		return handleContainerStartErr(ctx, errors.Wrap(err, "failed to create truncate func"), pgContainer, t)
	}

	goose.SetTableName("accounts_api.migrations")
	if err := goose.RunContext(ctx, "up", pdb.DBS().Writer.DB, migrationsDirRelPath); err != nil {
		return handleContainerStartErr(ctx, errors.Wrap(err, "failed to apply goose migrations for test"), pgContainer, t)
	}

	return pdb, pgContainer
}

// StartContainerDex starts postgres container with default test settings. Caller must terminate container.
func StartContainerDex(ctx context.Context, t *testing.T) testcontainers.Container {
	dexPort := "5556"
	wd, err := os.Getwd()
	basepath := strings.Replace(wd, "/controller", "", 1)
	dexCr := testcontainers.ContainerRequest{
		Image:        "dexidp/dex",
		Cmd:          []string{"dex", "serve", "/config.docker.yaml"},
		ExposedPorts: []string{fmt.Sprintf("%s/tcp", dexPort)},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      filepath.Join(basepath, "/test/config.docker.yaml"),
				ContainerFilePath: "/config.docker.yaml",
				FileMode:          0o755,
			},
			{
				HostFilePath:      filepath.Join(basepath, "/test/dex.db"),
				ContainerFilePath: "/dex.db",
				FileMode:          0o755,
			},
		},
	}

	dexContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: dexCr,
		Started:          true,
	})
	if err != nil {
		if dexContainer != nil {
			dexContainer.Terminate(ctx) //nolint
		}
		t.Fatal(err)
	}
	mappedPort, err := dexContainer.MappedPort(ctx, nat.Port(dexPort))
	if err != nil {
		if err != nil {
			if dexContainer != nil {
				dexContainer.Terminate(ctx) //nolint
			}
			t.Fatal(err)
		}
	}
	fmt.Printf("dex container session %s ready and running at port: %s \n", dexContainer.SessionID(), mappedPort)

	return dexContainer
}

func GetContainerAddress(tc testcontainers.Container) (string, error) {
	mappedPort, err := tc.MappedPort(context.Background(), nat.Port("5556/tcp"))
	if err != nil {
		return "", err
	}

	host, err := tc.Host(context.Background())
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("http://%s:%s", host, mappedPort.Port()), nil
}

func handleContainerStartErr(ctx context.Context, err error, container testcontainers.Container, t *testing.T) (db.Store, testcontainers.Container) {
	if err != nil {
		if container != nil {
			container.Terminate(ctx) //nolint
		}
		t.Fatal(err)
	}
	return db.Store{}, container
}

// getTestDbSettings builds test db config.settings object
func getTestDbSettings() config.Settings {
	dbSettings := db.Settings{
		Name:               testDbName,
		Host:               "localhost",
		Port:               "6669",
		User:               "postgres",
		Password:           "postgres",
		MaxOpenConnections: 2,
		MaxIdleConnections: 2,
	}
	settings := config.Settings{
		LogLevel:    "info",
		DB:          dbSettings,
		ServiceName: "accounts-api",
	}
	return settings
}

// SetupAppFiber sets up app fiber with defaults for testing, like our production error handler.
func SetupAppFiber(logger zerolog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// copied from controllers.helpers.ErrorHandler - but temporarily in here to see if resolved circular deps issue
			code := fiber.StatusInternalServerError // Default 500 statuscode

			e, fiberTypeErr := err.(*fiber.Error)
			if fiberTypeErr {
				// Override status code if fiber.Error type
				code = e.Code
			}
			logger.Err(err).Str("httpStatusCode", strconv.Itoa(code)).
				Str("httpMethod", c.Method()).
				Str("httpPath", c.Path()).
				Msg("caught an error from http request")

			return c.Status(code).JSON(fiber.Map{
				"code":    code,
				"message": err.Error(),
			})
		},
	})
	return app
}

func BuildRequest(method, url, body, header string) *http.Request {
	req, _ := http.NewRequest(
		method,
		url,
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+header)

	return req
}

// AuthInjectorTestHandler injects fake jwt with sub
func EmailBasedAuthInjector(dexID, email string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		emailToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI0OTU1Y2FjMDA3Mjc5ODQzMGM3OTliNTE3ZDA1NzhhYjQ3NTBjNTMifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJnb29nbGUiLCJzdWIiOiJDZzB3TFRNNE5TMHlPREE0T1Mwd0VnWm5iMjluYkdVIiwiYXVkIjoiZXhhbXBsZS1hcHAiLCJleHAiOjE5MzM2ODgxMjEsImlhdCI6MTcxNzY4ODEyMSwiYXRfaGFzaCI6Ild5RjhCcm8zNWxKUnIzSjdTTHJoa3ciLCJlbWFpbCI6ImtpbGdvcmVAa2lsZ29yZS50cm91dCIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJuYW1lIjoiS2lsZ29yZSBUcm91dCJ9.Vie9vL3o8duL2XSv4q9kBISuFD2N-MGrKDGpHObD47JpEFzaT5RI2dv9EY6ckOHIbggqFIOfpBuK30J0bgBOnZXJFg_nxekZGKkBaBHg6_y6cKDX4Mw9zzTU_zu3Wc-NgEJ1JZJWR2r7AHv_FxvyRDj6BuC3akfUli4ApA_lSdl4VL-2z4yocKNxHWxdEJBp4LOSOix-lfQKseHaHqmA4b3SAgwL_LcoW3-4wkK0dtW5Uzk_Bo64DTMAiQ239vMa_JMclt9R1X4s-0NOOcIhXPmYxDDS9l8J0u1_p_DRuAhkn3nFdXtQ0MhYFhQWBb9hVPINBEZsupIEyM-dpe-iOA"
		c.Locals("user", emailToken)
		return c.Next()
	}
}

func WalletBasedAuthInjector(dexID string, ethAddr common.Address) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"provider_id":      "web3",
			"sub":              ksuid.New().String(),
			"ethereum_address": ethAddr.Hex(),
		})

		c.Locals("user", token)
		return c.Next()
	}
}

// TruncateTables truncates tables for the test db, useful to run as teardown at end of each DB dependent test.
func TruncateTables(db *sql.DB, t *testing.T) {
	_, err := db.Exec(`SELECT truncate_tables();`)
	if err != nil {
		fmt.Println("truncating tables failed.")
		t.Fatal(err)
	}
}

/** Test Setup functions. At some point may want to move elsewhere more generic **/

func Logger() *zerolog.Logger {
	l := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "accounts-api").
		Logger()
	return &l
}

func GenerateWallet() (*ecdsa.PrivateKey, *common.Address, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	userAddr := crypto.PubkeyToAddress(privateKey.PublicKey)

	return privateKey, &userAddr, nil
}

type eventService struct{}

func (e *eventService) Emit(event *services.Event) error {
	return nil
}

func NewEventService() services.EventService {
	return &eventService{}
}

type identityService struct {
}

var IdentityServiceResponse bool = true

func (i *identityService) VehiclesOwned(ctx context.Context, ethAddr common.Address) (bool, error) {
	return IdentityServiceResponse, nil
}

func (i *identityService) AftermarketDevicesOwned(ctx context.Context, ethAddr common.Address) (bool, error) {
	return IdentityServiceResponse, nil
}

func NewIdentityService(pass bool) services.IdentityService {
	return &identityService{}
}

type emailService struct {
}

func NewEmailService() services.EmailService {
	return &emailService{}
}

func (e *emailService) SendConfirmationEmail(ctx context.Context, emailTemplate *template.Template, userEmail, confCode string) error {
	return nil
}

func NewAccount(exec boil.ContextExecutor) (*models.Account, error) {
	acct := models.Account{
		ID:           ksuid.New().String(),
		DexID:        ksuid.New().String(),
		ReferralCode: null.StringFrom("ABCDEFGHIJKL"),
	}

	eml := models.Email{
		AccountID:    acct.ID,
		DexID:        acct.DexID,
		EmailAddress: "testemail@gmail.com",
		Confirmed:    true,
	}

	wallet := models.Wallet{
		AccountID:       acct.ID,
		DexID:           acct.DexID,
		EthereumAddress: common.HexToAddress("0x1234567890123456789012345678901234567890").Bytes(),
		Confirmed:       true,
	}

	if err := acct.Insert(context.Background(), exec, boil.Infer()); err != nil {
		return nil, err
	}

	if err := eml.Insert(context.Background(), exec, boil.Infer()); err != nil {
		return nil, err
	}

	if err := wallet.Insert(context.Background(), exec, boil.Infer()); err != nil {
		return nil, err
	}

	return models.Accounts(
		models.AccountWhere.ID.EQ(acct.ID),
		qm.Load(models.AccountRels.Wallet),
		qm.Load(models.AccountRels.Email),
	).One(context.Background(), exec)
}

func DeleteAll(exec boil.ContextExecutor) error {
	_, err := exec.Exec(`TRUNCATE TABLE accounts_api.accounts CASCADE;`)
	return err
}
