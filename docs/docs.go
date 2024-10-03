// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/v1/account": {
            "get": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Get attributes for the authenticated user.",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.UserResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    }
                }
            },
            "put": {
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Modify attributes for the authenticated user",
                "parameters": [
                    {
                        "description": "New field values",
                        "name": "userUpdateRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/internal_controller.UserUpdateRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.UserResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BearerAuth": []
                    }
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Create user account using an auth token in the header.",
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.UserResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    }
                }
            },
            "delete": {
                "summary": "Delete the authenticated user. Fails if the user has any devices.",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.StandardRes"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    },
                    "409": {
                        "description": "Returned if the user still has devices.",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    }
                }
            }
        },
        "/v1/account/accept-tos": {
            "post": {
                "summary": "Agree to the current terms of service",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.StandardRes"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    }
                }
            }
        },
        "/v1/account/link/email": {
            "post": {
                "tags": [
                    "email"
                ],
                "summary": "Send a confirmation email to the authenticated user",
                "parameters": [
                    {
                        "description": "Specifies the email to be linked",
                        "name": "confirmEmailRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/internal_controller.RequestEmailValidation"
                        }
                    }
                ],
                "responses": {
                    "204": {
                        "description": "No Content"
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    }
                }
            }
        },
        "/v1/account/link/email/token": {
            "post": {
                "tags": [
                    "email"
                ],
                "summary": "Link an email to existing wallet account; require a signed JWT from auth server",
                "parameters": [
                    {
                        "description": "Includes the email token",
                        "name": "linkEmailRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/internal_controller.TokenBody"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.StandardRes"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    }
                }
            }
        },
        "/v1/account/link/wallet/token": {
            "post": {
                "tags": [
                    "wallet"
                ],
                "summary": "Link a wallet to an existing account.",
                "parameters": [
                    {
                        "description": "JWT with an ethereum_address claim.",
                        "name": "linkWalletRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/internal_controller.TokenBody"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.StandardRes"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    }
                }
            }
        },
        "/v1/accounts/referral/submit": {
            "post": {
                "tags": [
                    "referral"
                ],
                "summary": "Takes the referral code, validates and stores it",
                "parameters": [
                    {
                        "description": "ReferralCode is the 6-digit, alphanumeric referral code from another user.",
                        "name": "submitReferralCodeRequest",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/internal_controller.SubmitReferralCodeRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.SubmitReferralCodeResponse"
                        }
                    },
                    "400": {
                        "description": "Bad Request",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/internal_controller.ErrorRes"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "internal_controller.ErrorRes": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "integer",
                    "example": 400
                },
                "message": {
                    "type": "string",
                    "example": "Malformed request body."
                }
            }
        },
        "internal_controller.RequestEmailValidation": {
            "type": "object",
            "properties": {
                "address": {
                    "type": "string",
                    "example": "kilgore@kilgore.trout"
                }
            }
        },
        "internal_controller.StandardRes": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "Operation succeeded."
                }
            }
        },
        "internal_controller.SubmitReferralCodeRequest": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "string",
                    "example": "ANBJN5"
                }
            }
        },
        "internal_controller.SubmitReferralCodeResponse": {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string"
                }
            }
        },
        "internal_controller.TokenBody": {
            "type": "object",
            "properties": {
                "token": {
                    "type": "string",
                    "example": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
                }
            }
        },
        "internal_controller.UserResponse": {
            "type": "object",
            "properties": {
                "acceptedTosAt": {
                    "description": "AcceptedTOSAt is the time at which the user last agreed to the terms of service.",
                    "type": "string",
                    "example": "2021-12-01T09:00:41Z"
                },
                "countryCode": {
                    "description": "CountryCode, if present, is a valid ISO 3166-1 alpha-3 country code.",
                    "type": "string",
                    "example": "USA"
                },
                "createdAt": {
                    "description": "CreatedAt is when the user first logged in.",
                    "type": "string",
                    "example": "2021-12-01T09:00:00Z"
                },
                "email": {
                    "description": "Email describes the user's email and the state of its confirmation.",
                    "allOf": [
                        {
                            "$ref": "#/definitions/internal_controller.UserResponseEmail"
                        }
                    ]
                },
                "id": {
                    "description": "ID is the user's DIMO-internal ID.",
                    "type": "string",
                    "example": "2mD8CtraxOCAAwIeydt2Q4oCiAQ"
                },
                "updatedAt": {
                    "description": "UpdatedAt reflects the time of the most recent account changes.",
                    "type": "string",
                    "example": "2021-12-01T09:00:00Z"
                },
                "wallet": {
                    "description": "Wallet describes the user's blockchain account.",
                    "allOf": [
                        {
                            "$ref": "#/definitions/internal_controller.UserResponseWeb3"
                        }
                    ]
                }
            }
        },
        "internal_controller.UserResponseEmail": {
            "type": "object",
            "properties": {
                "address": {
                    "description": "Address is the email address for the user.",
                    "type": "string",
                    "example": "koblitz@dimo.zone"
                },
                "confirmedAt": {
                    "description": "ConfirmedAt indicates the time at which the user confirmed the email. It may be null.",
                    "type": "string",
                    "example": "2021-12-01T09:00:41Z"
                }
            }
        },
        "internal_controller.UserResponseWeb3": {
            "type": "object",
            "properties": {
                "address": {
                    "description": "Address is the Ethereum address associated with the user.",
                    "type": "string",
                    "example": "0x142e0C7A098622Ea98E5D67034251C4dFA746B5d"
                }
            }
        },
        "internal_controller.UserUpdateRequest": {
            "type": "object",
            "properties": {
                "countryCode": {
                    "description": "CountryCode should be a valid ISO 3166-1 alpha-3 country code",
                    "type": "string",
                    "example": "USA"
                }
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "DIMO Accounts API",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
