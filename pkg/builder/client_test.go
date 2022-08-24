package builder_test

import (
	"fmt"
	"testing"

	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/types"
)

const (
	exampleRelayURL = "https://builder-relay-sepolia.flashbots.net"
)

func TestClientStatus(t *testing.T) {
	c := builder.New(exampleRelayURL)
	err := c.GetStatus()
	if err != nil {
		t.Error(err)
	}
}

func TestClientBid(t *testing.T) {
	c := builder.New(exampleRelayURL)
	bid, err := c.GetBid(100, types.Hash{}, types.PublicKey{})
	if err != nil {
		t.Error(err)
	}
	fmt.Println(bid)
}
