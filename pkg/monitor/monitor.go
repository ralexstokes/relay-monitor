package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/analysis"
	"github.com/ralexstokes/relay-monitor/pkg/api"
	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/config"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"github.com/ralexstokes/relay-monitor/pkg/output"
	"github.com/ralexstokes/relay-monitor/pkg/store"
	"go.uber.org/zap"
)

const eventBufferSize uint = 32

type Monitor struct {
	logger *zap.Logger

	api       *api.Server
	collector *data.Collector
	analyzer  *analysis.Analyzer
	output    *output.Output
}

func parseRelaysFromEndpoint(logger *zap.SugaredLogger, relayEndpoints []string) []*builder.Client {
	var relays []*builder.Client
	for _, endpoint := range relayEndpoints {
		relay, err := builder.NewClient(endpoint, logger)
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

func New(ctx context.Context, appConf *config.Config, zapLogger *zap.Logger) (*Monitor, error) {
	logger := zapLogger.Sugar()

	fileOutput, err := output.NewFileOutput(ctx, appConf.Output.Path, appConf.Kafka)
	if err != nil {
		return nil, fmt.Errorf("could not create output file: %v", err)
	}

	relays := parseRelaysFromEndpoint(logger, appConf.Relays)

	consensusClient, err := consensus.NewClient(ctx, appConf.Consensus.Endpoint, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("could not instantiate consensus client: %v", err)
	}

	clock := consensus.NewClock(consensusClient.GenesisTime, consensusClient.SecondsPerSlot, consensusClient.SlotsPerEpoch)
	now := time.Now().Unix()

	// Start with the last slot for stability
	currentSlot := clock.CurrentSlot(now) - 1
	currentEpoch := clock.EpochForSlot(currentSlot)

	err = consensusClient.LoadCurrentContext(ctx, currentSlot, currentEpoch)
	if err != nil {
		logger.Panic("could not load the current context from the consensus client")
	}

	events := make(chan data.Event, eventBufferSize)
	collector := data.NewCollector(zapLogger, relays, clock, consensusClient, fileOutput, appConf.Region, events)
	store := store.NewMemoryStore()
	analyzer := analysis.NewAnalyzer(zapLogger, relays, events, store, consensusClient, clock, fileOutput, appConf.Region)

	apiServer := api.New(appConf.Api, zapLogger, analyzer, events, clock, store, consensusClient)
	return &Monitor{
		logger:    zapLogger,
		api:       apiServer,
		collector: collector,
		analyzer:  analyzer,
		output:    fileOutput,
	}, nil
}

func (s *Monitor) Run(ctx context.Context) {
	logger := s.logger.Sugar()

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

func (s *Monitor) Stop() {
	logger := s.logger.Sugar()
	logger.Info("Shutting down monitor...")

	s.output.Close()
}
