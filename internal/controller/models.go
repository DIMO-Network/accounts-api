package controller

import (
	_ "embed"
	"regexp"
	"time"
)

var referralCodeRegex = regexp.MustCompile(`^[A-Z0-9]{6}$`)

// emailPattern is a rough email validation regular expression taken from the HTML 5 spec.
// https://html.spec.whatwg.org/multipage/input.html#valid-e-mail-address
var emailPattern = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// emailCodeDigits are the digits allowed in an email confirmation code.
var emailCodeDigits = []rune("0123456789")

type TokenBody struct {
	Token string `json:"token"`
}

type UserResponseEmail struct {
	// Address is the email address for the user.
	Address string `json:"address,omitempty" swaggertype:"string" example:"koblitz@dimo.zone"`
	// Confirmed indicates whether the user has confirmed the address by entering a code.
	Confirmed bool `example:"false" json:"confirmed"`
	// ConfirmationSentAt is the time at which we last sent a confirmation email. This will only
	// be present if we've sent an email but the code has not been sent back to us.
	ConfirmationSentAt *time.Time `json:"confirmationSentAt,omitempty" swaggertype:"string" example:"2021-12-01T09:01:12Z"`
}

type UserResponseWallet struct {
	// Address is the Ethereum address associated with the user.
	Address string `json:"address,omitempty" swaggertype:"string" example:"0x142e0C7A098622Ea98E5D67034251C4dFA746B5d"`
}

type UserResponse struct {
	// ID is the user's DIMO-internal ID.
	ID string `json:"id" example:"ChFrb2JsaXR6QGRpbW8uem9uZRIGZ29vZ2xl"`
	// Email describes the user's email and the state of its confirmation.
	Email *UserResponseEmail `json:"email,omitempty"`
	// Wallet describes the user's blockchain account.
	Wallet *UserResponseWallet `json:"wallet,omitempty"`
	// CreatedAt is when the user first logged in.
	CreatedAt time.Time `json:"createdAt,omitempty" swaggertype:"string" example:"2021-12-01T09:00:00Z"`
	// UpdatedAt reflects the time of the most recent account changes.
	UpdatedAt time.Time `json:"updatedAt,omitempty" swaggertype:"string" example:"2021-12-01T09:00:00Z"`
	// CountryCode, if present, is a valid ISO 3166-1 alpha-3 country code.
	CountryCode string `json:"countryCode,omitempty" swaggertype:"string" example:"USA"`
	// AcceptedTOSAt is the time at which the user agreed to the terms of service.
	AcceptedTOSAt *time.Time `json:"acceptedTOSAt,omitempty" swaggertype:"string" example:"2021-12-01T09:00:41Z"`
	// ReferralCode is the user's referral code to be given to others. It is an 8 alphanumeric code,
	// only present if the account has a confirmed Ethereum address.
	ReferralCode string    `json:"referralCode,omitempty" swaggertype:"string" example:"ANB95N"`
	ReferredBy   string    `json:"referredBy,omitempty" swaggertype:"string" example:"0x3497B704a954789BC39999262510DE9B09Ff1366"`
	ReferredAt   time.Time `json:"referredAt,omitempty" swaggertype:"string" example:"2021-12-01T09:00:41Z"`
}

type SubmitReferralCodeRequest struct {
	ReferralCode string `json:"referralCode" example:"ANB1N5"`
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
	EmailAddress string `json:"email,omitempty" swaggertype:"string" example:"kilgore@kilgore.trout"`
}

type CompleteEmailValidation struct {
	// Key is the 6-digit number from the confirmation email
	Key string `json:"key" example:"010990"`
}

type ErrorRes struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
