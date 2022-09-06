package monitor

import "github.com/ralexstokes/relay-monitor/pkg/api"

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
