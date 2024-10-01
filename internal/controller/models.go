package controller

import (
	_ "embed"
	"regexp"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var referralCodeRegex = regexp.MustCompile(`^[A-Z0-9]{6}$`)
var emailPattern = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// var digits = []rune("0123456789")

type TokenBody struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"`
}

type UserResponseEmail struct {
	// Address is the email address for the user.
	Address string `json:"address" swaggertype:"string" example:"koblitz@dimo.zone"`
	// Confirmed indicates whether the user has confirmed the address by entering a code.
	Confirmed bool `example:"false" json:"confirmed"`
	// CodeExpiresAt is the time at which the current confirmation code will expire. This will only
	// be present if we've sent an email but the code has not been sent back to us. In that case, Confirmed
	// will certainly be false.
	CodeExpiresAt *time.Time `json:"codeExpiresAt" swaggertype:"string" example:"2021-12-01T09:01:12Z"`
}

type UserResponseWeb3 struct {
	// Address is the Ethereum address associated with the user.
	Address common.Address `json:"address" swaggertype:"string" example:"0x142e0C7A098622Ea98E5D67034251C4dFA746B5d"`
}

type UserResponse struct {
	// ID is the user's DIMO-internal ID.
	ID string `json:"id" example:"2mD8CtraxOCAAwIeydt2Q4oCiAQ"`

	// Email describes the user's email and the state of its confirmation.
	Email *UserResponseEmail `json:"email"`
	// Wallet describes the user's blockchain account.
	Wallet *UserResponseWeb3 `json:"wallet"`
	// CountryCode, if present, is a valid ISO 3166-1 alpha-3 country code.
	CountryCode *string `json:"countryCode" swaggertype:"string" example:"USA"`
	// AcceptedTOSAt is the time at which the user last agreed to the terms of service.
	AcceptedTOSAt *time.Time `json:"acceptedTosAt,omitempty" swaggertype:"string" example:"2021-12-01T09:00:41Z"`
	// ReferralCode is the user's referral code to be given to others. It is an 8 alphanumeric code,
	// only present if the account has a confirmed Ethereum address.
	ReferralCode string     `json:"-" swaggertype:"string" example:"ANB95N"`
	ReferredBy   string     `json:"-" swaggertype:"string" example:"0x3497B704a954789BC39999262510DE9B09Ff1366"`
	ReferredAt   *time.Time `json:"-" swaggertype:"string" example:"2021-12-01T09:00:41Z"`

	// CreatedAt is when the user first logged in.
	CreatedAt time.Time `json:"createdAt" example:"2021-12-01T09:00:00Z"`
	// UpdatedAt reflects the time of the most recent account changes.
	UpdatedAt time.Time `json:"updatedAt" example:"2021-12-01T09:00:00Z"`
}

type SubmitReferralCodeRequest struct {
	Code string `json:"code" example:"ANBJN5"`
}

type SubmitReferralCodeResponse struct {
	Message string `json:"message"`
}

// UserUpdateRequest describes a user's request to modify or delete certain fields
// Currently contains only CountryCode as dedicated endpoints exist for other types
// of updates a user might make
type UserUpdateRequest struct {
	// CountryCode should be a valid ISO 3166-1 alpha-3 country code
	CountryCode string `json:"countryCode,omitempty" swaggertype:"string" example:"USA"`
}

// RequestEmailValidation request body used for adding an email that cannot be authenticated via federated sign in to account
type RequestEmailValidation struct {
	Address string `json:"address" swaggertype:"string" example:"kilgore@kilgore.trout"`
}

type CompleteEmailValidation struct {
	// Code is the 6-digit number from the confirmation email
	Code string `json:"code" example:"010990"`
}

type ErrorRes struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type StandardRes struct {
	Message string `json:"message"`
}
