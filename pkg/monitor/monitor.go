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

type ConsensusConfig struct {
	Endpoint string `yaml:"endpoint"`
}

type Config struct {
	Network   *NetworkConfig   `yaml:"network"`
	Consensus *ConsensusConfig `yaml:"consensus"`
	Relays    []string         `yaml:"relays"`
	Api       *api.Config      `yaml:"api"`
}

type Monitor struct {
	relays        []*builder.Client
	apiServer     *api.Server
	logger        *zap.Logger
	networkConfig *NetworkConfig

	relayFaults     map[string]*RelayFaults
	relayFaultsLock sync.Mutex

	clock           *consensus.Clock
	consensusClient *consensus.Client
}

func New(config *Config, zapLogger *zap.Logger) *Monitor {
	logger := zapLogger.Sugar()

	var relays []*builder.Client
	relayFaults := make(map[string]*RelayFaults)
	for _, endpoint := range config.Relays {
		relay, err := builder.NewClient(endpoint)
		if err != nil {
			logger.Warnf("could not instantiate relay at %s: %v", endpoint, err)
			continue
		}

		err = relay.GetStatus()
		if err != nil {
			logger.Warnf("relay %s has status error: %v", endpoint, err)
			continue
		}

		relays = append(relays, relay)
		relayFaults[relay.ID()] = &RelayFaults{}
	}

	clock := consensus.NewClock(config.Network.GenesisTime, config.Network.SlotsPerSecond, config.Network.SlotsPerEpoch)
	return &Monitor{
		relays:          relays,
		apiServer:       api.New(config.Api, zapLogger),
		logger:          zapLogger,
		networkConfig:   config.Network,
		relayFaults:     relayFaults,
		clock:           clock,
		consensusClient: consensus.NewClient(config.Consensus.Endpoint, clock, zapLogger),
	}
}

func (s *Monitor) monitorRelay(relay *builder.Client, wg *sync.WaitGroup) {
	logger := s.logger.Sugar()

	relayID := relay.ID()
	logger.Infof("monitoring relay %s", relayID)

	for slot := range s.clock.TickSlots() {
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
		bid, err := relay.GetBid(slot, parentHash, *publicKey)
		if err != nil {
			logger.Warnw("could not get bid from relay", "error", err, "relayPublicKey", relayID, "slot", slot, "parentHash", parentHash, "proposer", publicKey)
		} else if bid != nil {
			s.relayFaultsLock.Lock()
			faults := s.relayFaults[relayID]
			faults.ValidBids += 1
			s.relayFaultsLock.Unlock()
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

func (s *Monitor) handleRelayFaultsRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)

	s.relayFaultsLock.Lock()
	defer s.relayFaultsLock.Unlock()

	logger := s.logger.Sugar()
	err := encoder.Encode(s.relayFaults)
	if err != nil {
		logger.Errorw("could not encode relay faults", "error", err)
	}
}

func (s *Monitor) serveApi() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/relay-monitor/faults", s.handleRelayFaultsRequest)
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
