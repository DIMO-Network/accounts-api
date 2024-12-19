package cio

import (
	"github.com/DIMO-Network/accounts-api/internal/config"
	analytics "github.com/customerio/cdp-analytics-go"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mixpanel/mixpanel-go"
	"github.com/rs/zerolog"
)

const walletTrait = "wallet"

type Client struct {
	client                  analytics.Client
	mixClient               *mixpanel.ApiClient
	disableCustomerIOEvents bool
}

func New(settings *config.Settings, logger *zerolog.Logger) (*Client, error) {
	mixpanel.NewApiClient()
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
	mixpanel.NewPeopleProperties(
		id, map[string]any{
			string(mixpanel.PeopleEmailProperty): email,
		},
	)

	c.mixClient.PeopleSet()

	return c.client.Enqueue(analytics.Identify{
		UserId: id,
		Traits: analytics.NewTraits().SetEmail(email),
	})
}

func (c *Client) SetWallet(id string, wallet common.Address) error {
	mixpanel.NewPeopleProperties(
		id, map[string]any{
			walletTrait: wallet.Hex(),
		},
	)

	return c.client.Enqueue(analytics.Identify{
		UserId: id,
		Traits: analytics.NewTraits().Set(walletTrait, wallet.Hex()),
	})
}
