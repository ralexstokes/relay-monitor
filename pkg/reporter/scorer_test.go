package reporter

import (
	"testing"

	"github.com/ralexstokes/relay-monitor/pkg/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testScorer() *Scorer {
	return NewScorer(zap.NewNop().Sugar())
}

func TestComputeReputationScore(t *testing.T) {
	scorer := testScorer()
	records := []*types.Record{
		{
			Slot: 20,
		},
	}
	score, err := scorer.ComputeReputationScore(records, 20)
	require.NoError(t, err)
	require.Equal(t, float64(0), score)

	records = []*types.Record{
		{
			Slot: 1,
		},
	}
	score, err = scorer.ComputeReputationScore(records, 7201)
	require.NoError(t, err)
	require.Equal(t, float64(100), score)

	records = []*types.Record{}
	score, err = scorer.ComputeReputationScore(records, 30)
	require.NoError(t, err)
	require.Equal(t, 100.0, score)
}

func TestBidDeliveryScore(t *testing.T) {
	scorer := testScorer()

	score, err := scorer.ComputeBidDeliveryScore(100, 99, &types.SlotBounds{
		EndSlot: types.SlotPtr(99),
	})
	require.NoError(t, err)
	require.Equal(t, float64(100), score)

	score, err = scorer.ComputeBidDeliveryScore(50, 100, &types.SlotBounds{
		StartSlot: types.SlotPtr(1),
		EndSlot:   types.SlotPtr(100),
	})
	require.NoError(t, err)
	require.Equal(t, float64(50), score)

	score, err = scorer.ComputeBidDeliveryScore(25, 99, &types.SlotBounds{
		StartSlot: nil,
		EndSlot:   nil,
	})
	require.NoError(t, err)
	require.Equal(t, float64(25), score)

	score, err = scorer.ComputeBidDeliveryScore(0, 100, &types.SlotBounds{
		StartSlot: types.SlotPtr(1),
	})
	require.NoError(t, err)
	require.Equal(t, float64(0), score)

	score, err = scorer.ComputeBidDeliveryScore(1, 1000, &types.SlotBounds{
		EndSlot: types.SlotPtr(100),
	})
	require.NoError(t, err)
	require.Greater(t, score, float64(0))
}
