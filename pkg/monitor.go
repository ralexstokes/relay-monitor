package monitor

import (
	"sync"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/api"
	"go.uber.org/zap"
)

type Config struct {
	Relays []string    `yaml:"relays"`
	Api    *api.Config `yaml:"api"`
}

type Monitor struct {
	relays    []string
	apiServer *api.Server
	logger    *zap.Logger
}

func New(config *Config, logger *zap.Logger) *Monitor {
	return &Monitor{
		relays:    config.Relays,
		apiServer: api.New(config.Api, logger),
		logger:    logger,
	}
}

func (s *Monitor) watchRelays(wg *sync.WaitGroup) {
	logger := s.logger.Sugar()
	for {
		for _, relay := range s.relays {
			logger.Infow("watching relay", "endpoint", relay)
		}

		time.Sleep(3 * time.Second)
	}
	wg.Done()
}

func (s *Monitor) serveApi(wg *sync.WaitGroup) error {
	return s.apiServer.Run(wg)
}

func (s *Monitor) Run() {
	var wg sync.WaitGroup

	// TODO error graph

	wg.Add(1)
	go s.watchRelays(&wg)

	wg.Add(1)
	go s.serveApi(&wg)

	wg.Wait()
}
