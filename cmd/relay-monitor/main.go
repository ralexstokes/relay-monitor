package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/config"
	"github.com/ralexstokes/relay-monitor/pkg/monitor"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var (
	configFile          = flag.String("config", "config.example.yaml", "path to config file")
	defaultKafkaTimeout = time.Second * 10
)

func main() {
	flag.Parse()

	loggingConfig := zap.NewDevelopmentConfig()
	loggingConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	zapLogger, err := loggingConfig.Build()
	if err != nil {
		log.Fatalf("could not open log file: %v", err)
	}

	logger := zapLogger.Sugar()
	zap.ReplaceGlobals(zapLogger)

	data, err := os.ReadFile(*configFile)
	if err != nil {
		logger.Fatalf("could not read config file: %v", err)
	}

	appConf := &config.Config{}
	err = yaml.Unmarshal(data, appConf)
	if err != nil {
		logger.Fatalf("could not load config: %v", err)
	}

	// parse bootstrap servers as CSV
	if appConf.Kafka != nil {
		appConf.Kafka.BootstrapServers = strings.Split(appConf.Kafka.BootstrapServersStr, ",")
		if appConf.Kafka.Timeout == 0 {
			appConf.Kafka.Timeout = defaultKafkaTimeout
		}
	}

	ctx := context.Background()
	logger.Infof("starting relay monitor for %s network", appConf.Network.Name)
	m, err := monitor.New(ctx, appConf, zapLogger)
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
