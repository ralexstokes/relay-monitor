package api

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

type Config struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type Server struct {
	config *Config
	logger *zap.Logger
}

func New(config *Config, logger *zap.Logger) *Server {
	return &Server{
		config: config,
		logger: logger,
	}
}

func (s *Server) Run(mux *http.ServeMux) error {
	logger := s.logger.Sugar()
	host := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	logger.Infof("API server listening on %s", host)
	return http.ListenAndServe(host, mux)
}
