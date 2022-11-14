package monitor

import "github.com/ralexstokes/relay-monitor/pkg/api"

type NetworkConfig struct {
	Name               string `yaml:"name"`
	GenesisForkVersion uint32 `yaml:"genesis_fork_version"`
	GenesisTime        uint64 `yaml:"genesis_time"`
	SecondsPerSlot     uint64 `yaml:"seconds_per_slot"`
	SlotsPerEpoch      uint64 `yaml:"slots_per_epoch"`
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
