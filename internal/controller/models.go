package controller

import (
	_ "embed"
	"regexp"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/volatiletech/null/v8"
)

var referralCodeRegex = regexp.MustCompile(`^[A-Z0-9]{6}$`)
var emailPattern = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type ConfirmEmailRequest struct {
	// Key is the 6-digit number from the confirmation email
	Key string `json:"key" example:"010990"`
}

// TODO AE: find out what body will be
type TokenBody struct {
	Token string `json:"token"`
}

type UserResponseEmail struct {
	// Address is the email address for the user.
	Address string `json:"address" swaggertype:"string" example:"koblitz@dimo.zone",omitempty`
	// Confirmed indicates whether the user has confirmed the address by entering a code.
	Confirmed bool `json:"confirmed" example:"false",omitempty`
	// ConfirmationSentAt is the time at which we last sent a confirmation email. This will only
	// be present if we've sent an email but the code has not been sent back to us.
	ConfirmationSentAt time.Time `json:"confirmationSentAt" swaggertype:"string" example:"2021-12-01T09:01:12Z",omitempty`
}

type UserResponseWeb3 struct {
	// Address is the Ethereum address associated with the user.
	Address common.Address `json:"address" swaggertype:"string" example:"0x142e0C7A098622Ea98E5D67034251C4dFA746B5d",omitempty`
	// Confirmed indicates whether the user has confirmed the address by signing a challenge
	// message.
	Confirmed bool `json:"confirmed" example:"false",omitempty`
	// Used indicates whether the user has used this address to perform any on-chain
	// actions like minting, claiming, or pairing.
	Used bool `json:"used" example:"false",omitempty`
	// InApp indicates whether this is an in-app wallet, managed by the DIMO app.
	Provider string `json:"inApp" example:"false",omitempty`
}

type UserResponse struct {
	// ID is the user's DIMO-internal ID.
	ID string `json:"id" example:"ChFrb2JsaXR6QGRpbW8uem9uZRIGZ29vZ2xl"`
	// Email describes the user's email and the state of its confirmation.
	Email *UserResponseEmail `json:"email",omitempty`
	// Web3 describes the user's blockchain account.
	Web3 *UserResponseWeb3 `json:"web3",omitempty`
	// CreatedAt is when the user first logged in.
	CreatedAt time.Time `json:"createdAt" swaggertype:"string" example:"2021-12-01T09:00:00Z",omitempty`
	// CountryCode, if present, is a valid ISO 3166-1 alpha-3 country code.
	CountryCode string `json:"countryCode" swaggertype:"string" example:"USA",omitempty`
	// AgreedTosAt is the time at which the user last agreed to the terms of service.
	AgreedTOSAt time.Time `json:"agreedTosAt" swaggertype:"string" example:"2021-12-01T09:00:41Z",omitempty`
	// ReferralCode is the user's referral code to be given to others. It is an 8 alphanumeric code,
	// only present if the account has a confirmed Ethereum address.
	ReferralCode string    `json:"referralCode" swaggertype:"string" example:"ANB95N",omitempty`
	ReferredBy   string    `json:"referredBy" swaggertype:"string" example:"0x3497B704a954789BC39999262510DE9B09Ff1366",omitempty`
	ReferredAt   time.Time `json:"referredAt" swaggertype:"string" example:"2021-12-01T09:00:41Z",omitempty`
}

type SubmitReferralCodeRequest struct {
	// ReferralCode is the 6-digit, alphanumeric referral code from another user.
	ReferralCode string `json:"referralCode" example:"ANB95N"`
}

type SubmitReferralCodeResponse struct {
	Message string `json:"message"`
}

const UserCreationEventType = "com.dimo.zone.user.create"

type UserCreationEventData struct {
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"userId"`
	Method    string    `json:"method"`
}

type optionalString struct {
	Defined bool
	Value   null.String
}

// UserUpdateRequest describes a user's request to modify or delete certain fields
type UserUpdateRequest struct {
	Email struct {
		// Address, if present, should be a valid email address. Note when this field
		// is modified the user's verification status will reset.
		Address optionalString `json:"address" swaggertype:"string" example:"neal@dimo.zone"`
	} `json:"email"`
	Web3 struct {
		// Address, if present, should be a valid ethereum address. Note when this field
		// is modified the user's address verification status will reset.
		Address optionalString `json:"address" swaggertype:"string" example:"0x71C7656EC7ab88b098defB751B7401B5f6d8976F"`
		// InApp, if true, indicates that the address above corresponds to an in-app wallet.
		// You can only set this when setting a new wallet. It defaults to false.
		InApp bool `json:"inApp" example:"true"`
	} `json:"web3"`
	// CountryCode, if specified, should be a valid ISO 3166-1 alpha-3 country code
	CountryCode optionalString `json:"countryCode" swaggertype:"string" example:"USA"`
}

var digits = []rune("0123456789")

type ChallengeResponse struct {
	// Challenge is the message to be signed.
	Challenge string `json:"challenge"`
	// ExpiresAt is the time at which the signed challenge will no longer be accepted.
	ExpiresAt time.Time `json:"expiresAt"`
}

type ConfirmEthereumRequest struct {
	// Signature is the result of signing the provided challenge message using the address in
	// question.
	Signature string `json:"signature"`
}

type AltAccount struct {
	// Type is the authentication provider, one of "web3", "apple", "google".
	Type string `json:"type"`
	// Login is the login username for the provider, either an email address
	// or an EIP-55-compliant ethereum address.
	Login string `json:"login"`
}

type AlternateAccountsResponse struct {
	// OtherAccounts is a list of any other accounts that share email or
	// ethereum address with the provided token.
	OtherAccounts []*AltAccount `json:"otherAccounts"`
}
