package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/analysis"
	"github.com/ralexstokes/relay-monitor/pkg/api"
	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"github.com/ralexstokes/relay-monitor/pkg/store"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

const eventBufferSize uint = 32

type Monitor struct {
	logger *zap.Logger

	api       *api.Server
	collector *data.Collector
	analyzer  *analysis.Analyzer
}

func parseRelaysFromEndpoint(ctx context.Context, store *store.PostgresStore, relayEndpoints []string, logger *zap.SugaredLogger) []*builder.Client {
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

		// Save relay to DB if it doesn't exist.
		err = store.PutRelay(ctx, &types.Relay{
			Pubkey:   relay.PublicKey.String(),
			Hostname: relay.Hostname(),
			Endpoint: relay.Endpoint(),
		})
		if err != nil {
			logger.Warnf("could not save relay %s to DB: %v", endpoint, err)
			continue
		}

		relays = append(relays, relay)
	}
	if len(relays) == 0 {
		logger.Warn("could not parse any relays, please check configuration")
	}
	return relays
}

func New(ctx context.Context, config *Config, zapLogger *zap.Logger) (*Monitor, error) {
	logger := zapLogger.Sugar()

	// Create the store.
	store, err := store.NewPostgresStore(config.Store.Dsn, zapLogger)
	if err != nil {
		logger.Fatal("could not instantiate postgres store", zap.Error(err))
	}

	// Parse the relays from the config.
	relays := parseRelaysFromEndpoint(ctx, store, config.Relays, logger)

	// Instantiate the consensus client.
	consensusClient, err := consensus.NewClient(ctx, config.Consensus.Endpoint, zapLogger)
	if err != nil {
		return nil, fmt.Errorf("could not instantiate consensus client: %v", err)
	}

	// Handle the timing: get the current slot and epoch.
	clock := consensus.NewClock(consensusClient.GenesisTime, consensusClient.SecondsPerSlot, consensusClient.SlotsPerEpoch)
	now := time.Now().Unix()
	currentSlot := clock.CurrentSlot(now)
	currentEpoch := clock.EpochForSlot(currentSlot)

	err = consensusClient.LoadCurrentContext(ctx, currentSlot, currentEpoch)
	if err != nil {
		logger.Warn("could not load the current context from the consensus client")
	}

	// Instantiate the data collector.
	events := make(chan data.Event, eventBufferSize)
	collector := data.NewCollector(zapLogger, relays, clock, consensusClient, events)

	// Instantiate the analysis.
	analyzer := analysis.NewAnalyzer(zapLogger, relays, events, store, consensusClient, clock)

	// Instantiate the API server.
	apiServer := api.New(config.Api, zapLogger, analyzer, events, clock, store, consensusClient)

	return &Monitor{
		logger:    zapLogger,
		api:       apiServer,
		collector: collector,
		analyzer:  analyzer,
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
