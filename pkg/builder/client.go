package builder

import (
	"encoding/hex"
	"net/url"

	"github.com/ralexstokes/relay-monitor/pkg/types"
	"github.com/umbracle/go-eth-consensus/http"
)

type Client struct {
	endpoint  string
	PublicKey types.PublicKey
	client    *http.BuilderEndpoint
}

func (c *Client) String() string {
	return hex.EncodeToString(c.PublicKey[:])
}

func NewClient(endpoint string) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	var pub types.PublicKey
	buf, err := hex.DecodeString(u.User.Username()[2:])
	if err != nil {
		return nil, err
	}
	copy(pub[:], buf)

	return &Client{
		endpoint:  endpoint,
		PublicKey: pub,
	}, nil
}

// GetStatus implements the `status` endpoint in the Builder API
func (c *Client) GetStatus() (bool, error) {
	return c.client.Status()
}

// GetBid implements the `getHeader` endpoint in the Builder API
func (c *Client) GetBid(slot types.Slot, parentHash types.Hash, publicKey types.PublicKey) (*http.SignedBuilderBid, error) {
	return c.client.GetExecutionPayload(slot, parentHash, publicKey)
}
