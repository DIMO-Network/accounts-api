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
	"testing"
	"time"

	"github.com/DIMO-Network/shared/db"
	"github.com/MicahParks/keyfunc/v3"
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

var (
	migrationsDirRelPath = "../../migrations"
)

var dexEmailUsers = []struct {
	Email     string
	AuthToken string
}{
	{
		Email:     "harry.jekyll@test.com",
		AuthToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI5NmVkODUwNjQ0MWI2ZGY0MWQ1NjA3OTFlYmU2MTI3MjJjNmQ1ZDUifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJnb29nbGUiLCJzdWIiOiJDaVF3T0dFNE5qZzBZaTFrWWpnNExUUmlOek10T1RCaE9TMHpZMlF4TmpZeFpqVTBOallTQm1kdmIyZHNaUSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxOTM0Mjg2MzE0LCJpYXQiOjE3MTgyODYzMTQsImF0X2hhc2giOiJyZXdIM1o3RFY5M2FvM2xRZWFVeml3IiwiZW1haWwiOiJoYXJyeS5qZWt5bGxAdGVzdC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6ImFkbWluIn0.kKzJNAZdMXbgnqZJZvUonF3P4ArTQ8eDtvm3a4VaQCaWCyEdmh8qYiJT5XzaqkUQTgse2kfi38R3OGSDU3ZDxToX9yfz1SGR5rQLi5CqkXgaKkdXK-nmlwvKSEdTqmR5LqOQGHVZYuyYK7H4KTI8hKbKy67dZKW5v6iwHX7y9s3D4sNnouW4nJfQGRSm1SlgVJIw6U_H_oFuFRGDnJkvsHI2bjZfdmzwNG-M4j8-lDJo2IphB4kxEICaw8IbH8lbKP0XflVbFeZGxC4GQcpVKxaXLmI1-dgwU2mDPPlfacRs2qTvHjUuPQdx7Mn1ual24aXc35DttfYcfJkyk-vcyg",
	},
	{
		Email:     "hari@seldon.com",
		AuthToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI5NmVkODUwNjQ0MWI2ZGY0MWQ1NjA3OTFlYmU2MTI3MjJjNmQ1ZDUifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJnb29nbGUiLCJzdWIiOiJDaVF3T0dFNE5qZzBZaTFrWWpnNExUUmlOek10T1RCaE9TMHpZMlF4TmpZeFpqVTBOallTQm1kdmIyZHNaUSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxOTM0Mjg5NjkyLCJpYXQiOjE3MTgyODk2OTIsImF0X2hhc2giOiJncHk4SUtXWWZVdTRYdkF6M05WUVl3IiwiZW1haWwiOiJoYXJpQHNlbGRvbi5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6ImFkbWluIn0.St590DD2pUQ7xG3T-RW6sts060se45hyOrsg-aHNdwUMhWExyjfdCnK0d7dNwApSCGhV-5KGm1Go6OkEhlOU9I0mnWmBXZCiclysiVzQiIdENPb1U-s49caAnGgBRZ-GpE28bll3QGOoVlfPmO8Zj4PuGvH_ylYiMaeJPW46F7-53LevjGVQeQdlXV69_N2_5i-64L3JNzTFfwu4CLWLk7UuknTrRiKFoAirP-hFYS_-tPg7Qm6PLxcfOsTxDLcnwqsSRdSiwNkgPOipyDsqZGVP0E0HR0qxl0Xp79kUOzKKI91dsmkD9x3RrqKlk7u4ce6h2WqeGf_2OIfZpzmw9A",
	},
	{
		Email:     "kilgore@kilgore.trout",
		AuthToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI5NmVkODUwNjQ0MWI2ZGY0MWQ1NjA3OTFlYmU2MTI3MjJjNmQ1ZDUifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJnb29nbGUiLCJzdWIiOiJDaVF3T0dFNE5qZzBZaTFrWWpnNExUUmlOek10T1RCaE9TMHpZMlF4TmpZeFpqVTBOallTQm1kdmIyZHNaUSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxOTM0Mjg2Njg1LCJpYXQiOjE3MTgyODY2ODUsImF0X2hhc2giOiJDNmtueW1zTF80Y0NsRGJsVXVXTFlBIiwiZW1haWwiOiJraWxnb3JlQGtpbGdvcmUudHJvdXQiLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6ImFkbWluIn0.MDKEOjp_KOvIFLYrzMCSSY1BOqK5nlIva3mBagJiMl7en2NzloqxmOAifY9H7qWT5nFiIEqQ2sF-Dws2RE2ggbhhEHqXL97xS47cJ9w3PJkbh4k9EwJogw5brc01bKj5r10i5nFtLRDDAnQBOK8wM3khlYxxNZT_Of8eaXyxJ4RCA18svM16k248Z3AVRh6D0V2_mahedLeclX9sDPNax87vLbzTj_RRGuEELNYfIj1cxJAQtmwJYY_j2VaKL8BsdhwqR1saIdvJGi4iMWwMlrfNimMZmWCQBXKJjeV9qvpn0FsYZUoKNS_i6s97POi_IZ8zFYKrE27et9L94LUm4A",
	},
	{
		Email:     "edward.hyde@example.com",
		AuthToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI5NmVkODUwNjQ0MWI2ZGY0MWQ1NjA3OTFlYmU2MTI3MjJjNmQ1ZDUifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJnb29nbGUiLCJzdWIiOiJDaVF3T0dFNE5qZzBZaTFrWWpnNExUUmlOek10T1RCaE9TMHpZMlF4TmpZeFpqVTBOallTQm1kdmIyZHNaUSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxOTM0Mjg2NzAzLCJpYXQiOjE3MTgyODY3MDMsImF0X2hhc2giOiJBdnZrUnBaVXRzU3JfbUlBa0Y3U0tBIiwiZW1haWwiOiJlZHdhcmQuaHlkZUBleGFtcGxlLmNvbSIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJuYW1lIjoiYWRtaW4ifQ.n0kB7BSn9I39EUxsbePFTZ7s4lNb-dUwzV_bRsFhwRpENUD_oabEWQaEl6rbUP4QF0ViY35L4d-wdWQPOCCfvzacu5PWqQynCKKcO9XZ9el5lVkKFAvMOvcyjXNcoKLr6Yo8ycQCsnDwZuyW7FM28BjIoZYvHco8hMlQHIUFJ6-pwL2CIequ4-8_gTgPoyPaHJDxMdsxKqKH3dJno7HsFPTp8CZfV7pxbav_q7mrSUWAdL1jjgnyJInuZIhqDtY0x-I6ZzRiyjrwGDHzx2J_E48JTlqwt_ZR3pBwY3E72qHYriIRYIzrrIU07aBMpnOCL-0HzwDvmxiwjggje5h0jA",
	},
}

