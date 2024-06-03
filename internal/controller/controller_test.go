package controller

import (
	"accounts-api/internal/config"
	"accounts-api/internal/services"
	"accounts-api/internal/test"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/DIMO-Network/shared/db"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

const migrationsDirRelPath = "../../migrations"

var secretKey = []byte("secret-key")

type AccountControllerTestSuite struct {
	suite.Suite
	app             *fiber.App
	pdb             db.Store
	container       testcontainers.Container
	ctx             context.Context
	controller      Controller
	eventService    services.EventService
	identityService services.IdentityService
}

// SetupSuite starts container db
func (s *AccountControllerTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.pdb, s.container = test.StartContainerDatabase(s.ctx, s.T(), migrationsDirRelPath)
	s.eventService = test.NewEventService()
	s.identityService = test.NewIdentityService(&config.Settings{}, true)
	s.app = fiber.New()

	acctCont, err := NewAccountController(&config.Settings{}, s.pdb, s.eventService, s.identityService, test.Logger())
	s.Require().NoError(err)
	s.controller = *acctCont
}

// TearDownSuite cleanup at end by terminating container
func (s *AccountControllerTestSuite) TearDownSuite() {
	fmt.Printf("shutting down postgres at with session: %s \n", s.container.SessionID())
	test.TruncateTables(s.pdb.DBS().Writer.DB, s.T())
	if err := s.container.Terminate(s.ctx); err != nil {
		s.T().Fatal(err)
	}
	s.app.Shutdown()
}

func TestDevicesControllerTestSuite(t *testing.T) {
	suite.Run(t, new(AccountControllerTestSuite))
}

func (s *AccountControllerTestSuite) Test_CRUD_EmailBasedAccount() {
	userEmail := "test_email@gmail.com"
	wallet := "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"
	dexID := ksuid.New().String()
	userAuth := test.EmailBasedAuthInjector(dexID, userEmail)
	s.app.Get("/", userAuth, s.controller.GetOrCreateUserAccount)
	s.app.Post("/link/wallet/token", userAuth, s.controller.LinkWalletToken)
	s.app.Delete("/", userAuth, s.controller.DeleteUser)

	// Get Request Create Account
	getReq := test.BuildRequest("GET", "/", "")
	getResp, _ := s.app.Test(getReq)
	getBody, err := io.ReadAll(getResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, getResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(getBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().Equal(userEmail, userResp.Email.Address)
	s.Assert().Nil(userResp.Web3)

	// Link Wallet
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"provider_id":      "web3",
		"sub":              dexID,
		"ethereum_address": wallet,
	})

	tokenString, err := token.SignedString(secretKey)
	s.Require().NoError(err)

	var tb TokenBody
	tb.Token = tokenString
	b, _ := json.Marshal(tb)

	putReq := test.BuildRequest("POST", "/link/wallet/token", string(b))
	putResp, _ := s.app.Test(putReq)
	_, err = io.ReadAll(putResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, putResp.StatusCode)

	finalCheckReq := test.BuildRequest("GET", "/", "")
	finalCheckResp, _ := s.app.Test(finalCheckReq)
	body, err := io.ReadAll(finalCheckResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, finalCheckResp.StatusCode)

	var check UserResponse
	if err := json.Unmarshal(body, &check); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().Equal(userEmail, check.Email.Address)
	s.Assert().NotNil(check.Web3)

	s.identityService = test.NewIdentityService(&config.Settings{}, false)
	ctrl, _ := NewAccountController(&config.Settings{}, s.pdb, s.eventService, s.identityService, test.Logger())
	s.controller = *ctrl
	deleteReq := test.BuildRequest("DELETE", "/", "")
	deleteResp, _ := s.app.Test(deleteReq)
	_, err = io.ReadAll(deleteReq.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, deleteResp.StatusCode)

}

func (s *AccountControllerTestSuite) Test_GenerateReferralCode() {
	numUniqueCodes := 100
	uniqueCodes := make(map[string]interface{})
	for i := 0; i < numUniqueCodes; i++ {
		refCode, err := s.controller.GenerateReferralCode(context.Background())
		s.Require().NoError(err)
		uniqueCodes[refCode] = nil
	}

	s.Assert().Equal(numUniqueCodes, len(uniqueCodes))
}

func (s *AccountControllerTestSuite) Test_JWTDecode() {

	bodyToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjVmNjRiYzQwZjMwYmIzMWY0NGQzOGM3MzhiZTc3OTEzYTM5Yzc3OTQifQ.eyJpc3MiOiJodHRwczovL2F1dGguZGV2LmRpbW8uem9uZSIsInByb3ZpZGVyX2lkIjoiZ29vZ2xlIiwic3ViIjoiQ2hVeE1ETTNOelUxTWpFNE9URXdOemswTVRJeE5EY1NCbWR2YjJkc1pRIiwiYXVkIjoiZGltby1kcml2ZXIiLCJleHAiOjE3MTgzMDQxODMsImlhdCI6MTcxNzA5NDU4Mywibm9uY2UiOiIxWEpZcWNsbTV4bUQ3NWM3WVNmSFl2aU5sdGtoaElTU1RjcUNYU0tCcHNZIiwiYXRfaGFzaCI6Im5NaHlVOEItc0twOGR5RWJNN0JCRGciLCJjX2hhc2giOiJQaDdKSERoZ1dSMXJfeGVTSDR6UmlnIiwiZW1haWwiOiJhbGx5c29uLmVuZ2xpc2hAZ21haWwuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWV9.TeI8a4cetrt52tXnvBHVqSm9cNGsKUEDqYs6HZCEQIH4RtsroZNvBc5fpIQeFKoUxzUBFH64U_geAyDgOC5zabMMBo8oeRQLj_KNwIrsrUYukHf79VCYH89J1nShMvYWuJISjw9bmnndK5GD5KKcCGXhW8qUDUqJNBTk0hI76FkBp7jx1yma3_qIcApyI7bnhxgCJhrrTZ41Y3aByZOnOXYyt-4uu7WM545Jnz9MChu27bZGA_O0RBvSObJ_M1pb7nI10bUH2DRXwo1-7BurPF-clewr4riOxv9jGFzJyVgPvpQN2vyecWWkRVqxHEB672EEQBX0M-pe-HajLYGmKw"

	p := jwt.NewParser()
	clm := jwt.MapClaims{}
	_, _, _ = p.ParseUnverified(bodyToken, &clm)

	claims := getUserAccountInfos(clm)

	s.Assert().Equal(claims.DexID, "ChUxMDM3NzU1MjE4OTEwNzk0MTIxNDcSBmdvb2dsZQ")
	s.Assert().Equal(claims.EmailAddress, "allyson.english@gmail.com")
	s.Assert().Equal(claims.ProviderID, "google")

}
