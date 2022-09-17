package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/ralexstokes/relay-monitor/pkg/analysis"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

const (
	methodNotSupported              = "method not supported"
	GetFaultEndpoint                = "/api/v1/relay-monitor/faults"
	RegisterValidatorEndpoint       = "/eth/v1/builder/validators"
	PostAuctionTranscriptEndpoint   = "/api/v1/relay-monitor/transcript"
	DefaultEpochSpanForFaultsWindow = 256
)

type Config struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
}

type Span struct {
	Start types.Epoch `json:"start_epoch,string"`
	End   types.Epoch `json:"end_epoch,string"`
}

type FaultsResponse struct {
	Span                 Span `json:"span"`
	analysis.FaultRecord `json:"data"`
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

// `computeSpan` ensures that `startEpoch` and `endEpoch` cover a "sensible" span where:
//   - `endEpoch` - `startEpoch` == `span` such that `startEpoch` >= 0 and `endEpoch` <= `math.MaxUint64`
//     (so that the span is smaller than requested against the boundaries)
func computeSpanFromRequest(startEpochRequest, endEpochRequest *types.Epoch, targetSpan uint64, currentEpoch types.Epoch) (types.Epoch, types.Epoch) {
	var startEpoch types.Epoch
	endEpoch := currentEpoch

	if startEpochRequest == nil && endEpochRequest == nil {
		diff := int(endEpoch) - int(targetSpan)
		if diff < 0 {
			startEpoch = 0
		} else {
			startEpoch = types.Epoch(diff)
		}
	} else if startEpochRequest != nil && endEpochRequest == nil {
		startEpoch = *startEpochRequest
		boundary := math.MaxUint64 - targetSpan
		if startEpoch > boundary {
			diff := startEpoch - boundary
			endEpoch = startEpoch + diff
		} else {
			endEpoch = startEpoch + targetSpan
		}
	} else if startEpochRequest == nil && endEpochRequest != nil {
		endEpoch = *endEpochRequest
		if endEpoch > targetSpan {
			startEpoch = endEpoch - targetSpan
		} else {
			startEpoch = 0
		}
	} else {
		startEpoch = *startEpochRequest
		endEpoch = *endEpochRequest
	}
	// TODO these can be quite far apart... scope so a caller can't cause a large amount of work
	return startEpoch, endEpoch
}

func (s *Server) handleFaultsRequest(w http.ResponseWriter, r *http.Request) {
	logger := s.logger.Sugar()

	q := r.URL.Query()

	startEpochStr := q.Get("start")
	var startEpochRequest *types.Epoch
	if startEpochStr != "" {
		startEpochValue, err := strconv.ParseUint(startEpochStr, 10, 64)
		if err != nil {
			logger.Errorw("error parsing query param for faults request", "err", err, "startEpoch", startEpochStr)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		epoch := types.Epoch(startEpochValue)
		startEpochRequest = &epoch
	}

	endEpochStr := q.Get("end")
	var endEpochRequest *types.Epoch
	if endEpochStr != "" {
		endEpochValue, err := strconv.ParseUint(endEpochStr, 10, 64)
		if err != nil {
			logger.Errorw("error parsing query param for faults request", "err", err, "endEpoch", endEpochStr)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		epoch := types.Epoch(endEpochValue)
		endEpochRequest = &epoch
	}

	// TODO implement current epoch
	var currentEpoch types.Epoch
	startEpoch, endEpoch := computeSpanFromRequest(startEpochRequest, endEpochRequest, DefaultEpochSpanForFaultsWindow, currentEpoch)
	faults := s.analyzer.GetFaults(startEpoch, endEpoch)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := FaultsResponse{
		Span: Span{
			Start: startEpoch,
			End:   endEpoch,
		},
		FaultRecord: faults,
	}
	encoder := json.NewEncoder(w)
	err := encoder.Encode(response)
	if err != nil {
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
	mux.HandleFunc("/", get(s.handleFaultsRequest))
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
