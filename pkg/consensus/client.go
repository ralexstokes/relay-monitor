package consensus

import "github.com/ralexstokes/relay-monitor/pkg/types"

type Client struct {
}

func NewClient(endpoint string) *Client {
	return &Client{}
}

func (c *Client) GetParentHash(slot types.Slot) types.Hash {
	return types.Hash{}
}

func (c *Client) GetProposerPublicKey(slot types.Slot) types.PublicKey {
	return types.PublicKey{}
}
