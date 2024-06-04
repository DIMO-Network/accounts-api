package controller

import (
	"accounts-api/internal/config"
	"accounts-api/internal/services"
	"accounts-api/internal/test"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"

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
	container       testcontainers.Container
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
	s.pdb, s.container = test.StartContainerDatabase(s.ctx, s.T(), migrationsDirRelPath)
	s.settings = &config.Settings{
		JWTKeySetURL: "http://127.0.0.1:5556/dex/keys",
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

	// finalCheckReq := test.BuildRequest("GET", "/", "")
	// finalCheckResp, _ := s.app.Test(finalCheckReq)
	// body, err := io.ReadAll(finalCheckResp.Body)
	// s.Require().NoError(err)
	// s.Assert().Equal(200, finalCheckResp.StatusCode)

	// var check UserResponse
	// if err := json.Unmarshal(body, &check); err != nil {
	// 	s.Require().NoError(err)
	// }

	// s.Assert().Equal(userEmail, check.Email.Address)
	// s.Assert().NotNil(check.Web3)

	// Delete Account
	s.identityService = test.NewIdentityService(false) // need to reset mock to return false
	s.controller = s.RecreateAccountController()
	deleteReq := test.BuildRequest("DELETE", "/", "")
	deleteResp, _ := s.app.Test(deleteReq)
	_, err = io.ReadAll(deleteReq.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, deleteResp.StatusCode)

}

func (s *AccountControllerTestSuite) Test_CRUD_WalletBasedAccount() {
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

	// Link Wallet
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"provider_id":   "web3",
		"sub":           dexID,
		"email_address": userEmail,
	})

	tokenString, err := token.SignedString(secretKey)
	s.Require().NoError(err)

	var tb TokenBody
	tb.Token = tokenString
	b, _ := json.Marshal(tb)

	putReq := test.BuildRequest("POST", "/link/email/token", string(b))
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

	bodyToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6ImMzMjNkZjkyMjY3ZTg5YzUyYjBlYjY5ZDE3Y2Y5MGU4NTdlOTczNGEifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJtb2NrIiwic3ViIjoiQ2cwd0xUTTROUzB5T0RBNE9TMHdFZ1J0YjJOciIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxNzE3NTcwMjg4LCJpYXQiOjE3MTc1MjcwODgsImF0X2hhc2giOiJZendVUnlqTnNVc2RuNm1NVVUyeGhRIiwiZW1haWwiOiJraWxnb3JlQGtpbGdvcmUudHJvdXQiLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IktpbGdvcmUgVHJvdXQifQ.Bk9PBrb3arTdBgLPWjzUh9Yu08LAwNp0-Ncwe3v9ZbWOWddWfL41DYkDYqoyo3nUx52rNTwX1GSHGoYHFwcL7IkKJlcmQif-sXQCvPksNyngw4uXeueff3bwxtEUg2MLjqHXBhgvUSC_bCbZ0ejSfuZAtJGVaqCuYf9pN6JBcb9qjRltBwbQkUFwChuZSYLB3RrLesvTY0MOWIGJMBiug8FP_rJ8sLBvQ2QuQZiCZVF3Epy25OIRv2DplLMxehbmAA2kHVlOA0CfmEtEymvh1Hf5lRlwq-7vWdsZ26j5Ui4pOo5zvO90jmeOKejc8jI4Ivrz411L_92apvpbsS9uAw"

	jwkResource, err := keyfunc.NewDefaultCtx(context.Background(), []string{"http://127.0.0.1:5556/dex/keys"}) // Context is used to end the refresh goroutine.
	if err != nil {
		log.Fatalf("Failed to create a keyfunc.Keyfunc from the server's URL.\nError: %s", err)
	}

	tbClaims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(bodyToken, &tbClaims, jwkResource.Keyfunc)
	s.Require().NoError(err)

	claims := getUserAccountInfos(tbClaims)

	s.Assert().Equal(claims.DexID, "Cg0wLTM4NS0yODA4OS0wEgRtb2Nr")
	s.Assert().Equal(claims.EmailAddress, "kilgore@kilgore.trout")
	s.Assert().Equal(claims.ProviderID, "mock")

}
