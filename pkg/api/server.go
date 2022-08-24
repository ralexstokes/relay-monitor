package api

import (
	"sync"

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

func (s *Server) Run(wg *sync.WaitGroup) error {
	stop := make(chan struct{})
	logger := s.logger.Sugar()
	logger.Infof("API server listening on %s:%d", s.config.Host, s.config.Port)
	<-stop
	wg.Done()
	return nil
}
