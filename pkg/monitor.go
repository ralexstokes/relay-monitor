package monitor

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/ralexstokes/relay-monitor/pkg/api"
	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"go.uber.org/zap"
)

type NetworkConfig struct {
	Name           string `yaml:"name"`
	GenesisTime    uint64 `yaml:"genesis_time"`
	SlotsPerSecond uint64 `yaml:"slots_per_second"`
	SlotsPerEpoch  uint64 `yaml:"slots_per_epoch"`
}

type Config struct {
	Network *NetworkConfig `yaml:"network"`
	Relays  []string       `yaml:"relays"`
	Api     *api.Config    `yaml:"api"`
}

type RelayMetrics struct {
	Bids uint `json:"bids"`
}

type Monitor struct {
	relays        []*builder.Client
	apiServer     *api.Server
	logger        *zap.Logger
	networkConfig *NetworkConfig

	relayMetrics     map[string]*RelayMetrics `json:"relay_metrics"`
	relayMetricsLock sync.Mutex

	clock           *consensus.Clock
	consensusClient *consensus.Client
}

func New(config *Config, zapLogger *zap.Logger) *Monitor {
	logger := zapLogger.Sugar()

	relays := []*builder.Client{}
	relayMetrics := make(map[string]*RelayMetrics)
	for _, endpoint := range config.Relays {
		relay, err := builder.NewClient(endpoint)
		if err != nil {
			logger.Warnf("could not instantiate relay at %s: %s", endpoint, err)
			continue
		}

		err = relay.GetStatus()
		if err != nil {
			logger.Warnf("relay %s has status error: %s", endpoint, err)
			continue
		}

		relays = append(relays, relay)
		relayMetrics[relay.ID()] = &RelayMetrics{}
	}

	return &Monitor{
		relays:          relays,
		apiServer:       api.New(config.Api, zapLogger),
		logger:          zapLogger,
		networkConfig:   config.Network,
		relayMetrics:    relayMetrics,
		clock:           consensus.NewClock(config.Network.GenesisTime, config.Network.SlotsPerSecond, config.Network.SlotsPerEpoch),
		consensusClient: consensus.NewClient(""),
	}
}

func (s *Monitor) monitorRelay(relay *builder.Client, wg *sync.WaitGroup) {
	logger := s.logger.Sugar()

	relayID := relay.ID()
	logger.Infof("monitoring relay %s", relayID)

	for slot := range s.clock.TickSlots() {
		parentHash := s.consensusClient.GetParentHash(slot)
		publicKey := s.consensusClient.GetProposerPublicKey(slot)
		bid, err := relay.GetBid(slot, parentHash, publicKey)
		if err != nil {
			logger.Warnf("could not get bid from relay %s", relay)
		} else {
			s.relayMetricsLock.Lock()
			metrics := s.relayMetrics[relayID]
			metrics.Bids += 1
			s.relayMetricsLock.Unlock()
			logger.Debugf("got bid: %v", bid)
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

	encoder.Encode(s.relayMetrics)
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
	go s.monitorRelays(&wg)

	go s.serveApi()

	wg.Wait()
}
