package monitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/api"
)

type Config struct {
	Relays []string    `yaml:"relays"`
	Api    *api.Config `yaml:"api"`
}

type Monitor struct {
	relays    []string
	apiServer *api.Server
}

func New(config *Config) *Monitor {
	return &Monitor{
		relays:    config.Relays,
		apiServer: api.New(config.Api),
	}
}

func (s *Monitor) watchRelays(wg *sync.WaitGroup) {
	for {
		for _, relay := range s.relays {
			fmt.Printf("watching %s\n", relay)
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
