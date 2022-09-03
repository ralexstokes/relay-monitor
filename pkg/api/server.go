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
	Srv    *http.Server
}

func New(config *Config, logger *zap.Logger) *Server {
	return &Server{
		config: config,
		logger: logger,
		Srv: &http.Server{
			Addr: fmt.Sprintf("%s:%d", config.Host, config.Port),
		},
	}
}

func (s *Server) Run(mux *http.ServeMux) error {
	s.Srv.Handler = mux
	logger := s.logger.Sugar()
	logger.Infof("API server listening on %s:%d", s.config.Host, s.config.Port)
	return s.Srv.ListenAndServe()
}

func (s *Server) Shutdown() error {
	return s.Srv.Shutdown(context.Background())
}
