package services

import (
	"context"
	"net/http"
	"time"

	"github.com/DIMO-Network/accounts-api/internal/config"

	"github.com/Khan/genqlient/graphql"
	"github.com/ethereum/go-ethereum/common"
)

const (
	contentTypeHeaderKey = "Content-Type"
	jsonContentType      = "application/json"
)

var minimumConnections = 1

type IdentityService interface {
	AftermarketDevicesOwned(ctx context.Context, ethAddr common.Address) (bool, error)
	VehiclesOwned(ctx context.Context, ethAddr common.Address) (bool, error)
}

type identityAPI struct {
	client graphql.Client
}

func NewIdentityService(settings *config.Settings) IdentityService {
	graphqlClient := graphql.NewClient(settings.IdentityAPIURL, &http.Client{
		Timeout: 5 * time.Second,
	})
	return &identityAPI{
		client: graphqlClient,
	}
}

func (i *identityAPI) VehiclesOwned(ctx context.Context, ethAddr common.Address) (bool, error) {
	resp, err := vehicles(ctx, i.client, &minimumConnections, &VehiclesFilter{Owner: &ethAddr})
	if len(resp.Vehicles.GetNodes()) > 0 {
		return true, nil
	}

	return false, err
}

func (i *identityAPI) AftermarketDevicesOwned(ctx context.Context, ethAddr common.Address) (bool, error) {
	resp, err := aftermarketDevices(ctx, i.client, &minimumConnections, &AftermarketDevicesFilter{Owner: &ethAddr})
	if len(resp.AftermarketDevices.GetNodes()) > 0 {
		return true, nil
	}

	return false, err
}
