package api

import (
	"fmt"
	"sync"
)

type Config struct {
	Host string
	Port uint16
}

type Server struct {
	config *Config
}

func New(config *Config) *Server {
	return &Server{
		config: config,
	}
}

func (s *Server) Run(wg *sync.WaitGroup) error {
	stop := make(chan struct{})
	fmt.Printf("API server listening on %s:%d\n", s.config.Host, s.config.Port)
	<-stop
	wg.Done()
	return nil
}
