package reporter

import (
	"math"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

type Scorer struct {
	clock *consensus.Clock

	logger *zap.SugaredLogger
}

func NewScorer(clock *consensus.Clock, logger *zap.SugaredLogger) *Scorer {
	return &Scorer{
		clock:  clock,
		logger: logger,
	}
}

///
/// Scoring functions
///

// ComputeTimeWeightedScore computes a score based on the time since the most recent fault.
func (scorer *Scorer) ComputeTimeWeightedScore(faultRecords []*types.Record) (float64, error) {
	// Perfect score if there are no fault records.
	if len(faultRecords) == 0 {
		return 100, nil
	}

	// Controls the rate of decay.
	lambda := 0.1

	// Consider only the most recent fault record.
	t := scorer.clock.CurrentSlot(time.Now().Unix())
	t_most_recent := faultRecords[0].Slot

	return 100 * (1 - math.Exp(-lambda*(float64(t-t_most_recent)))), nil
}

// ComputeReputationScore computes a score based on the reputation of the relay.
func (scorer *Scorer) ComputeReputationScore(faultRecords []*types.Record) (float64, error) {
	// TODO allow selection of more than one scoring function.
	return scorer.ComputeTimeWeightedScore(faultRecords)
}

// ComputeBidDeliveryScore computes a score based on the number of bids delivered.
func (scorer *Scorer) ComputeBidDeliveryScore(countBidsAnalyzed, currentSlot uint64, slotBounds *types.SlotBounds) (float64, error) {
	var slotDiff uint64
	if slotBounds.StartSlot == nil && slotBounds.EndSlot == nil {
		slotDiff = currentSlot
	} else if slotBounds.EndSlot == nil {
		slotDiff = currentSlot - *slotBounds.StartSlot
	} else if slotBounds.StartSlot == nil {
		slotDiff = *slotBounds.EndSlot
	} else {
		slotDiff = *slotBounds.EndSlot - *slotBounds.StartSlot
	}
	return math.Min(100, 100*(float64(countBidsAnalyzed)/float64(slotDiff+1))), nil
}
