package monitor

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/api"
	"github.com/ralexstokes/relay-monitor/pkg/relay"
	"go.uber.org/zap"
)

type NetworkConfig struct {
	Name string `yaml:"name"`
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
	relays        []*relay.Client
	apiServer     *api.Server
	logger        *zap.Logger
	networkConfig *NetworkConfig

	relayMetrics     map[string]*RelayMetrics `json:"relay_metrics"`
	relayMetricsLock sync.Mutex
}

func New(config *Config, zapLogger *zap.Logger) *Monitor {
	logger := zapLogger.Sugar()

	relays := []*relay.Client{}
	relayMetrics := make(map[string]*RelayMetrics)
	for _, endpoint := range config.Relays {
		relay, err := relay.New(endpoint)
		if err != nil {
			logger.Warnf("could not instantiate relay: %s", err)
			continue
		}

		relays = append(relays, relay)
		relayMetrics[relay.PublicKey()] = &RelayMetrics{}
	}

	return &Monitor{
		relays:        relays,
		apiServer:     api.New(config.Api, zapLogger),
		logger:        zapLogger,
		networkConfig: config.Network,
		relayMetrics:  relayMetrics,
	}
}

func (s *Monitor) monitorRelay(relay *relay.Client, wg *sync.WaitGroup) {
	logger := s.logger.Sugar()
	publicKey := relay.PublicKey()
	for {
		_, err := relay.FetchBid()
		if err != nil {
			logger.Warnf("could not get bid from relay %s", relay)
		} else {
			s.relayMetricsLock.Lock()
			s.relayMetrics[publicKey].Bids += 1
			s.relayMetricsLock.Unlock()
		}
		logger.Debugw("polling relay", "endpoint", relay)
		time.Sleep(12 * time.Second)
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
	for _, relay := range s.relays {
		logger.Infof("monitoring relay at %s", relay)
	}

	// TODO error graph

	var wg sync.WaitGroup
	wg.Add(1)
	go s.monitorRelays(&wg)

	go s.serveApi()

	wg.Wait()
}
