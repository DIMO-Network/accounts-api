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
	"time"

	"github.com/DIMO-Network/shared/db"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/ethereum/go-ethereum/common"
	jwtware "github.com/gofiber/contrib/jwt"
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
	settings        *config.Settings
	pdb             db.Store
	pgContainer     testcontainers.Container
	dexContainer    testcontainers.Container
	ctx             context.Context
	controller      *Controller
	eventService    services.EventService
	identityService services.IdentityService
	emailService    services.EmailService
}

// SetupSuite starts container db
func (s *AccountControllerTestSuite) SetupSuite() {
	s.app = fiber.New()
	s.ctx = context.Background()
	s.eventService = test.NewEventService()
	s.emailService = test.NewEmailService()
	s.identityService = test.NewIdentityService(true)
	s.pdb, s.pgContainer = test.StartContainerDatabase(s.ctx, s.T(), migrationsDirRelPath)
	s.dexContainer = test.StartContainerDex(s.ctx, s.T())
	time.Sleep(5 * time.Second) // TODOAE: need to add wait for log w regex to container req
	addr, err := test.GetContainerAddress(s.dexContainer)
	s.Require().NoError(err)

	s.settings = &config.Settings{
		JWTKeySetURL: fmt.Sprintf("%s/dex/keys", addr),
	}
	s.app.Use(jwtware.New(jwtware.Config{
		JWKSetURLs: []string{s.settings.JWTKeySetURL},
	}))

	s.controller = s.RecreateAccountController()
}

func (s *AccountControllerTestSuite) RecreateAccountController() *Controller {
	acctCont, err := NewAccountController(s.ctx, s.pdb, s.eventService, s.identityService, s.emailService, s.settings, test.Logger())
	s.Require().NoError(err)
	return acctCont
}

// TearDownSuite cleanup at end by terminating containers
func (s *AccountControllerTestSuite) TearDownSuite() {
	fmt.Printf("shutting down dex container with session: %s \n", s.dexContainer.SessionID())
	test.TruncateTables(s.pdb.DBS().Writer.DB, s.T())
	if err := s.dexContainer.Terminate(s.ctx); err != nil {
		s.T().Fatal(err)
	}

	fmt.Printf("shutting down postgres container with session: %s \n", s.pgContainer.SessionID())
	test.TruncateTables(s.pdb.DBS().Writer.DB, s.T())
	if err := s.pgContainer.Terminate(s.ctx); err != nil {
		s.T().Fatal(err)
	}

	s.app.Shutdown()
}

func TestDevicesControllerTestSuite(t *testing.T) {
	suite.Run(t, new(AccountControllerTestSuite))
}

func (s *AccountControllerTestSuite) Test_EmailFirstAccount_CreateAndDelete() {
	userEmail := "test_email@gmail.com"
	dexID := ksuid.New().String()
	userAuth := test.EmailBasedAuthInjector(dexID, userEmail)
	s.app.Get("/", userAuth, s.controller.GetOrCreateUserAccount)
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

	// Delete Account
	s.identityService = test.NewIdentityService(false) // need to reset mock to return false
	s.controller = s.RecreateAccountController()
	deleteReq := test.BuildRequest("DELETE", "/", "")
	deleteResp, _ := s.app.Test(deleteReq)
	_, err = io.ReadAll(deleteReq.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, deleteResp.StatusCode)

}

// func (s *AccountControllerTestSuite) Test_EmailFirstAccount_LinkWallet() {
// 	acct := models.Account{
// 		ID:    ksuid.New().String(),
// 		DexID: ksuid.New().String(),
// 	}

// 	eml := models.Email{
// 		AccountID:    acct.ID,
// 		DexID:        acct.DexID,
// 		EmailAddress: "test_email@gmail.com",
// 		Confirmed:    true,
// 	}

// 	if err := acct.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
// 		s.T().Fatal(err)
// 	}

// 	if err := eml.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
// 		s.T().Fatal(err)
// 	}

// 	wallet := "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"
// 	userAuth := test.EmailBasedAuthInjector(acct.DexID, eml.EmailAddress)
// 	s.app.Get("/", userAuth, s.controller.GetOrCreateUserAccount)
// 	s.app.Post("/link/wallet/token", userAuth, s.controller.LinkWalletToken)

