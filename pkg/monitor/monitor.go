package monitor

import (
	"context"

	"github.com/ralexstokes/relay-monitor/pkg/analysis"
	"github.com/ralexstokes/relay-monitor/pkg/api"
	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"go.uber.org/zap"
)

const eventBufferSize uint = 32

type Monitor struct {
	logger        *zap.Logger
	networkConfig *NetworkConfig

	api       *api.Server
	collector *data.Collector
	analyzer  *analysis.Analyzer
}

func parseRelaysFromEndpoint(logger *zap.SugaredLogger, relayEndpoints []string) []*builder.Client {
	var relays []*builder.Client
	for _, endpoint := range relayEndpoints {
		relay, err := builder.NewClient(endpoint)
		if err != nil {
			logger.Warnf("could not instantiate relay at %s: %v", endpoint, err)
			continue
		}

		err = relay.GetStatus()
		if err != nil {
			logger.Warnf("relay %s has status error: %v", endpoint, err)
			continue
		}

		relays = append(relays, relay)
	}
	if len(relays) == 0 {
		logger.Warn("could not parse any relays, please check configuration")
	}
	return relays
}

func New(config *Config, zapLogger *zap.Logger) *Monitor {
	logger := zapLogger.Sugar()

	relays := parseRelaysFromEndpoint(logger, config.Relays)
	clock := consensus.NewClock(config.Network.GenesisTime, config.Network.SlotsPerSecond, config.Network.SlotsPerEpoch)
	consensusClient := consensus.NewClient(config.Consensus.Endpoint, clock, zapLogger)
	events := make(chan data.Event, eventBufferSize)
	collector := data.NewCollector(zapLogger, relays, clock, consensusClient, events)
	analyzer := analysis.NewAnalyzer(zapLogger, relays, events)
	apiServer := api.New(config.Api, zapLogger, analyzer, events)
	return &Monitor{
		logger:        zapLogger,
		networkConfig: config.Network,
		api:           apiServer,
		collector:     collector,
		analyzer:      analyzer,
	}
}

func (s *Monitor) Run(ctx context.Context) {
	logger := s.logger.Sugar()

	logger.Infof("starting relay monitor for %s network", s.networkConfig.Name)

	go func() {
		err := s.collector.Run(ctx)
		if err != nil {
			logger.Warn("error running collector: %v", err)
		}
	}()
	go func() {
		err := s.analyzer.Run(ctx)
		if err != nil {
			logger.Warn("error running collector: %v", err)
		}
	}()

	err := s.api.Run(ctx)
	if err != nil {
		logger.Warn("error running API server: %v", err)
	}
}
