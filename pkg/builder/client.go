package builder

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/ralexstokes/relay-monitor/pkg/metrics"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

const clientTimeoutSec = 2

type Client struct {
	logger    *zap.SugaredLogger
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

func NewClient(endpoint string, logger *zap.SugaredLogger) (*Client, error) {
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
		logger:    logger,
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
	t := prometheus.NewTimer(metrics.GetBid)
	defer t.ObserveDuration()

	bidUrl := c.endpoint + fmt.Sprintf("/eth/v1/builder/header/%d/%s/%s", slot, parentHash, publicKey)
	req, err := http.NewRequest(http.MethodGet, bidUrl, nil)
	if err != nil {
		return nil, 0, &types.ClientError{Type: types.RelayError, Code: 500, Message: err.Error()}
	}
	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		return nil, 0, &types.ClientError{Type: types.RelayError, Code: 500, Message: err.Error()}
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, uint64(duration), nil
	}
	if resp.StatusCode != http.StatusOK {
		rspBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			c.logger.Debugw("failed to read response body", zap.Error(err))
			return nil, uint64(duration), err
		}

		errorMsg := &types.ClientError{}
		err = json.Unmarshal(rspBytes, errorMsg)
		if err != nil {
			c.logger.Debug("failed to unmarshal response body", "body", string(rspBytes), zap.Error(err))
			return nil, uint64(duration), &types.ClientError{Type: types.RelayError, Code: resp.StatusCode, Message: "Unable to parse relay response"}
		}

		return nil, uint64(duration), &types.ClientError{Type: types.RelayError, Code: resp.StatusCode, Message: errorMsg.Message}
	}

	var bid types.GetHeaderResponse
	err = json.NewDecoder(resp.Body).Decode(&bid)
	if err != nil {
		return bid, uint64(duration), &types.ClientError{Type: types.RelayError, Code: 500, Message: err.Error()}
	}
	return bid, uint64(duration), err
}
