package controller

import (
	"accounts-api/internal/config"
	"accounts-api/internal/services"
	"accounts-api/internal/test"
	"accounts-api/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
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
	"github.com/volatiletech/sqlboiler/v4/boil"
)

var (
	secretKey            = []byte("secret-key")
	migrationsDirRelPath = "../../migrations"
	dexIDEmail           = "Cg0wLTM4NS0yODA4OS0wEgZnb29nbGU"
	dexIDWallet          = "CioweGYzOUZkNmU1MWFhZDg4RjZGNGNlNmFCODgyNzI3OWNmZkZiOTIyNjYSBHdlYjM"
	userWallet           = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	userEmail            = "kilgore@kilgore.trout"
	emailProvider        = "google"
	emailBasedAuthToken  = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI0OTU1Y2FjMDA3Mjc5ODQzMGM3OTliNTE3ZDA1NzhhYjQ3NTBjNTMifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJnb29nbGUiLCJzdWIiOiJDZzB3TFRNNE5TMHlPREE0T1Mwd0VnWm5iMjluYkdVIiwiYXVkIjoiZXhhbXBsZS1hcHAiLCJleHAiOjE5MzM2ODgxMjEsImlhdCI6MTcxNzY4ODEyMSwiYXRfaGFzaCI6Ild5RjhCcm8zNWxKUnIzSjdTTHJoa3ciLCJlbWFpbCI6ImtpbGdvcmVAa2lsZ29yZS50cm91dCIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJuYW1lIjoiS2lsZ29yZSBUcm91dCJ9.Vie9vL3o8duL2XSv4q9kBISuFD2N-MGrKDGpHObD47JpEFzaT5RI2dv9EY6ckOHIbggqFIOfpBuK30J0bgBOnZXJFg_nxekZGKkBaBHg6_y6cKDX4Mw9zzTU_zu3Wc-NgEJ1JZJWR2r7AHv_FxvyRDj6BuC3akfUli4ApA_lSdl4VL-2z4yocKNxHWxdEJBp4LOSOix-lfQKseHaHqmA4b3SAgwL_LcoW3-4wkK0dtW5Uzk_Bo64DTMAiQ239vMa_JMclt9R1X4s-0NOOcIhXPmYxDDS9l8J0u1_p_DRuAhkn3nFdXtQ0MhYFhQWBb9hVPINBEZsupIEyM-dpe-iOA"
	walletBasedAuthToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI0OTU1Y2FjMDA3Mjc5ODQzMGM3OTliNTE3ZDA1NzhhYjQ3NTBjNTMifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJ3ZWIzIiwic3ViIjoiQ2lvd2VHWXpPVVprTm1VMU1XRmhaRGc0UmpaR05HTmxObUZDT0RneU56STNPV05tWmtaaU9USXlOallTQkhkbFlqTSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxOTMzNjg5MDAyLCJpYXQiOjE3MTc2ODkwMDIsImF0X2hhc2giOiJtR2NsQ3Y3ekR4Z1FZWGtVaDFwdV9RIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJldGhlcmV1bV9hZGRyZXNzIjoiMHhmMzlGZDZlNTFhYWQ4OEY2RjRjZTZhQjg4MjcyNzljZmZGYjkyMjY2IiwibmFtZSI6IjB4ZjM5RmQ2ZTUxYWFkODhGNkY0Y2U2YUI4ODI3Mjc5Y2ZmRmI5MjI2NiJ9.lYk7Odm51_tbEBfDkLwKeZUGWN_T8FwezIipxFWENRmFgzxFeX9KK0qm0eEpBoEEXcsaDPA52eYnVd-XTOq5ymBJW3eEcwr3EOkUsv-FlwD9fZXVD9X62-HXLNkA0lpxxY9WLc3JjgSM2Gnn2001Pmj5uNcYfGH2m6EBXli7EJufOHb8_D6S3lQZNpK_bNidOvu6pP1v3F3hFZWJl0aG4VMoJFPuJo3Wv1lToq4Jc9ZuxUJ7VqYRNPRCPFRFUWel6in4Rf1GpT1X-GyuILDiwn_MOwpc1ndw2YvabfTcZsiCflmtB1DsEC6bJXhFbEE7pUao_PmhpgBcISbDhsETqw"
)

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

	acctCont, err := NewAccountController(s.ctx, s.pdb, s.eventService, s.identityService, s.emailService, s.settings, test.Logger())
	s.controller = acctCont
	s.app.Get("/", s.controller.GetOrCreateUserAccount)
	s.app.Put("/", s.controller.UpdateUser)
	s.app.Delete("/", s.controller.DeleteUser)

	s.app.Post("/agree-tos", s.controller.AgreeTOS)
	s.app.Post("/referral/submit", s.controller.SubmitReferralCode)
	s.app.Post("/link/wallet/token", s.controller.LinkWalletToken)
	s.app.Post("/link/email/token", s.controller.LinkEmailToken)
	s.app.Post("/link/email", s.controller.LinkEmail)
	s.app.Post("/link/email/confirm", s.controller.ConfirmEmail)

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
	// Create Account
	getReq := test.BuildRequest("GET", "/", "", emailBasedAuthToken)
	getResp, _ := s.app.Test(getReq)
	getBody, err := io.ReadAll(getResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, getResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(getBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().Equal(userEmail, userResp.Email.Address)
	s.Assert().True(userResp.Email.Confirmed)
	s.Assert().Nil(userResp.Web3)

	// Delete Account
	deleteReq := test.BuildRequest("DELETE", "/", "", emailBasedAuthToken)
	deleteResp, err := s.app.Test(deleteReq)
	s.Require().NoError(err)
	_, err = io.ReadAll(deleteReq.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, deleteResp.StatusCode)

}

func (s *AccountControllerTestSuite) Test_EmailFirstAccount_UpdateAccount() {
	acct := models.Account{
		ID:    ksuid.New().String(),
		DexID: dexIDEmail,
	}

	eml := models.Email{
		AccountID:    acct.ID,
		DexID:        dexIDEmail,
		EmailAddress: userEmail,
		Confirmed:    true,
	}

	if err := acct.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	if err := eml.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	updateBody := UserUpdateRequest{
		CountryCode: "USA",
	}
	updateBodyBytes, _ := json.Marshal(updateBody)

	putReq := test.BuildRequest("PUT", "/", string(updateBodyBytes), emailBasedAuthToken)
	putResp, _ := s.app.Test(putReq)
	putBody, err := io.ReadAll(putResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, putResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(putBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().NotNil(userResp.Email)
	s.Assert().Equal(userEmail, userResp.Email.Address)
	s.Assert().Equal(updateBody.CountryCode, userResp.CountryCode)
	test.DeleteAll(s.pdb.DBS().Writer)

}

func (s *AccountControllerTestSuite) Test_EmailFirstAccount_AgreeTOS() {
	acct := models.Account{
		ID:    ksuid.New().String(),
		DexID: dexIDEmail,
	}

	eml := models.Email{
		AccountID:    acct.ID,
		DexID:        dexIDEmail,
		EmailAddress: userEmail,
		Confirmed:    true,
	}

	if err := acct.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	if err := eml.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	putReq := test.BuildRequest("POST", "/agree-tos", "", emailBasedAuthToken)
	putResp, _ := s.app.Test(putReq)
	_, err := io.ReadAll(putResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, putResp.StatusCode)
	test.DeleteAll(s.pdb.DBS().Writer)
}

func (s *AccountControllerTestSuite) Test_EmailFirstAccount_LinkWallet() {
	acct := models.Account{
		ID:    ksuid.New().String(),
		DexID: dexIDEmail,
	}

	eml := models.Email{
		AccountID:    acct.ID,
		DexID:        dexIDEmail,
		EmailAddress: userEmail,
		Confirmed:    true,
	}

	if err := acct.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	if err := eml.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	linkWalletBody := TokenBody{
		Token: walletBasedAuthToken,
	}
	linkWalletBodyBytes, _ := json.Marshal(linkWalletBody)

	putReq := test.BuildRequest("POST", "/link/wallet/token", string(linkWalletBodyBytes), emailBasedAuthToken)
	putResp, _ := s.app.Test(putReq)
	_, err := io.ReadAll(putResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, putResp.StatusCode)

	getReq := test.BuildRequest("GET", "/", "", emailBasedAuthToken)
	getResp, _ := s.app.Test(getReq)
	getBody, err := io.ReadAll(getResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, getResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(getBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().NotNil(userResp.Email)
	s.Assert().Equal(userEmail, userResp.Email.Address)
	s.Assert().NotNil(userResp.Web3)
	s.Assert().Equal(userResp.Web3.Address.Hex(), userWallet)
	test.DeleteAll(s.pdb.DBS().Writer)
}

func (s *AccountControllerTestSuite) Test_WalletFirstAccount_CreateAndDelete() {
	// Create Account
	getReq := test.BuildRequest("GET", "/", "", walletBasedAuthToken)
	getResp, _ := s.app.Test(getReq)
	getBody, err := io.ReadAll(getResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, getResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(getBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().Nil(userResp.Email)
	s.Assert().Equal(userWallet, userResp.Web3.Address.Hex())
	s.Assert().True(userResp.Web3.Confirmed)

	// Set identity svc to be consistent with eligible deletion state
	test.IdentityServiceResponse = false

	// Delete Account
	deleteReq := test.BuildRequest("DELETE", "/", "", walletBasedAuthToken)
	deleteResp, err := s.app.Test(deleteReq)
	s.Require().NoError(err)
	_, err = io.ReadAll(deleteReq.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, deleteResp.StatusCode)
	test.DeleteAll(s.pdb.DBS().Writer)

}

func (s *AccountControllerTestSuite) Test_WalletFirstAccount_LinkEmailToken() {
	acct := models.Account{
		ID:    ksuid.New().String(),
		DexID: dexIDWallet,
	}

	wallet := models.Wallet{
		AccountID:       acct.ID,
		DexID:           acct.DexID,
		EthereumAddress: common.Hex2Bytes(strings.Replace(userWallet, "0x", "", 1)),
		Confirmed:       true,
	}

	if err := acct.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	if err := wallet.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	linkEmailBody := TokenBody{
		Token: emailBasedAuthToken,
	}
	linkEmailBodyBytes, _ := json.Marshal(linkEmailBody)

	putReq := test.BuildRequest("POST", "/link/email/token", string(linkEmailBodyBytes), walletBasedAuthToken)
	putResp, _ := s.app.Test(putReq)
	_, err := io.ReadAll(putResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, putResp.StatusCode)

	getReq := test.BuildRequest("GET", "/", "", walletBasedAuthToken)
	getResp, _ := s.app.Test(getReq)
	getBody, err := io.ReadAll(getResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, getResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(getBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().NotNil(userResp.Email)
	s.Assert().Equal(userEmail, userResp.Email.Address)
	s.Assert().NotNil(userResp.Web3)
	s.Assert().Equal(userResp.Web3.Address.Hex(), userWallet)
	test.DeleteAll(s.pdb.DBS().Writer)
}

func (s *AccountControllerTestSuite) Test_WalletFirstAccount_LinkEmailConfirm() {
	acct := models.Account{
		ID:    ksuid.New().String(),
		DexID: dexIDWallet,
	}

	wallet := models.Wallet{
		AccountID:       acct.ID,
		DexID:           acct.DexID,
		EthereumAddress: common.Hex2Bytes(strings.Replace(userWallet, "0x", "", 1)),
		Confirmed:       true,
	}

	if err := acct.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	if err := wallet.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	putReq := test.BuildRequest("POST", "/link/email", "", walletBasedAuthToken)
	putResp, _ := s.app.Test(putReq)
	_, err := io.ReadAll(putResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, putResp.StatusCode)

	confirmReq := test.BuildRequest("POST", "/link/email/confirm", "", walletBasedAuthToken)
	confirmResp, _ := s.app.Test(confirmReq)
	_, err = io.ReadAll(confirmResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, putResp.StatusCode)
	test.DeleteAll(s.pdb.DBS().Writer)
}

func (s *AccountControllerTestSuite) Test_SubmitReferralCode() {
	refAcct, err := test.NewAccount(s.pdb.DBS().Writer) // Create Referrer Account
	s.Require().NoError(err)

	acct := models.Account{
		ID:    ksuid.New().String(),
		DexID: dexIDWallet,
	}

	wallet := models.Wallet{
		AccountID:       acct.ID,
		DexID:           acct.DexID,
		EthereumAddress: common.Hex2Bytes(strings.Replace(userWallet, "0x", "", 1)),
		Confirmed:       true,
	}

	if err := acct.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	if err := wallet.Insert(s.ctx, s.pdb.DBS().Writer, boil.Infer()); err != nil {
		s.T().Fatal(err)
	}

	referralCodeBody := SubmitReferralCodeRequest{
		ReferralCode: refAcct.ReferralCode.String,
	}
	referralCodeBodyBytes, _ := json.Marshal(referralCodeBody)

	// Set identity svc to be consistent with referral eligibility
	test.IdentityServiceResponse = false

	postReq := test.BuildRequest("POST", "/referral/submit", string(referralCodeBodyBytes), walletBasedAuthToken)
	postResp, _ := s.app.Test(postReq)
	_, err = io.ReadAll(postResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, postResp.StatusCode)
	test.DeleteAll(s.pdb.DBS().Writer)
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
	bodyToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI0OTU1Y2FjMDA3Mjc5ODQzMGM3OTliNTE3ZDA1NzhhYjQ3NTBjNTMifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJnb29nbGUiLCJzdWIiOiJDZzB3TFRNNE5TMHlPREE0T1Mwd0VnWm5iMjluYkdVIiwiYXVkIjoiZXhhbXBsZS1hcHAiLCJleHAiOjE5MzM2ODgxMjEsImlhdCI6MTcxNzY4ODEyMSwiYXRfaGFzaCI6Ild5RjhCcm8zNWxKUnIzSjdTTHJoa3ciLCJlbWFpbCI6ImtpbGdvcmVAa2lsZ29yZS50cm91dCIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJuYW1lIjoiS2lsZ29yZSBUcm91dCJ9.Vie9vL3o8duL2XSv4q9kBISuFD2N-MGrKDGpHObD47JpEFzaT5RI2dv9EY6ckOHIbggqFIOfpBuK30J0bgBOnZXJFg_nxekZGKkBaBHg6_y6cKDX4Mw9zzTU_zu3Wc-NgEJ1JZJWR2r7AHv_FxvyRDj6BuC3akfUli4ApA_lSdl4VL-2z4yocKNxHWxdEJBp4LOSOix-lfQKseHaHqmA4b3SAgwL_LcoW3-4wkK0dtW5Uzk_Bo64DTMAiQ239vMa_JMclt9R1X4s-0NOOcIhXPmYxDDS9l8J0u1_p_DRuAhkn3nFdXtQ0MhYFhQWBb9hVPINBEZsupIEyM-dpe-iOA"
	jwkResource, err := keyfunc.NewDefaultCtx(context.Background(), []string{s.settings.JWTKeySetURL})
	s.Require().NoError(err)

	tbClaims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(bodyToken, &tbClaims, jwkResource.Keyfunc)
	s.Require().NoError(err)

	claims := getUserAccountInfos(tbClaims)

	s.Require().Equal(claims.DexID, dexIDEmail)
	s.Require().Equal(claims.EmailAddress, userEmail)
	s.Require().Equal(claims.ProviderID, emailProvider)

}
