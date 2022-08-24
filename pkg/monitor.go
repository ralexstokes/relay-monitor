package monitor

import (
	"sync"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/api"
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

type Monitor struct {
	relays        []string
	apiServer     *api.Server
	logger        *zap.Logger
	networkConfig *NetworkConfig
}

func New(config *Config, logger *zap.Logger) *Monitor {
	return &Monitor{
		relays:        config.Relays,
		apiServer:     api.New(config.Api, logger),
		logger:        logger,
		networkConfig: config.Network,
	}
}

func (s *Monitor) watchRelays(wg *sync.WaitGroup) {
	logger := s.logger.Sugar()
	for {
		for _, relay := range s.relays {
			logger.Debugw("watching relay", "endpoint", relay)
		}

		time.Sleep(3 * time.Second)
	}
	wg.Done()
}

func (s *Monitor) serveApi(wg *sync.WaitGroup) error {
	return s.apiServer.Run(wg)
}

func (s *Monitor) Run() {
	logger := s.logger.Sugar()

	logger.Infow("starting relay monitor", "network", s.networkConfig.Name, "relays", s.relays)

	var wg sync.WaitGroup

	// TODO error graph

	wg.Add(1)
	go s.watchRelays(&wg)

	wg.Add(1)
	go s.serveApi(&wg)

	wg.Wait()
}
