package builder_test

import (
	"testing"

	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"go.uber.org/zap/zaptest"
)

const (
	exampleRelayURL = "https://0x845bd072b7cd566f02faeb0a4033ce9399e42839ced64e8b2adcfc859ed1e8e1a5a293336a49feac6d9a5edb779be53a@builder-relay-sepolia.flashbots.net"
)

func TestClientStatus(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	c, err := builder.NewClient(exampleRelayURL, logger)
	if err != nil {
		t.Error(err)
		return
	}

	err = c.GetStatus()
	if err != nil {
		t.Error(err)
		return
	}
}
