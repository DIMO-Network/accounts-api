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
	client, err := analytics.NewWithConfig(settings.CustomerIOAPIKey, analytics.Config{
		Callback: callbackI{
			logger: logger,
		},
	})
	if err != nil {
		return nil, err
	}

	if settings.DisableCustomerIOEvents {
		logger.Info().Msg("Customer.io events are disabled")
	}

	return &Client{
		client:                  client,
		disableCustomerIOEvents: settings.DisableCustomerIOEvents,
	}, nil

}

func (c *Client) SendCustomerIoEvent(customerID string, email *string, wallet *common.Address) error {
	if c.disableCustomerIOEvents {
		return nil
	}

	userTraits := analytics.NewTraits()
	if email != nil {
		userTraits.SetEmail(*email)
	}

	if wallet != nil {
		userTraits.Set(walletTrait, wallet.Hex())
	}

	return c.client.Enqueue(analytics.Identify{
		UserId: customerID,
		Traits: userTraits,
	})
}

func (c *Client) Close() {
	c.client.Close()
}

// callbackI is used to log when a message send succeeded or failed
type callbackI struct {
	logger *zerolog.Logger
}

func (c callbackI) Failure(m analytics.Message, err error) {
	id := m.(analytics.Identify)
	c.logger.Error().Err(err).Interface("traits", id.Traits).Msgf("failed to send customer io identify message for customer: %s", id.UserId)
}

func (c callbackI) Success(_ analytics.Message) {
}