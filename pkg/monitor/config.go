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

type ExecutionConfig struct {
	Endpoint string `yaml:"endpoint"`
}

type Config struct {
	Network   *NetworkConfig   `yaml:"network"`
	Consensus *ConsensusConfig `yaml:"consensus"`
	Execution *ExecutionConfig `yaml:"execution"`
	Relays    []string         `yaml:"relays"`
	Api       *api.Config      `yaml:"api"`
}