var dexWalletUsers = []struct {
	Wallet    string
	AuthToken string
}{
	{
		Wallet:    "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		AuthToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI5NmVkODUwNjQ0MWI2ZGY0MWQ1NjA3OTFlYmU2MTI3MjJjNmQ1ZDUifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJ3ZWIzIiwic3ViIjoiQ2lvd2VHWXpPVVprTm1VMU1XRmhaRGc0UmpaR05HTmxObUZDT0RneU56STNPV05tWmtaaU9USXlOallTQkhkbFlqTSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxOTM0Mjg2ODEzLCJpYXQiOjE3MTgyODY4MTMsImF0X2hhc2giOiJMYmItd2Fnc2RZYzY4MF8weDlUUEpBIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJldGhlcmV1bV9hZGRyZXNzIjoiMHhmMzlGZDZlNTFhYWQ4OEY2RjRjZTZhQjg4MjcyNzljZmZGYjkyMjY2IiwibmFtZSI6IjB4ZjM5RmQ2ZTUxYWFkODhGNkY0Y2U2YUI4ODI3Mjc5Y2ZmRmI5MjI2NiJ9.MAIFC3eTbDJvCdDADqw05_nBru1p8gtubi49LUcLn36l8yxml89MbmeD9TVFv4nyfi3VtgvCUfaphS-8rP3tuIP32H-pXnCpqrU0EwUjFT6H6pNwMKQrptynER3xQbQFK-LOKoiRs0wuygObVlu9mpL5ygc1-tdEsUwi7yJlrCxBjFJN9sei3Gu4a2cHFQ4opuF1UMWCoyHWXJYiZJS8ykPNZIu0azbcwB48M98QKmL97R5-VIvq0h2oi-IaSEjMyS2xYa5Y2g9PWo4LGgzGJNs4GsM_TFI_ciwqw8mBmTBLEYrEKjVde_pP7YyC82lUwL9X1MDXaGoGTUvqYJHIcQ",
	},
	{
		Wallet:    "0x2BBB5d347D7F4a312199C30869253094499aB049",
		AuthToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI5NmVkODUwNjQ0MWI2ZGY0MWQ1NjA3OTFlYmU2MTI3MjJjNmQ1ZDUifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJ3ZWIzIiwic3ViIjoiQ2lvd2VESkNRa0kxWkRNME4wUTNSalJoTXpFeU1UazVRek13T0RZNU1qVXpNRGswTkRrNVlVSXdORGtTQkhkbFlqTSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxOTM0Mjg3MDAzLCJpYXQiOjE3MTgyODcwMDMsImF0X2hhc2giOiJ6NnJ4QzFEdTVwQ2ZXeUZIS08tVGZRIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJldGhlcmV1bV9hZGRyZXNzIjoiMHgyQkJCNWQzNDdEN0Y0YTMxMjE5OUMzMDg2OTI1MzA5NDQ5OWFCMDQ5IiwibmFtZSI6IjB4MkJCQjVkMzQ3RDdGNGEzMTIxOTlDMzA4NjkyNTMwOTQ0OTlhQjA0OSJ9.pXCVE_9cA46kJP87zWEHhDu-cqf-DN8GBRYBtrNeJ5GkxOHLYs9dkWGoul2_jEmH1F9YKI_C7r3vQdn7JI7cxs_Rcfx70TlbcQ18nErXzjoXE1r3lOHOmysgfONWWxgNgi2HfuF8nEGaMsPVxs9Fc7iIEhfDMWi3RZECxO2gzVtylRJ7t5PcThvGeg1EW4KgUHnq81AfJdhGU5XAgYc9eeVd0-mLnCnt4vkSj93T9F3Tth-6NhOf2l1SfuJQ9Lpi5vXKfHNTFYe0mld1UEBiY5zCM3v9iJw5DyVAMzmCd8ATHSwJWzKYWV0x9K_7JRoqYx6LqCxedcuOf6COtudXJw",
	},
	{
		Wallet:    "0xBe396b4B84a4139EA5C4dbaDde98AE8364eB21e8",
		AuthToken: "eyJhbGciOiJSUzI1NiIsImtpZCI6ImI5NmVkODUwNjQ0MWI2ZGY0MWQ1NjA3OTFlYmU2MTI3MjJjNmQ1ZDUifQ.eyJpc3MiOiJodHRwOi8vMTI3LjAuMC4xOjU1NTYvZGV4IiwicHJvdmlkZXJfaWQiOiJ3ZWIzIiwic3ViIjoiQ2lvd2VFSmxNemsyWWpSQ09EUmhOREV6T1VWQk5VTTBaR0poUkdSbE9UaEJSVGd6TmpSbFFqSXhaVGdTQkhkbFlqTSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxOTM0Mjg3MDYxLCJpYXQiOjE3MTgyODcwNjEsImF0X2hhc2giOiItZ3Y5RDl1NmpZQWZPa2NsVzJTaGlRIiwiZW1haWxfdmVyaWZpZWQiOmZhbHNlLCJldGhlcmV1bV9hZGRyZXNzIjoiMHhCZTM5NmI0Qjg0YTQxMzlFQTVDNGRiYURkZTk4QUU4MzY0ZUIyMWU4IiwibmFtZSI6IjB4QmUzOTZiNEI4NGE0MTM5RUE1QzRkYmFEZGU5OEFFODM2NGVCMjFlOCJ9.rA3sWT8rWNGebtpXhaU9kmszwNIcyWkUFN7lmKB4nbYtbClWkx0wqp0onP0y0Yl-Nt98PgiCxg5p-6yha2Vd0GiiIPore9GUAQ6KZMX4BZ4VcUqhqSw877z5FuyLUoxCOOz8V-rZM3AiN5YjuL9jUH5KChlsXSYt5EcAWfoUVMaRdmz1_VwpvumTwo_gbAZhqo7izbBKrQzs--tmOqM0BpDEKfsn6G-Q0YRkVIc35JHZ7ehtAw-NR1ufbaHtgX31S6mPqEkUNpXXj0Q-m3dyeWQhP8Qd6HtY0EzhqLzIUGhuMFjUh_tAq2naKfH6okC8Rwu7Xs2Jw2j_gbfxnAtivg",
	},
}

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
		JWTKeySetURL:                       fmt.Sprintf("%s/dex/keys", addr),
		AllowableEmailConfirmationLateness: time.Minute * 1,
	}
	s.app.Use(jwtware.New(jwtware.Config{
		JWKSetURLs: []string{s.settings.JWTKeySetURL},
	}))

	acctCont, err := NewAccountController(s.ctx, s.pdb, s.eventService, s.identityService, s.emailService, s.settings, test.Logger())
	s.Assert().NoError(err)
	s.controller = acctCont
	s.app.Post("/", s.controller.CreateUserAccount)
	s.app.Get("/", s.controller.GetUserAccount)
	s.app.Delete("/", s.controller.DeleteUser)
	s.app.Put("/update", s.controller.UpdateUser)

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

	if err := s.app.Shutdown(); err != nil {
		s.T().Fatal(err)
	}
}

