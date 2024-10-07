package cio

import (
	"github.com/DIMO-Network/accounts-api/internal/config"

	analytics "github.com/customerio/cdp-analytics-go"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
)

const walletTrait = "wallet"

type Client struct {
	client                  analytics.Client
	disableCustomerIOEvents bool
}

func New(settings *config.Settings, logger *zerolog.Logger) (*Client, error) {
	client, err := analytics.NewWithConfig(settings.CustomerIOAPIKey, analytics.Config{})
	if err != nil {
		return nil, err
	}

	return &Client{
		client:                  client,
		disableCustomerIOEvents: settings.DisableCustomerIOEvents,
	}, nil

}

func (c *Client) SetEmail(id, email string) error {
	return c.client.Enqueue(analytics.Identify{
		UserId: id,
		Traits: analytics.NewTraits().SetEmail(email),
	})
}

func (c *Client) SetWallet(id string, wallet common.Address) error {
	return c.client.Enqueue(analytics.Identify{
		UserId: id,
		Traits: analytics.NewTraits().Set(walletTrait, wallet.Hex()),
	})
}
