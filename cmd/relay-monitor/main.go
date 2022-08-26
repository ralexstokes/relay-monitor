package main

import (
	"flag"
	"log"
	"os"

	monitor "github.com/ralexstokes/relay-monitor/pkg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var (
	configFile = flag.String("config", "config.example.yaml", "path to config file")
)

func main() {
	flag.Parse()

	loggingConfig := zap.NewDevelopmentConfig()
	loggingConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	zapLogger, err := loggingConfig.Build()
	if err != nil {
		log.Fatalf("could not open log file")
	}
	defer zapLogger.Sync()

	logger := zapLogger.Sugar()

	data, err := os.ReadFile(*configFile)
	if err != nil {
		logger.Fatalf("could not read config file: %s", err)
	}

	config := &monitor.Config{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		logger.Fatalf("could not load config: %s", err)
	}

	m := monitor.New(config, zapLogger)
	m.Run()
}
