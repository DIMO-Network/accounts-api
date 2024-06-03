package services

import (
	"accounts-api/internal/config"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const (
	contentTypeHeaderKey = "Content-Type"
	jsonContentType      = "application/json"
)

type IdentityService interface {
	AftermarketDevicesOwned(ctx context.Context, ethAddr common.Address) (bool, error)
	VehiclesOwned(ctx context.Context, ethAddr common.Address) (bool, error)
}

type identityAPI struct {
	url    string
	client *http.Client
}

func NewIdentityService(settings *config.Settings) IdentityService {
	return &identityAPI{
		url: settings.IdentityAPIURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		}}
}

func (i *identityAPI) VehiclesOwned(ctx context.Context, ethAddr common.Address) (bool, error) {
	payloadBytes, err := json.Marshal(GraphQLRequest{
		Query: fmt.Sprintf(`{vehicles(first: 1,filterBy:{owner: "%s"}){nodes{tokenId}}}`, ethAddr.Hex()),
	})
	if err != nil {
		return false, err
	}

	reader := bytes.NewReader(payloadBytes)
	req, err := http.NewRequest(http.MethodPost, i.url, reader)
	if err != nil {
		return false, err
	}

	req.Header.Set(contentTypeHeaderKey, jsonContentType)
	resp, err := i.client.Do(req.WithContext(ctx))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var id IdentityVehicleResponse
	if err := json.Unmarshal(body, &id); err != nil {
		return false, err
	}

	if len(id.Data.Vehicles.Nodes) > 0 {
		return true, nil
	}

	return false, err
}

func (i *identityAPI) AftermarketDevicesOwned(ctx context.Context, ethAddr common.Address) (bool, error) {
	payloadBytes, err := json.Marshal(GraphQLRequest{
		Query: fmt.Sprintf(`{aftermarketDevices(first: 1,filterBy:{owner: "%s"}){nodes{tokenId}}}`, ethAddr.Hex()),
	})
	if err != nil {
		return false, err
	}

	reader := bytes.NewReader(payloadBytes)
	req, err := http.NewRequest(http.MethodPost, i.url, reader)
	if err != nil {
		return false, err
	}

	req.Header.Set(contentTypeHeaderKey, jsonContentType)
	resp, err := i.client.Do(req.WithContext(ctx))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var id IdentityAftermarketDeviceResponse
	if err := json.Unmarshal(body, &id); err != nil {
		return false, err
	}

	if len(id.Data.AftermarketDevices.Nodes) > 0 {
		return true, nil
	}

	return false, err
}

type GraphQLRequest struct {
	Query string `json:"query"`
}

type IdentityVehicleResponse struct {
	Data struct {
		Vehicles struct {
			Nodes []struct {
				TokenID int `json:"tokenId"`
			} `json:"nodes"`
		} `json:"vehicles"`
	} `json:"data"`
}

type IdentityAftermarketDeviceResponse struct {
	Data struct {
		AftermarketDevices struct {
			Nodes []struct {
				TokenID int `json:"tokenId"`
			} `json:"nodes"`
		} `json:"aftermarketDevices"`
	} `json:"data"`
}
