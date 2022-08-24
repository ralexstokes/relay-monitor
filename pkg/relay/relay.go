package relay

import "net/url"

type Client struct {
	endpoint  string
	publicKey string
}

func New(endpoint string) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	publicKey := u.User.Username()

	return &Client{
		endpoint:  endpoint,
		publicKey: publicKey,
	}, nil
}

func (c *Client) PublicKey() string {
	return c.publicKey
}

func (c *Client) String() string {
	return c.endpoint
}

func (c *Client) FetchBid() (string, error) {
	return "", nil
}
