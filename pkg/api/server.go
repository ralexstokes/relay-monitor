package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	fb_types "github.com/flashbots/go-boost-utils/types"
	"github.com/gorilla/mux"
	"github.com/ralexstokes/relay-monitor/pkg/analysis"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/crypto"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"github.com/ralexstokes/relay-monitor/pkg/reporter"
	"github.com/ralexstokes/relay-monitor/pkg/store"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

const (
	methodNotSupported              = "method not supported"
	RegisterValidatorEndpoint       = "/eth/v1/builder/validators"
	PostAuctionTranscriptEndpoint   = "/monitor/v1/transcript"
	DefaultEpochSpanForFaultsWindow = 256

	// Relay fault endpoints.
	GetFaultStatsReportEndpoint   = "/monitor/v1/fault/stats"
	GetFaultStatsEndpoint         = "/monitor/v1/fault/stats/{pubkey:0x[a-fA-F0-9]+}"
	GetFaultRecordsReportEndpoint = "/monitor/v1/fault/records"
	GetFaultRecordsEndpoint       = "/monitor/v1/fault/records/{pubkey:0x[a-fA-F0-9]+}"

	// Relay scoring endpoints.
	GetReputationScoresEndpoint  = "/monitor/v1/scores/reputation"
	GetReputationScoreEndpoint   = "/monitor/v1/scores/reputation/{pubkey:0x[a-fA-F0-9]+}"
	GetBidDeliveryScoresEndpoint = "/monitor/v1/scores/bid_delivery"
	GetBidDeliveryScoreEndpoint  = "/monitor/v1/scores/bid_delivery/{pubkey:0x[a-fA-F0-9]+}"

	// Metrics endpoints.
	GetValidatorsEndpoint              = "/monitor/v1/metrics/validators/count"
	GetValidatorsRegistrationsEndpoint = "/monitor/v1/metrics/validators/registration_count"
	GetBidsAnalyzedCount               = "/monitor/v1/metrics/bids/analyzed_count"
	GetBidsAnalyzedValidCount          = "/monitor/v1/metrics/bids/analyzed_count_valid"
	GetBidsAnalyzedFaultCount          = "/monitor/v1/metrics/bids/analyzed_count_fault"
)

func New(config *Config, logger *zap.Logger, analyzer *analysis.Analyzer, events chan<- data.Event, clock *consensus.Clock, store store.Storer, consensusClient *consensus.Client) *Server {
	return &Server{
		config:          config,
		logger:          logger.Sugar(),
		analyzer:        analyzer,
		events:          events,
		clock:           clock,
		store:           store,
		reporter:        reporter.NewReporter(store, reporter.NewScorer(clock, logger.Sugar()), logger.Sugar()),
		consensusClient: consensusClient,
	}
}

