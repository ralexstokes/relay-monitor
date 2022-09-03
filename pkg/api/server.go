package api

import (
	"context"
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
	srv    *http.Server
}

func New(config *Config, logger *zap.Logger) *Server {
	return &Server{
		config: config,
		logger: logger,
		srv: &http.Server{
			Addr: fmt.Sprintf("%s:%d", config.Host, config.Port),
		},
	}
}

func (s *Server) Run(mux *http.ServeMux) error {
	s.srv.Handler = mux
	logger := s.logger.Sugar()
	logger.Infof("API server listening on %s:%d", s.config.Host, s.config.Port)
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown() error {
	return s.srv.Shutdown(context.Background())
}
