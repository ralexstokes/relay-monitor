package builder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

func (c *Client) Endpoint() string {
	return c.endpoint
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
func (c *Client) GetBid(slot types.Slot, parentHash types.Hash, publicKey types.PublicKey) (*types.Bid, uint64, error) {
	bidUrl := c.endpoint + fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", slot, parentHash, publicKey)
	req, err := http.NewRequest(http.MethodGet, bidUrl, nil)
	if err != nil {
		return nil, 0, err
	}
	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		rspBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("failed to get bid with HTTP status code %d, body: %s", resp.StatusCode, string(rspBytes))
	}

	var bid boostTypes.GetHeaderResponse
	err = json.NewDecoder(resp.Body).Decode(&bid)
	return bid.Data, uint64(duration), err
}
