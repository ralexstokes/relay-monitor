package builder

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

type Client struct {
	endpoint string
	client   http.Client
}

func New(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
	}
}

// `status` endpoint in the Builder API
func (c *Client) GetStatus() error {
	url := c.endpoint + "/eth/v1/builder/status"
	req, err := http.NewRequest(http.MethodGet, url, nil)
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

// `getHeader` endpoint in the Builder API
func (c *Client) GetBid(slot types.Slot, parentHash types.Hash, publicKey types.PublicKey) (*types.Bid, error) {
	url := c.endpoint + fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", slot, parentHash, publicKey)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get bid with HTTP status code %d", resp.StatusCode)
	}

	var bid types.Bid
	err = json.NewDecoder(resp.Body).Decode(&bid)
	return &bid, err
}
