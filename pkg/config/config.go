package config

import (
	"time"
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

type KafkaConfig struct {
	Topic               string        `yaml:"topic"`
	BootstrapServersStr string        `yaml:"bootstrap_servers"`
	BootstrapServers    []string      `yaml:"-"`
	Timeout             time.Duration `yaml:"timeout"`
}

type ApiConfig struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type Config struct {
	Network   *NetworkConfig   `yaml:"network"`
	Consensus *ConsensusConfig `yaml:"consensus"`
	Relays    []string         `yaml:"relays"`
	Api       *ApiConfig       `yaml:"api"`
	Output    *OutputConfig    `yaml:"output"`
	Region    string           `yaml:"region"`
	Kafka     *KafkaConfig     `yaml:"kafka"`
}
