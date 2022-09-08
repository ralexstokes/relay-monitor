package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ralexstokes/relay-monitor/pkg/analysis"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

const (
	methodNotSupported            = "method not supported"
	GetFaultEndpoint              = "/api/v1/relay-monitor/faults"
	RegisterValidatorEndpoint     = "/eth/v1/builder/validators"
	PostAuctionTranscriptEndpoint = "/api/v1/relay-monitor/transcript"
)

type Config struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type Server struct {
	config *Config
	logger *zap.Logger

	analyzer *analysis.Analyzer
	events   chan<- data.Event
}

func New(config *Config, logger *zap.Logger, analyzer *analysis.Analyzer, events chan<- data.Event) *Server {
	return &Server{
		config:   config,
		logger:   logger,
		analyzer: analyzer,
		events:   events,
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleRegisterValidator(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Sugar()

	var registrations []types.SignedValidatorRegistration
	err := json.NewDecoder(r.Body).Decode(&registrations)
	if err != nil {
		logger.Warn("could not decode signed validator registration")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logger.Debugw("got validator registration", "data", registrations)

	payload := data.ValidatorRegistrationEvent{
		Registrations: registrations,
	}
	// TODO what if this is full?
	s.events <- data.Event{Payload: payload}

	// TODO API says we validate the data, but this is pushed to another task
	// block until validations complete?
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAuctionTranscript(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Sugar()

	var transcript types.AuctionTranscript
	err := json.NewDecoder(r.Body).Decode(&transcript)
	if err != nil {
		logger.Warn("could not decode auction transcript")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logger.Debugw("got auction transcript", "data", transcript)

	payload := data.AuctionTranscriptEvent{
		Transcript: &transcript,
	}
	// TODO what if this is full?
	s.events <- data.Event{Payload: payload}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) Run(ctx context.Context) error {
	logger := s.logger.Sugar()
	host := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	logger.Infof("API server listening on %s", host)

	mux := http.NewServeMux()
	mux.HandleFunc(GetFaultEndpoint, get(s.handleFaultsRequest))
	mux.HandleFunc(RegisterValidatorEndpoint, post(s.handleRegisterValidator))
	mux.HandleFunc(PostAuctionTranscriptEndpoint, post(s.handleAuctionTranscript))
	return http.ListenAndServe(host, mux)
}

func get(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handler(w, r)
		default:
			w.WriteHeader(404)
			n, err := w.Write([]byte(methodNotSupported))
			if n != len(methodNotSupported) {
				http.Error(w, "error writing message", http.StatusInternalServerError)
				return
			}
			if err != nil {
				http.Error(w, "error writing message", http.StatusInternalServerError)
				return
			}
		}
	}
}

func post(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			handler(w, r)
		default:
			w.WriteHeader(404)
			n, err := w.Write([]byte(methodNotSupported))
			if n != len(methodNotSupported) {
				http.Error(w, "error writing message", http.StatusInternalServerError)
				return
			}
			if err != nil {
				http.Error(w, "error writing message", http.StatusInternalServerError)
				return
			}
		}
	}
}