func TestDevicesControllerTestSuite(t *testing.T) {
	suite.Run(t, new(AccountControllerTestSuite))
}

func (s *AccountControllerTestSuite) Test_EmailFirstAccount_CreateAndDelete() {
	// Create Account
	createAcctReq := test.BuildRequest("POST", "/", "", dexEmailUsers[0].AuthToken)
	createAcctResp, _ := s.app.Test(createAcctReq)
	createAcctBody, err := io.ReadAll(createAcctResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, createAcctResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(createAcctBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().Equal(dexEmailUsers[0].Email, userResp.Email.Address)
	s.Assert().True(userResp.Email.Confirmed)
	s.Assert().Nil(userResp.Web3)

	// Delete Account
	deleteReq := test.BuildRequest("DELETE", "/", "", dexEmailUsers[0].AuthToken)
	deleteResp, err := s.app.Test(deleteReq)
	s.Require().NoError(err)
	_, err = io.ReadAll(deleteReq.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, deleteResp.StatusCode)

}

func (s *AccountControllerTestSuite) Test_EmailFirstAccount_UpdateAccount() {
	// Create Account
	createAcctReq := test.BuildRequest("POST", "/", "", dexEmailUsers[0].AuthToken)
	createAcctResp, _ := s.app.Test(createAcctReq)
	s.Assert().Equal(200, createAcctResp.StatusCode)

	updateBody := UserUpdateRequest{
		CountryCode: "USA",
	}
	updateBodyBytes, _ := json.Marshal(updateBody)

	putReq := test.BuildRequest("PUT", "/update", string(updateBodyBytes), dexEmailUsers[0].AuthToken)
	putResp, _ := s.app.Test(putReq)
	putBody, err := io.ReadAll(putResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, putResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(putBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().NotNil(userResp.Email)
	s.Assert().Equal(dexEmailUsers[0].Email, userResp.Email.Address)
	s.Assert().Equal(updateBody.CountryCode, userResp.CountryCode)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_EmailFirstAccount_AgreeTOS() {
	// Create Account
	createAcctReq := test.BuildRequest("POST", "/", "", dexEmailUsers[0].AuthToken)
	createAcctResp, _ := s.app.Test(createAcctReq)
	s.Assert().Equal(200, createAcctResp.StatusCode)

	// Account 1 Agree TOS
	putReq := test.BuildRequest("POST", "/agree-tos", "", dexEmailUsers[0].AuthToken)
	putResp, _ := s.app.Test(putReq)
	_, err := io.ReadAll(putResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, putResp.StatusCode)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_EmailFirstAccount_LinkWallet() {
	// Create Account
	createAcctReq := test.BuildRequest("POST", "/", "", dexEmailUsers[0].AuthToken)
	createAcctResp, _ := s.app.Test(createAcctReq)
	s.Assert().Equal(200, createAcctResp.StatusCode)

	// Link wallet via token
	linkWalletBody := TokenBody{
		Token: dexWalletUsers[0].AuthToken,
	}
	linkWalletBodyBytes, _ := json.Marshal(linkWalletBody)
	req := test.BuildRequest("POST", "/link/wallet/token", string(linkWalletBodyBytes), dexEmailUsers[0].AuthToken)
	resp, _ := s.app.Test(req)
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, resp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(body, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().NotNil(userResp.Email)
	s.Assert().Equal(dexEmailUsers[0].Email, userResp.Email.Address)
	s.Assert().NotNil(userResp.Web3)
	s.Assert().Equal(userResp.Web3.Address.Hex(), dexWalletUsers[0].Wallet)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_WalletFirstAccount_CreateAndDelete() {
	// Create Account
	createAcctReq := test.BuildRequest("POST", "/", "", dexWalletUsers[1].AuthToken)
	createAcctResp, _ := s.app.Test(createAcctReq)
	createAcctBody, err := io.ReadAll(createAcctResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, createAcctResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(createAcctBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().Nil(userResp.Email)
	s.Require().NotNil(userResp.Web3)
	s.Assert().Equal(dexWalletUsers[1].Wallet, userResp.Web3.Address.Hex())
	s.Assert().True(userResp.Web3.Confirmed)

	// Set identity svc to be consistent with eligible deletion state
	test.IdentityServiceResponse = false

	// Delete Account
	deleteReq := test.BuildRequest("DELETE", "/", "", dexWalletUsers[1].AuthToken)
	deleteResp, err := s.app.Test(deleteReq)
	s.Require().NoError(err)
	_, err = io.ReadAll(deleteReq.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, deleteResp.StatusCode)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))

}

func (s *AccountControllerTestSuite) Test_WalletFirstAccount_LinkEmailToken() {
	// Create Account
	createAcctReq := test.BuildRequest("POST", "/", "", dexWalletUsers[0].AuthToken)
	createAcctResp, _ := s.app.Test(createAcctReq)
	s.Assert().Equal(200, createAcctResp.StatusCode)

	// Link email via token
	linkEmailBody := TokenBody{
		Token: dexEmailUsers[2].AuthToken,
	}
	linkEmailBodyBytes, _ := json.Marshal(linkEmailBody)

	putReq := test.BuildRequest("POST", "/link/email/token", string(linkEmailBodyBytes), dexWalletUsers[0].AuthToken)
	putResp, _ := s.app.Test(putReq)
	_, err := io.ReadAll(putResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, putResp.StatusCode)

	getReq := test.BuildRequest("GET", "/", "", dexWalletUsers[0].AuthToken)
	getResp, _ := s.app.Test(getReq)
	getBody, err := io.ReadAll(getResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, getResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(getBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().NotNil(userResp.Email)
	s.Assert().Equal(dexEmailUsers[2].Email, userResp.Email.Address)
	s.Assert().NotNil(userResp.Web3)
	s.Assert().Equal(userResp.Web3.Address.Hex(), dexWalletUsers[0].Wallet)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_WalletFirstAccount_LinkEmailConfirm() {
	// Create Account
	createAcctReq := test.BuildRequest("POST", "/", "", dexWalletUsers[0].AuthToken)
	createAcctResp, _ := s.app.Test(createAcctReq)
	s.Assert().Equal(200, createAcctResp.StatusCode)

	// Link email via confirmation code
	linkEmailBody := RequestEmailValidation{
		EmailAddress: dexEmailUsers[0].Email,
	}
	linkEmailBodyBytes, _ := json.Marshal(linkEmailBody)
	postReq := test.BuildRequest("POST", "/link/email", string(linkEmailBodyBytes), dexWalletUsers[0].AuthToken)
	postResp, _ := s.app.Test(postReq)
	_, err := io.ReadAll(postResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, postResp.StatusCode)

	eml, err := models.Emails(models.EmailWhere.EmailAddress.EQ(dexEmailUsers[0].Email)).One(s.ctx, s.pdb.DBS().Reader)
	s.Require().NoError(err)

	confirmEmailBody := CompleteEmailValidation{
		Key: eml.Code.String,
	}
	confirmEmailBytes, _ := json.Marshal(confirmEmailBody)
	confirmReq := test.BuildRequest("POST", "/link/email/confirm", string(confirmEmailBytes), dexWalletUsers[0].AuthToken)
	confirmResp, _ := s.app.Test(confirmReq)
	_, err = io.ReadAll(confirmResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(204, confirmResp.StatusCode)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_SubmitReferralCode() {
	// Create Account
	createAcctReq := test.BuildRequest("POST", "/", "", dexWalletUsers[0].AuthToken)
	createAcctResp, _ := s.app.Test(createAcctReq)
	s.Assert().Equal(200, createAcctResp.StatusCode)

	refAcct, err := test.NewAccount(s.pdb.DBS().Writer) // Create Referrer Account
	s.Require().NoError(err)

	referralCodeBody := SubmitReferralCodeRequest{
		ReferralCode: refAcct.ReferralCode.String,
	}
	referralCodeBodyBytes, _ := json.Marshal(referralCodeBody)

	// Set identity svc to be consistent with referral eligibility
	test.IdentityServiceResponse = false

	// Submit referral
	postReq := test.BuildRequest("POST", "/referral/submit", string(referralCodeBodyBytes), dexWalletUsers[0].AuthToken)
	postResp, _ := s.app.Test(postReq)
	_, err = io.ReadAll(postResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, postResp.StatusCode)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
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
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_ConflictingEmail_Token() {
	// Create Account 1
	createEmailAcctReq := test.BuildRequest("POST", "/", "", dexEmailUsers[2].AuthToken)
	createEmailAcctResp, _ := s.app.Test(createEmailAcctReq)
	s.Assert().Equal(200, createEmailAcctResp.StatusCode)

	// Create Account 2
	createWalletAcctReq := test.BuildRequest("POST", "/", "", dexWalletUsers[1].AuthToken)
	createWalletAcctResp, _ := s.app.Test(createWalletAcctReq)
	createWalletAcctBody, err := io.ReadAll(createWalletAcctResp.Body)
	s.Require().NoError(err)
	fmt.Println("Body: ", string(createWalletAcctBody))
	s.Assert().Equal(200, createWalletAcctResp.StatusCode)

	// Account 2 attempts to link email of account 1 via token
	linkEmailBody := TokenBody{
		Token: dexEmailUsers[2].AuthToken,
	}
	linkEmailBodyBytes, _ := json.Marshal(linkEmailBody)

	linkEmailReq := test.BuildRequest("POST", "/link/email/token", string(linkEmailBodyBytes), dexWalletUsers[1].AuthToken)
	linkEmailResp, _ := s.app.Test(linkEmailReq)
	resp, err := io.ReadAll(linkEmailResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(string(resp), `models: unable to insert into emails: pq: duplicate key value violates unique constraint "emails_pkey"`)
	s.Assert().Equal(500, linkEmailResp.StatusCode)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_ConflictingEmail_Challenge() {
	// Create Account 1
	createEmailAcctReq := test.BuildRequest("POST", "/", "", dexEmailUsers[0].AuthToken)
	createEmailAcctResp, _ := s.app.Test(createEmailAcctReq)
	s.Assert().Equal(200, createEmailAcctResp.StatusCode)

	// Create Account 2
	createWalletAcctReq := test.BuildRequest("POST", "/", "", dexWalletUsers[0].AuthToken)
	createWalletAcctResp, _ := s.app.Test(createWalletAcctReq)
	s.Assert().Equal(200, createWalletAcctResp.StatusCode)

	linkEmailBody := RequestEmailValidation{
		EmailAddress: dexEmailUsers[0].Email,
	}

	linkEmailBodyBytes, _ := json.Marshal(linkEmailBody)
	postReq := test.BuildRequest("POST", "/link/email", string(linkEmailBodyBytes), dexWalletUsers[0].AuthToken)
	postResp, _ := s.app.Test(postReq)
	resp, err := io.ReadAll(postResp.Body)
	s.Require().NoError(err)
	s.Require().Equal(string(resp), `email address linked to another account`)
	s.Assert().Equal(500, postResp.StatusCode)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_ConflictingWallet() {
	// Create Account 1
	createEmailAcctReq := test.BuildRequest("POST", "/", "", dexEmailUsers[2].AuthToken)
	createEmailAcctResp, _ := s.app.Test(createEmailAcctReq)
	s.Assert().Equal(200, createEmailAcctResp.StatusCode)

	// Create Account 2
	createWalletAcctReq := test.BuildRequest("POST", "/", "", dexWalletUsers[0].AuthToken)
	createWalletAcctResp, _ := s.app.Test(createWalletAcctReq)
	s.Assert().Equal(200, createWalletAcctResp.StatusCode)

	// Account 1 attempts to link wallet of account 2 via token
	linkWalletToken := TokenBody{
		Token: dexWalletUsers[0].AuthToken,
	}
	linkWalletTokenBytes, _ := json.Marshal(linkWalletToken)

	linkWalletReq := test.BuildRequest("POST", "/link/wallet/token", string(linkWalletTokenBytes), dexEmailUsers[2].AuthToken)
	linkWalletResp, _ := s.app.Test(linkWalletReq)
	linkWalletBody, err := io.ReadAll(linkWalletResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(string(linkWalletBody), `models: unable to insert into wallets: pq: duplicate key value violates unique constraint "wallets_pkey"`)
	s.Assert().Equal(500, linkWalletResp.StatusCode)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_EmailFirst_AlternativeSignIn_Token() {
	// Create Account
	createAcctReq := test.BuildRequest("POST", "/", "", dexEmailUsers[0].AuthToken)
	createAcctResp, _ := s.app.Test(createAcctReq)
	s.Assert().Equal(200, createAcctResp.StatusCode)

	linkEmailBody := TokenBody{
		Token: dexWalletUsers[2].AuthToken,
	}
	linkEmailBodyBytes, _ := json.Marshal(linkEmailBody)

	linkWalletReq := test.BuildRequest("POST", "/link/wallet/token", string(linkEmailBodyBytes), dexEmailUsers[0].AuthToken)
	linkWalletResp, _ := s.app.Test(linkWalletReq)
	s.Assert().Equal(200, linkWalletResp.StatusCode)

	getAcctReq := test.BuildRequest("GET", "/", "", dexWalletUsers[2].AuthToken)
	getAcctResp, _ := s.app.Test(getAcctReq)
	getAcctBody, err := io.ReadAll(getAcctResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, getAcctResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(getAcctBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().NotNil(userResp.Email)
	s.Assert().Equal(dexEmailUsers[0].Email, userResp.Email.Address)
	s.Assert().NotNil(userResp.Web3)
	s.Assert().Equal(userResp.Web3.Address.Hex(), dexWalletUsers[2].Wallet)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_WalletFirst_AlternativeSignIn_Token() {
	// Create Account
	createAcctReq := test.BuildRequest("POST", "/", "", dexWalletUsers[0].AuthToken)
	createAcctResp, _ := s.app.Test(createAcctReq)
	s.Assert().Equal(200, createAcctResp.StatusCode)

	linkEmailBody := TokenBody{
		Token: dexEmailUsers[2].AuthToken,
	}
	linkEmailBodyBytes, _ := json.Marshal(linkEmailBody)

	linkEmailReq := test.BuildRequest("POST", "/link/email/token", string(linkEmailBodyBytes), dexWalletUsers[0].AuthToken)
	linkEmailResp, _ := s.app.Test(linkEmailReq)
	s.Assert().Equal(204, linkEmailResp.StatusCode)

	getAcctReq := test.BuildRequest("GET", "/", "", dexEmailUsers[2].AuthToken)
	getAcctResp, _ := s.app.Test(getAcctReq)
	getAcctBody, err := io.ReadAll(getAcctResp.Body)
	s.Require().NoError(err)
	s.Assert().Equal(200, getAcctResp.StatusCode)

	var userResp UserResponse
	if err := json.Unmarshal(getAcctBody, &userResp); err != nil {
		s.Require().NoError(err)
	}

	s.Assert().NotNil(userResp.Email)
	s.Assert().Equal(dexEmailUsers[2].Email, userResp.Email.Address)
	s.Assert().NotNil(userResp.Web3)
	s.Assert().Equal(userResp.Web3.Address.Hex(), dexWalletUsers[0].Wallet)
	s.Require().NoError(test.DeleteAll(s.pdb.DBS().Writer))
}

func (s *AccountControllerTestSuite) Test_JWTDecodeEmailAuthToken() {
	jwkResource, err := keyfunc.NewDefaultCtx(context.Background(), []string{s.settings.JWTKeySetURL})
	s.Require().NoError(err)

	for _, user := range dexEmailUsers {
		tbClaims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(user.AuthToken, &tbClaims, jwkResource.Keyfunc)
		s.Require().NoError(err)
		claims := *getUserAccountInfos(tbClaims)
		s.Require().Equal(*claims.EmailAddress, user.Email)
	}
}

func (s *AccountControllerTestSuite) Test_JWTDecodeWalletAuthToken() {
	jwkResource, err := keyfunc.NewDefaultCtx(context.Background(), []string{s.settings.JWTKeySetURL})
	s.Require().NoError(err)

	for _, user := range dexWalletUsers {
		tbClaims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(user.AuthToken, &tbClaims, jwkResource.Keyfunc)
		s.Require().NoError(err)
		claims := *getUserAccountInfos(tbClaims)
		s.Require().Equal(claims.EthereumAddress.Hex(), user.Wallet)
	}
}