// 	// Link Wallet
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
// 		"provider_id":      "web3",
// 		"sub":              acct.DexID,
// 		"ethereum_address": wallet,
// 	})

// 	tokenString, err := token.SignedString(secretKey)
// 	s.Require().NoError(err)

// 	var tb TokenBody
// 	tb.Token = tokenString
// 	b, _ := json.Marshal(tb)

// 	putReq := test.BuildRequest("POST", "/link/wallet/token", string(b))
// 	putResp, _ := s.app.Test(putReq)
// 	_, err = io.ReadAll(putResp.Body)
// 	s.Require().NoError(err)
// 	s.Assert().Equal(204, putResp.StatusCode)

// 	check := test.BuildRequest("GET", "/", "")
// 	checkResp, _ := s.app.Test(check)
// 	body, err := io.ReadAll(checkResp.Body)
// 	s.Require().NoError(err)
// 	s.Assert().Equal(200, checkResp.StatusCode)

// 	var resp UserResponse
// 	if err := json.Unmarshal(body, &check); err != nil {
// 		s.Require().NoError(err)
// 	}

// 	s.Assert().Equal(eml.EmailAddress, resp.Email.Address)
// 	s.Assert().NotNil(resp.Web3)
// 	s.Assert().Equal(resp.Web3.Address.Hex(), wallet)
// }

func (s *AccountControllerTestSuite) Test_WalletFirstAccount_CreateAndDelete() {
	userWallet := "test_email@gmail.com"
	userEmail := "0x71C7656EC7ab88b098defB751B7401B5f6d8976F"
	dexID := ksuid.New().String()
	userAuth := test.WalletBasedAuthInjector(dexID, common.HexToAddress(userWallet))
	s.app.Get("/", userAuth, s.controller.GetOrCreateUserAccount)
	s.app.Post("/link/email/token", userAuth, s.controller.LinkWalletToken)
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

	// Delete Account
	s.identityService = test.NewIdentityService(false) // need to reset mock to return false
	s.controller = s.RecreateAccountController()
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
	bodyToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjcwNTliYTk1MjZhYzhlMTcxOTJmZGJkOGRjYzk0NDEyNzkzMGEyOTEifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJtb2NrIiwic3ViIjoiQ2cwd0xUTTROUzB5T0RBNE9TMHdFZ1J0YjJOciIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxNzE3NjM0MjYzLCJpYXQiOjE3MTc1OTEwNjMsImF0X2hhc2giOiJZV3RPQ2RuU1UzQmVuNFNFWV9pVTNRIiwiZW1haWwiOiJraWxnb3JlQGtpbGdvcmUudHJvdXQiLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IktpbGdvcmUgVHJvdXQifQ.p9UiAr-4ed7wxaF5vMbAjoMFbUb9fJsJGPl8nP7m2N_2omq1jcy7ooR1MftzlaKzqAZK4oYiD25-ut3JIi3--5X9oEGsMwgs3Fx6JtN7Jep2ephnCyJEs3kmD_XKRwXynwK9Y4TE8BA-R_Vv0Cu_Z1_KYBWUpaxXiNXNBzaV5z3jKtGTy7GYkkZW6PFFD0L9m9i_R74hw2ajuhiTboEvm-KMTP5z1crfsXAIvRFJTCZ4Gn-fdN9EX1DiqLdawGMG-3Spwq7KyoBbsJ3pqP6lp3GO7f1ZhZvMKopv9wbXScSzFbxzpb6Dbbvkny1GFx6b3nX4m52GWrSJvJHWSxh6zg"
	jwkResource, err := keyfunc.NewDefaultCtx(context.Background(), []string{s.settings.JWTKeySetURL})
	s.Require().NoError(err)

	tbClaims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(bodyToken, &tbClaims, jwkResource.Keyfunc)
	s.Require().NoError(err)

	claims := getUserAccountInfos(tbClaims)

	s.Require().Equal(claims.DexID, "Cg0wLTM4NS0yODA4OS0wEgRtb2Nr")
	s.Require().Equal(claims.EmailAddress, "kilgore@kilgore.trout")
	s.Require().Equal(claims.ProviderID, "mock")

}
