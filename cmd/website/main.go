package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/flashbots/mev-boost-relay/common"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/reporter"
	"github.com/ralexstokes/relay-monitor/pkg/store"
	"github.com/ralexstokes/relay-monitor/pkg/website"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

var (
	configFile = flag.String("config", "config.website.example.yaml", "path to website config file")
)

func main() {
	flag.Parse()

	loggingConfig := zap.NewDevelopmentConfig()
	loggingConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	zapLogger, err := loggingConfig.Build()
	if err != nil {
		log.Fatalf("could not open log file: %v", err)
	}
	defer func() {
		err := zapLogger.Sync()
		if err != nil {
			log.Fatalf("could not flush log: %v", err)
		}
	}()
	logger := zapLogger.Sugar()

	data, err := os.ReadFile(*configFile)
	if err != nil {
		logger.Fatalf("could not read config file: %v", err)
	}

	config := &website.Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		logger.Fatalf("could not load config: %v", err)
	}

	networkInfo, err := common.NewEthNetworkDetails(config.Network.Name)
	if err != nil {
		logger.Fatalf("error getting network details")
	}

	logger.Infof("using network: %s", networkInfo.Name)

	// Create the store.
	store, err := store.NewPostgresStore(config.Store.Dsn, zapLogger)
	if err != nil {
		logger.Fatal("could not instantiate postgres store", zap.Error(err))
	}

	// Create the consensus client.
	consensusClient, err := consensus.NewClient(context.Background(), config.Consensus.Endpoint, zapLogger)
	if err != nil {
		logger.Fatal("could not instantiate consensus client", zap.Error(err))
	}
	clock := consensus.NewClock(consensusClient.GenesisTime, consensusClient.SecondsPerSlot, consensusClient.SlotsPerEpoch)

	// Create the reporter.
	reporter := reporter.NewReporter(store, reporter.NewScorer(clock, logger), logger)

	websiteListenAddr := fmt.Sprintf("%s:%d", config.Website.Host, config.Website.Port)

	// Create the website service
	opts := &website.WebserverOpts{
		ListenAddress:       websiteListenAddr,
		NetworkDetails:      networkInfo,
		Store:               store,
		Reporter:            reporter,
		Clock:               clock,
		ShowConfigDetails:   config.Website.ShowConfigDetails,
		LinkBeaconchain:     config.Website.LinkBeaconchain,
		LinkEtherscan:       config.Website.LinkEtherscan,
		LinkRelayMonitorAPI: config.Website.LinkRelayMonitorAPI,
		LookbackSlotsValue:  7200, // 24 hours
		Log:                 logger,
	}

	srv, err := website.NewWebserver(opts)
	if err != nil {
		logger.Fatal("failed to create service", zap.Error(err))
	}

	// Start the server
	logger.Infof("webserver starting on %s ...", websiteListenAddr)
	logger.Fatal(srv.StartServer())
}
