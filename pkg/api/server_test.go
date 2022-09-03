package api_test

import (
	"net/http"
	"testing"

	"github.com/ralexstokes/relay-monitor/pkg/api"
	"go.uber.org/zap"
)

const (
	fakeHost = "localhost"
	fakePort = 1559
)

func TestNew(t *testing.T) {
	api.New(&api.Config{}, &zap.Logger{})
}

func TestRun(t *testing.T) {
	s := api.New(&api.Config{
		Host: fakeHost,
		Port: fakePort,
	}, zap.NewExample())

	go func() {
		s.Run(http.NewServeMux())
	}()
	if err := s.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}
