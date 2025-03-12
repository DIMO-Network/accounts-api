package cio

import (
	"context"

	"github.com/DIMO-Network/accounts-api/internal/config"
	analytics "github.com/customerio/cdp-analytics-go"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
)

const walletTrait = "wallet"

// Need to rename this package.
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

func (c *Client) SetEmail(ctx context.Context, wallet common.Address, email string) error {
	return c.client.Enqueue(analytics.Identify{
		UserId: wallet.Hex(),
		Traits: analytics.NewTraits().SetEmail(email),
	})
}

func (c *Client) SetWallet(ctx context.Context, wallet common.Address) error {
	return c.client.Enqueue(analytics.Identify{
		UserId: wallet.Hex(),
		Traits: analytics.NewTraits().Set(walletTrait, wallet.Hex()),
	})
}
