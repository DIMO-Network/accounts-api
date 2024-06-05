package test

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"accounts-api/internal/config"
	"accounts-api/internal/services"

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
	dexCr := testcontainers.ContainerRequest{
		Image:        "dexidp/dex",
		Cmd:          []string{"dex", "serve", "/config.docker.yaml"},
		ExposedPorts: []string{fmt.Sprintf("%s:%s/tcp", dexPort, dexPort)},
		Mounts: testcontainers.ContainerMounts{
			{
				Source: testcontainers.GenericVolumeMountSource{
					Name: "./config.docker.yaml",
				},
				Target: "/config.docker.yaml",
			},
			{
				Source: testcontainers.GenericVolumeMountSource{
					Name: "./dex.db",
				},
				Target: "/dex.db",
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

func BuildRequest(method, url, body string) *http.Request {
	req, _ := http.NewRequest(
		method,
		url,
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	return req
}

// AuthInjectorTestHandler injects fake jwt with sub
func EmailBasedAuthInjector(dexID, email string) fiber.Handler {
	// provider, err := oidc.NewProvider(context.Background(), "http://127.0.0.1:5556/dex/keys")
	// provider.
	return func(c *fiber.Ctx) error {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"provider_id": "google",
			"sub":         dexID,
			"email":       email,
		})

		c.Locals("user", token)
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
	Pass bool
}

func (i *identityService) VehiclesOwned(ctx context.Context, ethAddr common.Address) (bool, error) {
	return i.Pass, nil
}

func (i *identityService) AftermarketDevicesOwned(ctx context.Context, ethAddr common.Address) (bool, error) {
	return i.Pass, nil
}

func NewIdentityService(pass bool) services.IdentityService {
	return &identityService{Pass: pass}
}

type emailService struct {
}

func NewEmailService() services.EmailService {
	return &emailService{}
}

func (e *emailService) SendConfirmationEmail(ctx context.Context, emailTemplate *template.Template, userEmail, confCode string) error {
	return nil
}
