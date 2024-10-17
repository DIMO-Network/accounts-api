package controller

import (
	_ "embed"
	"regexp"
	"time"
)

var referralCodeRegex = regexp.MustCompile(`^[A-Z0-9]{6}$`)

// emailPattern is the regular expression validation used for <input type="email"> from the HTML5 spec.
// https://html.spec.whatwg.org/multipage/input.html#email-state-(type=email)
var emailPattern = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type TokenBody struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"`
}

type UserResponseEmail struct {
	// Address is the email address for the user.
	Address string `json:"address" swaggertype:"string" example:"koblitz@dimo.zone"`
	// ConfirmedAt indicates the time at which the user confirmed the email. It may be null.
	ConfirmedAt *time.Time `example:"2021-12-01T09:00:41Z" json:"confirmedAt"`
}

type UserResponseWallet struct {
	// Address is the Ethereum address associated with the user.
	Address string `json:"address" swaggertype:"string" example:"0x142e0C7A098622Ea98E5D67034251C4dFA746B5d"`
}

type UserResponseReferral struct {
	// Code is the user's referral code.
	Code string `json:"code"`
	// ReferredBy is the address of the user who referred the calling user. It's possible for this
	// to be empty while ReferredAt is not, in the case when the referring user has deleted
	// their account.
	ReferredBy *string `json:"referredBy"`
	// The timestamp at which the user was referred. May be empty if the user wasn't referred.
	ReferredAt *time.Time `json:"referredAt"`
}

type UserResponse struct {
	// ID is the user's DIMO-internal ID.
	ID string `json:"id" example:"2mD8CtraxOCAAwIeydt2Q4oCiAQ"`

	// Email describes the user's email and the state of its confirmation.
	Email *UserResponseEmail `json:"email,omitempty"`
	// Wallet describes the user's blockchain account.
	Wallet *UserResponseWallet `json:"wallet,omitempty"`

	// Referral describes the account's referral code and information about who, if anyone,
	// referred the account. This is only available if the account has a linked wallet.
	Referral *UserResponseReferral `json:"referral,omitempty"`

	// CountryCode, if present, is a valid ISO 3166-1 alpha-3 country code.
	CountryCode *string `json:"countryCode" swaggertype:"string" example:"USA"`
	// AcceptedTOSAt is the time at which the user last agreed to the terms of service.
	AcceptedTOSAt *time.Time `json:"acceptedTosAt,omitempty" swaggertype:"string" example:"2021-12-01T09:00:41Z"`

	// CreatedAt is when the user first logged in.
	CreatedAt time.Time `json:"createdAt" example:"2021-12-01T09:00:00Z"`
	// UpdatedAt reflects the time of the most recent account changes.
	UpdatedAt time.Time `json:"updatedAt" example:"2021-12-01T09:00:00Z"`
}

type SubmitReferralCodeRequest struct {
	Code string `json:"code" example:"ANBJN5"`
}

// UserUpdateRequest describes a user's request to modify or delete certain fields
// Currently contains only CountryCode as dedicated endpoints exist for other types
// of updates a user might make
type UserUpdateRequest struct {
	// CountryCode should be a valid ISO 3166-1 alpha-3 country code
	CountryCode string `json:"countryCode,omitempty" swaggertype:"string" example:"USA"`
}

// AddEmailRequest request body used for adding an email that cannot be authenticated via federated sign in to account
type AddEmailRequest struct {
	Address string `json:"address" swaggertype:"string" example:"kilgore@kilgore.trout"`
}

type CompleteEmailValidation struct {
	// Code is the 6-digit number from the confirmation email
	Code string `json:"code" example:"010990"`
}

type ErrorRes struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"Malformed request body."`
}

type StandardRes struct {
	Message string `json:"message" example:"Operation succeeded."`
}
