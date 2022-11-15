package monitor

import (
	"encoding/binary"

	"github.com/ralexstokes/relay-monitor/pkg/api"
	"github.com/ralexstokes/relay-monitor/pkg/crypto"
	"github.com/ralexstokes/relay-monitor/pkg/types"
)

type NetworkConfig struct {
	Name               string `yaml:"name"`
	GenesisForkVersion uint32 `yaml:"genesis_fork_version"`
	GenesisTime        uint64 `yaml:"genesis_time"`
	SecondsPerSlot     uint64 `yaml:"seconds_per_slot"`
	SlotsPerEpoch      uint64 `yaml:"slots_per_epoch"`
}

func (n *NetworkConfig) SignatureDomain() crypto.Domain {
	genesisForkVersion := [4]byte{}
	binary.BigEndian.PutUint32(genesisForkVersion[0:4], n.GenesisForkVersion)
	return crypto.ComputeDomain(crypto.DomainTypeAppBuilder, genesisForkVersion, types.Root{})
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
