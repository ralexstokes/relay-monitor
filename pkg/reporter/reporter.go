package reporter

import (
	"context"
	"fmt"

	"github.com/ralexstokes/relay-monitor/pkg/store"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

// Reporter that a relay monitor can use to generate reports, such as fault reports.
type Reporter struct {
	store  store.Storer
	scorer *Scorer
	logger *zap.SugaredLogger
}

func NewReporter(store store.Storer, scorer *Scorer, logger *zap.SugaredLogger) *Reporter {
	return &Reporter{
		store:  store,
		scorer: scorer,
		logger: logger,
	}
}

///
/// Records
///

func (reporter *Reporter) GetAllInvalidBids(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds) ([]*types.Record, error) {
	return reporter.store.GetRecordsAnalysisWithinSlotBounds(ctx, relay.Pubkey, slotBounds, &types.AnalysisQueryFilter{
		Category:   types.ValidBidCategory,
		Comparator: "!=",
	})
}

func (reporter *Reporter) GetIgnoredPreferencesBids(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds) ([]*types.Record, error) {
	return reporter.store.GetRecordsAnalysisWithinSlotBounds(ctx, relay.Pubkey, slotBounds, &types.AnalysisQueryFilter{
		Category:   types.InvalidBidIgnoredPreferencesCategory,
		Comparator: "=",
	})
}

func (reporter *Reporter) GetConsensusInvalidBids(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds) ([]*types.Record, error) {
	return reporter.store.GetRecordsAnalysisWithinSlotBounds(ctx, relay.Pubkey, slotBounds, &types.AnalysisQueryFilter{
		Category:   types.InvalidBidConsensusCategory,
		Comparator: "=",
	})
}

///
/// Counts
///

func (reporter *Reporter) GetCountIgnoredPreferencesBids(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds) (uint64, error) {
	return reporter.store.GetCountAnalysisWithinSlotBounds(ctx, relay.Pubkey, slotBounds, &types.AnalysisQueryFilter{
		Category:   types.InvalidBidIgnoredPreferencesCategory,
		Comparator: "=",
	})
}

func (reporter *Reporter) GetCountConsensusInvalidBids(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds) (uint64, error) {
	return reporter.store.GetCountAnalysisWithinSlotBounds(ctx, relay.Pubkey, slotBounds, &types.AnalysisQueryFilter{
		Category:   types.InvalidBidConsensusCategory,
		Comparator: "=",
	})
}

func (reporter *Reporter) GetCountTotalValidBids(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds) (uint64, error) {
	return reporter.store.GetCountAnalysisWithinSlotBounds(ctx, relay.Pubkey, slotBounds, &types.AnalysisQueryFilter{
		Category:   types.ValidBidCategory,
		Comparator: "=",
	})
}

func (reporter *Reporter) GetCountTotalBids(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds) (uint64, error) {
	return reporter.store.GetCountAnalysisWithinSlotBounds(ctx, relay.Pubkey, slotBounds, nil)
}

///
/// Stats per Relay
///

func (reporter *Reporter) GetFaultStats(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds) (*types.FaultStats, error) {
	countTotalBids, err := reporter.GetCountTotalBids(ctx, relay, slotBounds)
	if err != nil {
		return nil, fmt.Errorf("could not get total bids: %v", err)
	}

	countConsensusInvalidBids, err := reporter.GetCountConsensusInvalidBids(ctx, relay, slotBounds)
	if err != nil {
		return nil, fmt.Errorf("could not get consensus invalid bids: %v", err)
	}

	countIgnoredPreferencesBids, err := reporter.GetCountIgnoredPreferencesBids(ctx, relay, slotBounds)
	if err != nil {
		return nil, fmt.Errorf("could not get ignored preferences bids: %v", err)
	}

	stats := &types.Stats{
		TotalBids:                countTotalBids,
		ConsensusInvalidBids:     countConsensusInvalidBids,
		IgnoredPreferencesBids:   countIgnoredPreferencesBids,
		PaymentInvalidBids:       0,
		MalformedPayloads:        0,
		ConsensusInvalidPayloads: 0,
		UnavailablePayloads:      0,
	}

	return &types.FaultStats{
		Stats: stats,
		Meta: &types.Meta{
			Hostname: relay.Hostname,
		},
	}, nil
}

///
/// Records per Relay
///

func (reporter *Reporter) GetFaultRecords(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds) (*types.FaultRecords, error) {
	consensusInvalidBids, err := reporter.GetConsensusInvalidBids(ctx, relay, slotBounds)
	if err != nil {
		return nil, fmt.Errorf("could not get consensus invalid bids: %v", err)
	}

	ignoredPreferencesBids, err := reporter.GetIgnoredPreferencesBids(ctx, relay, slotBounds)
	if err != nil {
		return nil, fmt.Errorf("could not get ignored preferences bids: %v", err)
	}

	records := &types.Records{
		ConsensusInvalidBids:     consensusInvalidBids,
		IgnoredPreferencesBids:   ignoredPreferencesBids,
		PaymentInvalidBids:       make([]*types.Record, 0),
		MalformedPayloads:        make([]*types.Record, 0),
		ConsensusInvalidPayloads: make([]*types.Record, 0),
		UnavailablePayloads:      make([]*types.Record, 0),
	}

	return &types.FaultRecords{
		Records: records,
		Meta: &types.Meta{
			Hostname: relay.Hostname,
		},
	}, nil
}

///
/// Reports
///

func (reporter *Reporter) GetFaultStatsReport(ctx context.Context, slotBounds *types.SlotBounds) (types.FaultStatsReport, error) {
	relays, err := reporter.store.GetRelays(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get relays from DB: %v", err)
	}

	faultStatsReport := make(types.FaultStatsReport)
	for _, relay := range relays {
		faultStats, err := reporter.GetFaultStats(ctx, relay, slotBounds)
		if err != nil {
			reporter.logger.Warnf("could not get fault stats for relay %s: %v", relay.Pubkey, err)
			continue
		}
		faultStatsReport[relay.Pubkey] = faultStats
	}
	return faultStatsReport, nil
}

func (reporter *Reporter) GetFaultRecordsReport(ctx context.Context, slotBounds *types.SlotBounds) (types.FaultRecordsReport, error) {
	relays, err := reporter.store.GetRelays(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get relays from DB: %v", err)
	}

	faultRecordsReport := make(types.FaultRecordsReport)
	for _, relay := range relays {
		faultRecords, err := reporter.GetFaultRecords(ctx, relay, slotBounds)
		if err != nil {
			reporter.logger.Warnf("could not get fault records for relay %s: %v", relay.Pubkey, err)
			continue
		}
		faultRecordsReport[relay.Pubkey] = faultRecords
	}
	return faultRecordsReport, nil
}

///
/// Scoring
///

func (reporter *Reporter) GetReputationScore(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds) (*types.Score, error) {
	// Get a list of all invlaid bids for the relay. Every invalid bid is
	// returned as a record.
	invalidBids, err := reporter.GetAllInvalidBids(ctx, relay, slotBounds)
	if err != nil {
		return nil, fmt.Errorf("could not get invalid bids: %v", err)
	}

	// Process the list of invalid bids (records) and compute the score.
	score, err := reporter.scorer.ComputeReputationScore(invalidBids)
	if err != nil {
		return nil, fmt.Errorf("could not calculate score: %v", err)
	}

	return &types.Score{
		Score: score,
		Meta: &types.Meta{
			Hostname: relay.Hostname,
		},
	}, nil
}

func (reporter *Reporter) GetReputationScoreReport(ctx context.Context, slotBounds *types.SlotBounds) (types.ScoreReport, error) {
	relays, err := reporter.store.GetRelays(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get relays from DB: %v", err)
	}

	scoresReport := make(types.ScoreReport)
	for _, relay := range relays {
		score, err := reporter.GetReputationScore(ctx, relay, slotBounds)
		if err != nil {
			reporter.logger.Warnf("could not get score for relay %s: %v", relay.Pubkey, err)
			continue
		}
		scoresReport[relay.Pubkey] = score
	}
	return scoresReport, nil
}

func (reporter *Reporter) GetBidDeliveryScore(ctx context.Context, relay *types.Relay, slotBounds *types.SlotBounds, currentSlot types.Slot) (*types.Score, error) {
	// First get the count of bids analyzed for the relay (this is equivalent to
	// the number of bids that were delivered by the relay in the given time).
	countBidsAnalyzed, err := reporter.GetCountTotalBids(ctx, relay, slotBounds)
	if err != nil {
		return nil, fmt.Errorf("could not get total bids: %v", err)
	}
	// Compute the bid delivery score for the relay.
	score, err := reporter.scorer.ComputeBidDeliveryScore(countBidsAnalyzed, currentSlot, slotBounds)
	reporter.logger.Infow("bid delivery score", "score", score, "countBidsAnalyzed", countBidsAnalyzed, "currentSlot", currentSlot, "slotBounds", slotBounds)
	if err != nil {
		return nil, fmt.Errorf("could not calculate score: %v", err)
	}

	return &types.Score{
		Score: score,
		Meta: &types.Meta{
			Hostname: relay.Hostname,
		},
	}, nil
}

func (reporter *Reporter) GetBidDeliveryScoreReport(ctx context.Context, slotBounds *types.SlotBounds, currentSlot types.Slot) (types.ScoreReport, error) {
	relays, err := reporter.store.GetRelays(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get relays from DB: %v", err)
	}

	scoresReport := make(types.ScoreReport)
	for _, relay := range relays {
		score, err := reporter.GetBidDeliveryScore(ctx, relay, slotBounds, currentSlot)
		if err != nil {
			reporter.logger.Warnf("could not get score for relay %s: %v", relay.Pubkey, err)
			continue
		}
		scoresReport[relay.Pubkey] = score
	}
	return scoresReport, nil
}
