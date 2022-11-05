package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/ralexstokes/relay-monitor/pkg/monitor"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var configFile = flag.String("config", "config.example.yaml", "path to config file")

func main() {
	flag.Parse()

	loggingConfig := zap.NewDevelopmentConfig()
	loggingConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	zapLogger, err := loggingConfig.Build()
	if err != nil {
		log.Fatalf("could not open log file: %v", err)
	}

	logger := zapLogger.Sugar()

	data, err := os.ReadFile(*configFile)
	if err != nil {
		logger.Fatalf("could not read config file: %v", err)
	}

	config := &monitor.Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		logger.Fatalf("could not load config: %v", err)
	}

	ctx := context.Background()
	logger.Infof("starting relay monitor for %s network", config.Network.Name)
	m, err := monitor.New(ctx, config, zapLogger)
	if err != nil {
		logger.Fatalf("could not start relay monitor: %v", err)
	}

	m.Run(ctx)

	defer func() {
		err := zapLogger.Sync()
		if err != nil {
			log.Fatalf("could not flush log: %v", err)
		}

		m.Stop()
	}()
}
