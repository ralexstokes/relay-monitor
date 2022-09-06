package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ralexstokes/relay-monitor/pkg/analysis"
	"go.uber.org/zap"
)

const GetFaultEndpoint = "/api/v1/relay-monitor/faults"

type Config struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type Server struct {
	config *Config
	logger *zap.Logger

	analyzer *analysis.Analyzer
}

func New(config *Config, logger *zap.Logger, analyzer *analysis.Analyzer) *Server {
	return &Server{
		config:   config,
		logger:   logger,
		analyzer: analyzer,
	}
}

func (s *Server) handleFaultsRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)

	faults := s.analyzer.GetFaults()
	err := encoder.Encode(faults)
	if err != nil {
		logger := s.logger.Sugar()
		logger.Errorw("could not encode relay faults", "error", err)
	}
}

func (s *Server) Run(ctx context.Context) error {
	logger := s.logger.Sugar()
	host := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	logger.Infof("API server listening on %s", host)

	mux := http.NewServeMux()
	mux.HandleFunc(GetFaultEndpoint, s.handleFaultsRequest)
	return http.ListenAndServe(host, mux)
}
