package cio

import (
	"context"
	"errors"

	"github.com/DIMO-Network/accounts-api/internal/config"
	analytics "github.com/customerio/cdp-analytics-go"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mixpanel/mixpanel-go"
	"github.com/rs/zerolog"
)

const walletTrait = "wallet"

// Need to rename this package.
type Client struct {
	client                  analytics.Client
	mixClient               *mixpanel.ApiClient
	disableCustomerIOEvents bool
}

func New(settings *config.Settings, logger *zerolog.Logger) (*Client, error) {
	var mixClient *mixpanel.ApiClient
	if settings.MixpanelProjectToken != "" {
		mixClient = mixpanel.NewApiClient(settings.MixpanelProjectToken)
	}

	client, err := analytics.NewWithConfig(settings.CustomerIOAPIKey, analytics.Config{})
	if err != nil {
		return nil, err
	}

	return &Client{
		client:                  client,
		disableCustomerIOEvents: settings.DisableCustomerIOEvents,
		mixClient:               mixClient,
	}, nil
}

func (c *Client) SetEmail(ctx context.Context, id, email string) error {
	// TODO(elffjs): This join is pretty gross. Separate these two services.
	var err error
	if c.mixClient != nil {
		pp := mixpanel.NewPeopleProperties(id, nil)
		pp.SetReservedProperty(mixpanel.PeopleEmailProperty, email)
		// TODO(elffjs): Really ought to bail if this fails for context reasons.
		err = errors.Join(err, c.mixClient.PeopleSet(ctx, []*mixpanel.PeopleProperties{pp}))
	}

	return errors.Join(err, c.client.Enqueue(analytics.Identify{
		UserId: id,
		Traits: analytics.NewTraits().SetEmail(email),
	}))
}

func (c *Client) SetWallet(ctx context.Context, id string, wallet common.Address) error {
	var err error
	if c.mixClient != nil {
		pp := mixpanel.NewPeopleProperties(id, map[string]any{
			walletTrait: wallet.Hex(),
		})
		err = errors.Join(err, c.mixClient.PeopleSet(ctx, []*mixpanel.PeopleProperties{pp}))
	}

	return errors.Join(err, c.client.Enqueue(analytics.Identify{
		UserId: id,
		Traits: analytics.NewTraits().Set(walletTrait, wallet.Hex()),
	}))
}
