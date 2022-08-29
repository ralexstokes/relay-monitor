package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/prysmaticlabs/prysm/v3/api/client/builder"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/helpers"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/ralexstokes/relay-monitor/pkg/api"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"go.uber.org/zap"
)

type NetworkConfig struct {
	Name           string `yaml:"name"`
	GenesisTime    uint64 `yaml:"genesis_time"`
	SlotsPerSecond uint64 `yaml:"slots_per_second"`
	SlotsPerEpoch  uint64 `yaml:"slots_per_epoch"`
}

type ConsensusConfig struct {
	Endpoint string `yaml:"endpoint"`
}

type Config struct {
	Network   *NetworkConfig   `yaml:"network"`
	Consensus *ConsensusConfig `yaml:"consensus"`
	Relays    []string         `yaml:"relays"`
	Api       *api.Config      `yaml:"api"`
}

type RelayMetrics struct {
	Bids uint `json:"bids"`
}

type Monitor struct {
	relays        []*builder.Client
	apiServer     *api.Server
	logger        *zap.Logger
	networkConfig *NetworkConfig

	relayMetrics     map[string]*RelayMetrics
	relayMetricsLock sync.Mutex

	clock           *slots.SlotTicker
	consensusClient *consensus.Client
}

func New(config *Config, zapLogger *zap.Logger) *Monitor {
	logger := zapLogger.Sugar()

	var relays []*builder.Client
	relayMetrics := make(map[string]*RelayMetrics)
	for _, endpoint := range config.Relays {
		relay, err := builder.NewClient(endpoint)
		if err != nil {
			logger.Warnf("could not instantiate relay at %s: %v", endpoint, err)
			continue
		}

		err = relay.Status(context.TODO())
		if err != nil {
			logger.Warnf("relay %s has status error: %v", endpoint, err)
			continue
		}

		relays = append(relays, relay)
		relayMetrics[relay.NodeURL()] = &RelayMetrics{}
	}

	genesisTime := time.Unix(int64(config.Network.GenesisTime), 0)
	slotsTicker := slots.NewSlotTicker(genesisTime, config.Network.SlotsPerEpoch)
	epochTicker := helpers.NewEpochTicker(genesisTime, config.Network.SlotsPerSecond*config.Network.SlotsPerEpoch)

	return &Monitor{
		relays:          relays,
		apiServer:       api.New(config.Api, zapLogger),
		logger:          zapLogger,
		networkConfig:   config.Network,
		relayMetrics:    relayMetrics,
		clock:           slotsTicker,
		consensusClient: consensus.NewClient(config.Consensus.Endpoint, slotsTicker, epochTicker, genesisTime, zapLogger),
	}
}

func (s *Monitor) monitorRelay(relay *builder.Client, wg *sync.WaitGroup) {
	logger := s.logger.Sugar()

	relayID := relay.NodeURL()
	logger.Infof("monitoring relay %s", relayID)

	for slot := range s.clock.C() {
		parentHash, err := s.consensusClient.GetParentHash(slot)
		if err != nil {
			logger.Warnw("error fetching bid", "error", err)
			continue
		}
		publicKey, err := s.consensusClient.GetProposerPublicKey(slot)
		if err != nil {
			logger.Warnw("error fetching bid", "error", err)
			continue
		}
		bid, err := relay.GetHeader(context.TODO(), slot, parentHash, *publicKey)
		if err != nil {
			logger.Warnw("could not get bid from relay", "error", err, "relayPublicKey", relayID, "slot", slot, "parentHash", parentHash, "proposer", publicKey)
		} else if bid != nil {
			s.relayMetricsLock.Lock()
			metrics := s.relayMetrics[relayID]
			metrics.Bids += 1
			s.relayMetricsLock.Unlock()
			logger.Debugw("got bid", "value", bid.Message.Value, "header", bid.Message.Header, "publicKey", bid.Message.Pubkey, "id", relayID)
		}
	}
	wg.Done()
}

func (s *Monitor) monitorRelays(wg *sync.WaitGroup) {
	for _, relay := range s.relays {
		wg.Add(1)
		go s.monitorRelay(relay, wg)
	}
	wg.Done()
}

func (s *Monitor) handleRelayMetricsRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)

	s.relayMetricsLock.Lock()
	defer s.relayMetricsLock.Unlock()

	logger := s.logger.Sugar()
	err := encoder.Encode(s.relayMetrics)
	if err != nil {
		logger.Errorw("could not encode relay metrics", "error", err)
	}
}

func (s *Monitor) serveApi() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/relay/metrics", s.handleRelayMetricsRequest)
	return s.apiServer.Run(mux)
}

func (s *Monitor) Run() {
	logger := s.logger.Sugar()

	logger.Infof("starting relay monitor for %s network", s.networkConfig.Name)

	// TODO error graph

	var wg sync.WaitGroup

	wg.Add(1)
	go s.consensusClient.Run(&wg)
	wg.Wait()

	wg.Add(1)
	go s.monitorRelays(&wg)

	go func() {
		err := s.serveApi()
		if err != nil {
			logger.Errorw("error serving api", "error", err)
		}
	}()

	wg.Wait()
}
