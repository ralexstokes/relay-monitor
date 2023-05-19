package monitor

import (
	"github.com/ralexstokes/relay-monitor/pkg/api"
)

type StoreConfig struct {
	Dsn string `yaml:"dsn"`
}

type NetworkConfig struct {
	Name string `yaml:"name"`
}

type ConsensusConfig struct {
	Endpoint string `yaml:"endpoint"`
}

type Config struct {
	Network   *NetworkConfig   `yaml:"network"`
	Consensus *ConsensusConfig `yaml:"consensus"`
	Relays    []string         `yaml:"relays"`
	Api       *api.Config      `yaml:"api"`
	Store     *StoreConfig     `yaml:"store"`
}