func (s *Server) handleBidsAnalyzedRequest(queryFilter *types.AnalysisQueryFilter, w http.ResponseWriter, r *http.Request) {

	q := r.URL.Query()

	var analysisCount uint64

	lookbackSlots := q.Get("lookbackSlots")
	lookbackMinutes := q.Get("lookbackMinutes")

	// Handle either slots or duration (minutes) lookback.
	if lookbackSlots != "" {
		lookbackSlotsValue, err := strconv.ParseUint(lookbackSlots, 10, 64)
		if err != nil {
			s.logger.Errorw("error parsing query param for analysis request", "err", err, "lookbackSlots", lookbackSlotsValue)
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		analysisCount, err = s.store.GetCountAnalysisLookbackSlots(context.Background(), lookbackSlotsValue, queryFilter)
		if err != nil {
			s.logger.Errorw("error executing query", "err", err)
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	} else if lookbackMinutes != "" {
		lookbackMinutesValue, err := strconv.ParseUint(lookbackMinutes, 10, 64)
		if err != nil {
			s.logger.Errorw("error parsing query param for analysis request", "err", err, "lookbackMinutes", lookbackMinutesValue)
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		// For now we only support lookback in minutes.
		duration := time.Duration(lookbackMinutesValue) * time.Minute
		analysisCount, err = s.store.GetCountAnalysisLookbackDuration(context.Background(), duration, queryFilter)
		if err != nil {
			s.logger.Errorw("error executing query", "err", err)
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		s.logger.Errorw("incomplete request, using default stats lookback", "lookbackSlots", lookbackSlots, "lookbackMinutes", lookbackMinutes)
		var err error
		// TODO: Make this configurable.
		analysisCount, err = s.store.GetCountAnalysisLookbackSlots(context.Background(), 7200, queryFilter)
		if err != nil {
			s.logger.Errorw("error executing query", "err", err)
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	response := CountResponse{
		Count: uint(analysisCount),
	}
	s.respondOK(w, response)
}

func (s *Server) handleBidsAnalyzedFaultCountRequest(w http.ResponseWriter, r *http.Request) {

	queryFilter := &types.AnalysisQueryFilter{
		Category:   types.ValidBidCategory,
		Comparator: "!=",
	}

	s.handleBidsAnalyzedRequest(queryFilter, w, r)
}

func (s *Server) handleBidsAnalyzedValidCountRequest(w http.ResponseWriter, r *http.Request) {

	queryFilter := &types.AnalysisQueryFilter{
		Category:   types.ValidBidCategory,
		Comparator: "=",
	}

	s.handleBidsAnalyzedRequest(queryFilter, w, r)
}

func (s *Server) handleBidsAnalyzedCountRequest(w http.ResponseWriter, r *http.Request) {
	s.handleBidsAnalyzedRequest(nil, w, r)
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

func (s *Server) currentEpoch() types.Epoch {
	now := time.Now().Unix()
	slot := s.clock.CurrentSlot(now)
	return s.clock.EpochForSlot(slot)
}

func (s *Server) currentSlot() types.Slot {
	now := time.Now().Unix()
	return s.clock.CurrentSlot(now)
}

func (s *Server) parseSlotBounds(q url.Values) (*types.SlotBounds, error) {
	// Parse the start and end slots.
	startSlotStr := q.Get("start")
	var startSlot *types.Slot
	if startSlotStr != "" {
		startSlotValue, err := strconv.ParseUint(startSlotStr, 10, 64)
		if err != nil {
			s.logger.Errorw("error parsing query param for faults request", "err", err, "startSlot", startSlotStr)
			return nil, err
		}
		startSlot = &startSlotValue
	}
	endSlotStr := q.Get("end")
	var endSlot *types.Slot
	if endSlotStr != "" {
		endSlotValue, err := strconv.ParseUint(endSlotStr, 10, 64)
		if err != nil {
			s.logger.Errorw("error parsing query param for faults request", "err", err, "endSlot", endSlotStr)
			return nil, err
		}
		endSlot = &endSlotValue
	}

	// If a window is specified, then we override the start and end slots and compute them from the
	// current slot.
	windowSlotStr := q.Get("window")
	if windowSlotStr != "" {
		windowSlot, err := strconv.ParseUint(windowSlotStr, 10, 64)
		if err != nil {
			s.logger.Errorw("error parsing query param for faults request", "err", err, "windowSlot", windowSlotStr)
			return nil, err
		}
		// TODO: move this to a constant.
		if windowSlot >= 100_000 {
			return nil, errors.New("window slot is too large")
		}
		currentSlot := s.currentSlot()

		startSlot := currentSlot - windowSlot
		endSlot := currentSlot

		return &types.SlotBounds{
			StartSlot: &startSlot,
			EndSlot:   &endSlot,
		}, nil
	}

	return &types.SlotBounds{
		StartSlot: startSlot,
		EndSlot:   endSlot,
	}, nil
}

func (s *Server) handleReputationScoresRequest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	slotBounds, err := s.parseSlotBounds(q)
	if err != nil {
		s.logger.Errorw("error parsing slot bounds", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	scoreReport, err := s.reporter.GetReputationScoreReport(context.Background(), slotBounds)
	if err != nil {
		s.logger.Errorw("error getting scores", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
	}

	response := ScoreReportResponse{
		SlotBounds: *slotBounds,
		Report:     scoreReport,
	}
	s.respondOK(w, response)
}

func (s *Server) handleReputationScoreRequest(w http.ResponseWriter, r *http.Request) {
	// Extract the relay pubkey from the URL.
	vars := mux.Vars(r)
	relayPubkeyHex := vars["pubkey"]

	pubkey, err := fb_types.HexToPubkey(relayPubkeyHex)
	if err != nil {
		s.logger.Errorw("error parsing pubkey", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(relayPubkeyHex) != 98 {
		s.respondError(w, http.StatusBadRequest, "invalid pubkey")
		return
	}

	q := r.URL.Query()

	slotBounds, err := s.parseSlotBounds(q)
	if err != nil {
		s.logger.Errorw("error parsing slot bounds", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Find the relay.
	relay, err := s.store.GetRelay(context.Background(), &pubkey)
	if err != nil {
		s.logger.Errorw("error getting relay", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	score, err := s.reporter.GetReputationScore(context.Background(), relay, slotBounds)
	if err != nil {
		s.logger.Errorw("error getting score", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
	}

	response := ScoreReponse{
		SlotBounds: *slotBounds,
		Score:      *score,
	}
	s.respondOK(w, response)
}

func (s *Server) handleBidDeliveryScoresRequest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	slotBounds, err := s.parseSlotBounds(q)
	if err != nil {
		s.logger.Errorw("error parsing slot bounds", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	scoreReport, err := s.reporter.GetBidDeliveryScoreReport(context.Background(), slotBounds, s.currentSlot())
	if err != nil {
		s.logger.Errorw("error getting scores", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
	}

	response := ScoreReportResponse{
		SlotBounds: *slotBounds,
		Report:     scoreReport,
	}
	s.respondOK(w, response)
}

func (s *Server) handleBidDeliveryScoreRequest(w http.ResponseWriter, r *http.Request) {
	// Extract the relay pubkey from the URL.
	vars := mux.Vars(r)
	relayPubkeyHex := vars["pubkey"]

	pubkey, err := fb_types.HexToPubkey(relayPubkeyHex)
	if err != nil {
		s.logger.Errorw("error parsing pubkey", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(relayPubkeyHex) != 98 {
		s.respondError(w, http.StatusBadRequest, "invalid pubkey")
		return
	}

	q := r.URL.Query()

	slotBounds, err := s.parseSlotBounds(q)
	if err != nil {
		s.logger.Errorw("error parsing slot bounds", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Find the relay.
	relay, err := s.store.GetRelay(context.Background(), &pubkey)
	if err != nil {
		s.logger.Errorw("error getting relay", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	score, err := s.reporter.GetBidDeliveryScore(context.Background(), relay, slotBounds, s.currentSlot())
	if err != nil {
		s.logger.Errorw("error getting score", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
	}

	response := ScoreReponse{
		SlotBounds: *slotBounds,
		Score:      *score,
	}
	s.respondOK(w, response)
}

func (s *Server) handleFaultRecordsReportRequest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	slotBounds, err := s.parseSlotBounds(q)
	if err != nil {
		s.logger.Errorw("error parsing slot bounds", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	faultRecords, err := s.reporter.GetFaultRecordsReport(context.Background(), slotBounds)
	if err != nil {
		s.logger.Errorw("error getting fault records", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
	}

	response := FaultRecordsReportResponse{
		SlotBounds: *slotBounds,
		Data:       faultRecords,
	}
	s.respondOK(w, response)
}

func (s *Server) handleFaultStatsReportRequest(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	slotBounds, err := s.parseSlotBounds(q)
	if err != nil {
		s.logger.Errorw("error parsing slot bounds", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	faultStatsReport, err := s.reporter.GetFaultStatsReport(context.Background(), slotBounds)
	if err != nil {
		s.logger.Errorw("error getting fault stats report", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
	}

	response := FaultStatsReportResponse{
		SlotBounds: *slotBounds,
		Data:       faultStatsReport,
	}
	s.respondOK(w, response)
}

func (s *Server) handleFaultRecordsRequest(w http.ResponseWriter, r *http.Request) {
	// Extract the relay pubkey from the URL.
	vars := mux.Vars(r)
	relayPubkeyHex := vars["pubkey"]

	pubkey, err := fb_types.HexToPubkey(relayPubkeyHex)
	if err != nil {
		s.logger.Errorw("error parsing pubkey", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(relayPubkeyHex) != 98 {
		s.respondError(w, http.StatusBadRequest, "invalid pubkey")
		return
	}

	q := r.URL.Query()

	slotBounds, err := s.parseSlotBounds(q)
	if err != nil {
		s.logger.Errorw("error parsing slot bounds", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Find the relay.
	relay, err := s.store.GetRelay(context.Background(), &pubkey)
	if err != nil {
		s.logger.Errorw("error getting relay", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	faultRecords, err := s.reporter.GetFaultRecords(context.Background(), relay, slotBounds)
	if err != nil {
		s.logger.Errorw("error getting fault records", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
	}

	response := FaultRecordsResponse{
		SlotBounds: *slotBounds,
		Data:       *faultRecords,
	}
	s.respondOK(w, response)
}

func (s *Server) handleFaultStatsRequest(w http.ResponseWriter, r *http.Request) {
	// Extract the relay pubkey from the URL.
	vars := mux.Vars(r)
	relayPubkeyHex := vars["pubkey"]

	pubkey, err := fb_types.HexToPubkey(relayPubkeyHex)
	if err != nil {
		s.logger.Errorw("error parsing pubkey", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if len(relayPubkeyHex) != 98 {
		s.respondError(w, http.StatusBadRequest, "invalid pubkey")
		return
	}

	q := r.URL.Query()

	slotBounds, err := s.parseSlotBounds(q)
	if err != nil {
		s.logger.Errorw("error parsing slot bounds", "err", err)
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Find the relay.
	relay, err := s.store.GetRelay(context.Background(), &pubkey)
	if err != nil {
		s.logger.Errorw("error getting relay", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	faultStats, err := s.reporter.GetFaultStats(context.Background(), relay, slotBounds)
	if err != nil {
		s.logger.Errorw("error getting fault stats", "err", err)
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := FaultStatsResponse{
		SlotBounds: *slotBounds,
		Data:       *faultStats,
	}
	s.respondOK(w, response)
}

func (s *Server) validateRegistrationTimestamp(registration, currentRegistration *types.SignedValidatorRegistration) error {
	timestamp := registration.Message.Timestamp
	deadline := time.Now().Add(10 * time.Second).Unix()
	if timestamp >= uint64(deadline) {
		return fmt.Errorf("invalid registration: too far in future, %+v", registration)
	}

	if currentRegistration != nil {
		lastTimestamp := currentRegistration.Message.Timestamp
		if timestamp < lastTimestamp {
			return fmt.Errorf("invalid registration: more recent successful registration, %+v", registration)
		}
	}

	return nil
}

func (s *Server) validateRegistrationSignature(registration *types.SignedValidatorRegistration) error {
	msg := registration.Message
	valid, err := crypto.VerifySignature(msg, s.consensusClient.SignatureDomainForBuilder(), msg.Pubkey[:], registration.Signature[:])
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("signature invalid for validator registration %+v", registration)
	}
	return nil
}

func (s *Server) validateRegistrationValidatorStatus(registration *types.SignedValidatorRegistration) error {
	publicKey := registration.Message.Pubkey
	status, err := s.consensusClient.GetValidatorStatus(&publicKey)
	if err != nil {
		return err
	}

	switch status {
	case consensus.StatusValidatorActive, consensus.StatusValidatorPending:
		return nil
	default:
		return fmt.Errorf("invalid registration: validator lifecycle status %s is not `active` or `pending`, %+v", status, registration)
	}
}

func (s *Server) validateRegistration(registration, currentRegistration *types.SignedValidatorRegistration) error {
	err := s.validateRegistrationTimestamp(registration, currentRegistration)
	if err != nil {
		return err
	}

	err = s.validateRegistrationSignature(registration)
	if err != nil {
		return err
	}

	err = s.validateRegistrationValidatorStatus(registration)
	if err != nil {
		return err
	}

	return nil
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *Server) respondError(w http.ResponseWriter, code int, message string) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	response := apiError{code, message}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", " ")

	if err := encoder.Encode(response); err != nil {
		s.logger.Errorw("couldn't write error response", "response", response, "error", err)
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (s *Server) respondOK(w http.ResponseWriter, response any) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Errorw("couldn't write OK response", "response", response, "error", err)
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (s *Server) handleCountValidators(w http.ResponseWriter, r *http.Request) {

	validators, err := s.store.GetCountValidators(context.Background())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response := CountResponse{
		Count: validators,
	}
	s.logger.Debugw("processed request for validators", "count", response.Count)

	s.respondOK(w, response)
}

func (s *Server) handleCountValidatorsRegistrations(w http.ResponseWriter, r *http.Request) {

	registrations, err := s.store.GetCountValidatorsRegistrations(context.Background())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	response := CountResponse{
		Count: registrations,
	}
	s.logger.Debugw("processed request for validators registrations", "count", response.Count)

	s.respondOK(w, response)
}

func (s *Server) handleRegisterValidator(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var registrations []types.SignedValidatorRegistration
	err := json.NewDecoder(r.Body).Decode(&registrations)
	if err != nil {
		s.logger.Warn("could not decode signed validator registration")
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	for _, registration := range registrations {
		currentRegistration, err := s.store.GetLatestValidatorRegistration(ctx, &registration.Message.Pubkey)
		if err != nil {
			s.logger.Warnw("could not get registrations for validator", "error", err, "registration", registration)
			s.respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		err = s.validateRegistration(&registration, currentRegistration)
		if err != nil {
			s.logger.Warnw("invalid validator registration in batch", "registration", registration, "error", err)
			s.respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	payload := data.ValidatorRegistrationEvent{
		Registrations: registrations,
	}
	// TODO what if this is full?
	s.events <- data.Event{Payload: payload}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAuctionTranscript(w http.ResponseWriter, r *http.Request) {

	var transcript types.AuctionTranscript
	err := json.NewDecoder(r.Body).Decode(&transcript)
	if err != nil {
		s.logger.Warn("could not decode auction transcript")
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.logger.Debugw("got auction transcript", "data", transcript)

	payload := data.AuctionTranscriptEvent{
		Transcript: &transcript,
	}
	// TODO what if this is full?
	s.events <- data.Event{Payload: payload}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) Run(ctx context.Context) error {
	host := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Infof("API server listening on %s", host)

	r := mux.NewRouter()
	r.HandleFunc("/", get(s.handleFaultStatsReportRequest))

	// Report route handlers.
	r.HandleFunc(GetFaultStatsReportEndpoint, get(s.handleFaultStatsReportRequest))
	r.HandleFunc(GetFaultRecordsReportEndpoint, get(s.handleFaultRecordsReportRequest))

	// Per-relay stats and records API route handlers.
	r.HandleFunc(GetFaultStatsEndpoint, get(s.handleFaultStatsRequest))
	r.HandleFunc(GetFaultRecordsEndpoint, get(s.handleFaultRecordsRequest))

	// Score route handlers.
	r.HandleFunc(GetReputationScoresEndpoint, get(s.handleReputationScoresRequest))
	r.HandleFunc(GetReputationScoreEndpoint, get(s.handleReputationScoreRequest))
	r.HandleFunc(GetBidDeliveryScoresEndpoint, get(s.handleBidDeliveryScoresRequest))
	r.HandleFunc(GetBidDeliveryScoreEndpoint, get(s.handleBidDeliveryScoreRequest))

	// Validator route handlers.
	r.HandleFunc(RegisterValidatorEndpoint, post(s.handleRegisterValidator))

	// Proposer-view auction view route handler.
	r.HandleFunc(PostAuctionTranscriptEndpoint, post(s.handleAuctionTranscript))

	// Metrics route handlers.
	r.HandleFunc(GetValidatorsEndpoint, get(s.handleCountValidators))
	r.HandleFunc(GetValidatorsRegistrationsEndpoint, get(s.handleCountValidatorsRegistrations))

	r.HandleFunc(GetBidsAnalyzedCount, get(s.handleBidsAnalyzedCountRequest))
	r.HandleFunc(GetBidsAnalyzedValidCount, get(s.handleBidsAnalyzedValidCountRequest))
	r.HandleFunc(GetBidsAnalyzedFaultCount, get(s.handleBidsAnalyzedFaultCountRequest))

	return http.ListenAndServe(host, r)
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
