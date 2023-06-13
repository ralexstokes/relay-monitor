package monitor

import (
	"github.com/ralexstokes/relay-monitor/pkg/api"
)

type NetworkConfig struct {
	Name string `yaml:"name"`
}

type ConsensusConfig struct {
	Endpoint string `yaml:"endpoint"`
}

type OutputConfig struct {
	Path string `yaml:"path"`
}

type Config struct {
	Network   *NetworkConfig   `yaml:"network"`
	Consensus *ConsensusConfig `yaml:"consensus"`
	Relays    []string         `yaml:"relays"`
	Api       *api.Config      `yaml:"api"`
	Output    *OutputConfig    `yaml:"output"`
	Region    string           `yaml:"region"`
}
