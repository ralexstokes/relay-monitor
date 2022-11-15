package builder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	boostTypes "github.com/flashbots/go-boost-utils/types"
	"github.com/ralexstokes/relay-monitor/pkg/types"
)

const clientTimeoutSec = 2

type Client struct {
	endpoint  string
	PublicKey types.PublicKey
	client    http.Client
}

func (c *Client) String() string {
	return c.PublicKey.String()
}

func NewClient(endpoint string) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	publicKeyStr := u.User.Username()
	var publicKey types.PublicKey
	err = publicKey.UnmarshalText([]byte(publicKeyStr))
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Timeout: clientTimeoutSec * time.Second,
	}
	return &Client{
		endpoint:  endpoint,
		PublicKey: publicKey,
		client:    client,
	}, nil
}

// GetStatus implements the `status` endpoint in the Builder API
func (c *Client) GetStatus() error {
	statusUrl := c.endpoint + "/eth/v1/builder/status"
	req, err := http.NewRequest(http.MethodGet, statusUrl, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("relay status was not healthy with HTTP status code %d", resp.StatusCode)
	}
	return nil
}

// GetBid implements the `getHeader` endpoint in the Builder API
func (c *Client) GetBid(slot types.Slot, parentHash types.Hash, publicKey types.PublicKey) (*types.Bid, bool, error) {
	bidUrl := c.endpoint + fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", slot, parentHash, publicKey)
	req, err := http.NewRequest(http.MethodGet, bidUrl, nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("failed to get bid with HTTP status code %d", resp.StatusCode)
	}

	var bid boostTypes.GetHeaderResponse
	err = json.NewDecoder(resp.Body).Decode(&bid)
	return bid.Data, true, err
}
