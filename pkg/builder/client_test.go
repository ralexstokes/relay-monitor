package builder_test

import (
	"testing"

	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/types"
)

const (
	exampleRelayURL = "https://builder-relay-sepolia.flashbots.net"
)

func TestClientStatus(t *testing.T) {
	c, err := builder.NewClient(exampleRelayURL)
	if err != nil {
		t.Error(err)
	}

	err = c.GetStatus()
	if err != nil {
		t.Error(err)
	}
}

func TestClientBid(t *testing.T) {
	c, err := builder.NewClient(exampleRelayURL)
	if err != nil {
		t.Error(err)
	}

	_, err = c.GetBid(100, types.Hash{}, types.PublicKey{})
	if err != nil {
		t.Error(err)
	}
}
