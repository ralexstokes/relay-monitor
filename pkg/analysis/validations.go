package analysis

import (
	"context"

	"github.com/holiman/uint256"
	"github.com/ralexstokes/relay-monitor/pkg/crypto"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

// validatePublicKey validates the public key of a bid to make sure it matches the relay public key
// specified by the bid context.
func (a *Analyzer) validatePublicKey(ctx context.Context, bidCtx *types.BidContext, bid *types.VersionedBid) (*types.InvalidBid, error) {
	bidPublicKey, err := bid.Builder()
	if err != nil {
		a.logger.Error("Failed to get bid public key", zap.Error(err))
		return nil, err
	}

	if bidCtx.RelayPublicKey != bidPublicKey {
		return &types.InvalidBid{
			Category: types.InvalidBidPublicKeyCategory,
			Reason:   types.AnalysisReasonIncorrectPublicKey,
		}, nil
	}

	return nil, nil
}

// validateSignature verifies the signature of the bid message using the bid's public key and the
// signature domain for the bid builder.
func (a *Analyzer) validateSignature(ctx context.Context, bidCtx *types.BidContext, bid *types.VersionedBid) (*types.InvalidBid, error) {
	bidMsg, err := bid.Message()
	if err != nil {
		return nil, err
	}

	bidSignature, err := bid.Signature()
	if err != nil {
		return nil, err
	}
	validSignature, err := crypto.VerifySignature(bidMsg, a.consensusClient.SignatureDomainForBuilder(), bidCtx.RelayPublicKey[:], bidSignature[:])
	if err != nil {
		return nil, err
	}

	if !validSignature {
		return &types.InvalidBid{
			Category: types.InvalidBidSignatureCategory,
			Reason:   types.AnalysisReasonInvalidSignature,
		}, nil
	}

	return nil, nil
}

// validateHeader validates various aspects of a bid message, including the hash, randomness,
// block number, gas used, timestamp, and base fee, to ensure that the bid is consistent with
// the consensus rules. If any of the validations fail, it returns an InvalidBid with a
// category of InvalidBidConsensusCategory and a reason corresponding to the validation that failed.
func (a *Analyzer) validateHeader(ctx context.Context, bidCtx *types.BidContext, bid *types.VersionedBid) (*types.InvalidBid, error) {
	bidParentHash, err := bid.ParentHash()
	if err != nil {
		return nil, err
	}

	// Validate the hash itself.
	if bidCtx.ParentHash != bidParentHash {
		return &types.InvalidBid{
			Category: types.InvalidBidConsensusCategory,
			Reason:   types.AnalysisReasonInvalidParentHash,
		}, nil
	}

	// Verify the RANDAO value.
	expectedRandomness, err := a.consensusClient.GetRandomnessForProposal(bidCtx.Slot)
	if err != nil {
		return nil, err
	}
	bidRandom, err := bid.PrevRandao()
	if err != nil {
		return nil, err
	}
	if expectedRandomness != bidRandom {
		return &types.InvalidBid{
			Category: types.InvalidBidConsensusCategory,
			Reason:   types.AnalysisReasonInvalidRandomValue,
		}, nil
	}

	// Verify the block number.
	expectedBlockNumber, err := a.consensusClient.GetBlockNumberForProposal(bidCtx.Slot)
	if err != nil {
		return nil, err
	}

	bidBlockNumber, err := bid.BlockNumber()
	if err != nil {
		return nil, err
	}
	if expectedBlockNumber != bidBlockNumber {
		return &types.InvalidBid{
			Category: types.InvalidBidConsensusCategory,
			Reason:   types.AnalysisReasonInvalidBlockNumber,
		}, nil
	}

	// Verify that the bid gas used is less than the gas limit.
	bidGasUsed, err := bid.GasUsed()
	if err != nil {
		return nil, err
	}
	bidGasLimit, err := bid.GasLimit()
	if err != nil {
		return nil, err
	}

	if bidGasUsed > bidGasLimit {
		return &types.InvalidBid{
			Category: types.InvalidBidConsensusCategory,
			Reason:   types.AnalysisReasonInvalidGasUsed,
		}, nil
	}

	// Verify the timestamp.
	bidTimestamp, err := bid.Timestamp()
	if err != nil {
		return nil, err
	}

	expectedTimestamp := a.clock.SlotInSeconds(bidCtx.Slot)
	if expectedTimestamp != int64(bidTimestamp) {
		return &types.InvalidBid{
			Category: types.InvalidBidConsensusCategory,
			Reason:   types.AnalysisReasonInvalidTimestamp,
		}, nil
	}

	// Verify the base fee.
	bidBaseFeeForGas, err := bid.BaseFeeForGas()
	if err != nil {
		return nil, err
	}

	expectedBaseFee, err := a.consensusClient.GetBaseFeeForProposal(bidCtx.Slot)
	if err != nil {
		return nil, err
	}
	baseFee := uint256.NewInt(0)
	baseFee.SetBytes(reverse(bidBaseFeeForGas[:]))
	if !expectedBaseFee.Eq(baseFee) {
		return &types.InvalidBid{
			Category: types.InvalidBidConsensusCategory,
			Reason:   types.AnalysisReasonInvalidBaseFee,
		}, nil
	}

	return nil, nil
}

// validateGasLimit validates the gas limit of a bid message to make sure it is within the bounds.
func (a *Analyzer) validateGasLimit(ctx context.Context, gasLimit, gasLimitPreference, blockNumber uint64) (bool, error) {
	if gasLimit == gasLimitPreference {
		return true, nil
	}

	parentGasLimit, err := a.consensusClient.GetParentGasLimit(ctx, blockNumber)
	if err != nil {
		return false, err
	}

	var expectedBound uint64
	if gasLimitPreference > gasLimit {
		expectedBound = parentGasLimit + (parentGasLimit / GasLimitBoundDivisor)
	} else {
		expectedBound = parentGasLimit - (parentGasLimit / GasLimitBoundDivisor)
	}

	return gasLimit == expectedBound, nil
}

// validateValidatorPrefence validates various aspects of the bid message to make sure it complies
// with the validator preferences. If any of the validations fail, it returns an InvalidBid with a
// category of InvalidBidIgnoredPreferencesCategory and a reason corresponding to the validation
// that failed.
func (a *Analyzer) validateValidatorPrefence(ctx context.Context, bidCtx *types.BidContext, bid *types.VersionedBid) (*types.InvalidBid, error) {
	logger := a.logger.Sugar()

	registration, err := a.store.GetLatestValidatorRegistration(ctx, bidCtx.ProposerPublicKey)
	if err != nil {
		return nil, err
	}

	// Only validate if there is a registration.
	if registration != nil {
		// Validate the fee recipient.

		bidFeeRecipient, err := bid.FeeRecipient()
		if err != nil {
			return nil, err
		}

		if registration.Message.FeeRecipient.String() != bidFeeRecipient.String() {
			return &types.InvalidBid{
				Reason:   types.AnalysisReasonIgnoredValidatorPreferenceFeeRecipient,
				Category: types.InvalidBidIgnoredPreferencesCategory,
			}, nil
		}

		// Validate the gas limit preference.
		gasLimitPreference := registration.Message.GasLimit

		bidGasLimit, err := bid.GasLimit()
		if err != nil {
			return nil, err
		}
		bidBlockNumber, err := bid.BlockNumber()
		if err != nil {
			return nil, err
		}

		// NOTE: need transaction set for possibility of payment transaction
		// so we defer analysis of fee recipient until we have the full payload

		valid, err := a.validateGasLimit(ctx, bidGasLimit, gasLimitPreference, bidBlockNumber)
		if err != nil {
			return nil, err
		}
		if !valid {
			return &types.InvalidBid{
				Reason:   types.AnalysisReasonIgnoredValidatorPreferenceGasLimit,
				Category: types.InvalidBidIgnoredPreferencesCategory,
			}, nil
		}
	} else {
		logger.Infow("validator not registered", "proposer", bidCtx.ProposerPublicKey.String())
	}

	return nil, nil
}

// validateBid performs multiple validations on a bid message to ensure that it is valid w.r.t
// consensus and respects validator preferences.
func (a *Analyzer) validateBid(ctx context.Context, bidCtx *types.BidContext, bid *types.VersionedBid) (*types.InvalidBid, error) {
	if bid == nil {
		return nil, nil
	}

	// Validate that the signature is correct.
	invalidBid, err := a.validateSignature(ctx, bidCtx, bid)
	if err != nil {
		return nil, err
	}
	if invalidBid != nil {
		return invalidBid, nil
	}

	// Validate that the public key is correct.
	invalidBid, err = a.validatePublicKey(ctx, bidCtx, bid)
	if err != nil {
		return nil, err
	}
	if invalidBid != nil {
		return invalidBid, nil
	}

	// Validate the header.
	invalidBid, err = a.validateHeader(ctx, bidCtx, bid)
	if err != nil {
		return nil, err
	}
	if invalidBid != nil {
		return invalidBid, nil
	}

	// Validate validator preferences.
	invalidBid, err = a.validateValidatorPrefence(ctx, bidCtx, bid)
	if err != nil {
		return nil, err
	}
	if invalidBid != nil {
		return invalidBid, nil
	}

	return nil, nil
}
