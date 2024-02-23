package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/holiman/uint256"
	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/crypto"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"github.com/ralexstokes/relay-monitor/pkg/output"
	"github.com/ralexstokes/relay-monitor/pkg/store"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

const (
	GasLimitBoundDivisor = 1024
	ExpectedKey          = "expected"
	ActualKey            = "actual"
	RelayerPubKey        = "pubKey"
	SlotKey              = "slot"
	ErrTypeKey           = "errType"
)

type Analyzer struct {
	logger *zap.Logger

	events <-chan data.Event

	store           store.Storer
	consensusClient *consensus.Client
	clock           *consensus.Clock

	faults     FaultRecord
	faultsLock sync.Mutex

	output *output.Output
	region string
}

func NewAnalyzer(logger *zap.Logger, relays []*builder.Client, events <-chan data.Event, store store.Storer, consensusClient *consensus.Client, clock *consensus.Clock, output *output.Output, region string) *Analyzer {
	faults := make(FaultRecord)
	for _, relay := range relays {
		faults[relay.PublicKey] = &Faults{}
	}
	return &Analyzer{
		logger:          logger,
		events:          events,
		store:           store,
		consensusClient: consensusClient,
		clock:           clock,
		faults:          faults,
		output:          output,
		region:          region,
	}
}

func (a *Analyzer) GetFaults(start, end types.Epoch) FaultRecord {
	a.faultsLock.Lock()
	defer a.faultsLock.Unlock()

	faults := make(FaultRecord)
	for relay, summary := range a.faults {
		summary := *summary
		faults[relay] = &summary
	}

	return faults
}

func (a *Analyzer) validateGasLimit(ctx context.Context, gasLimit uint64, gasLimitPreference uint64, blockNumber uint64) (bool, error) {
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

func (a *Analyzer) outputValidationError(validationError *InvalidBid) {
	if validationError == nil || validationError.Reason == "" {
		return
	}

	go func() {
		logger := a.logger.Sugar()

		// Expected and actual are not defined for all errors, extract them if present
		var expected interface{}
		var actual interface{}

		if v, okay := validationError.Context[ExpectedKey]; okay {
			expected = v
		}

		if v, okay := validationError.Context[ActualKey]; okay {
			actual = v
		}

		out := &data.ValidationOutput{
			Timestamp:      time.Unix(a.clock.SlotInSeconds(types.Slot(validationError.Context[SlotKey].(uint64))), 0),
			Region:         a.region,
			RelayPublicKey: validationError.Context[RelayerPubKey].(types.PublicKey).String(),
			Slot:           types.Slot(validationError.Context[SlotKey].(uint64)),
			Error: &data.ValidationErr{
				Type:     validationError.Context[ErrTypeKey].(types.ErrorType),
				Reason:   validationError.Reason,
				Expected: expected,
				Actual:   actual,
			},
		}

		outBytes, err := json.Marshal(out)
		if err != nil {
			logger.Warnw("unable to marshal output", "error", err, "content", out)
		}
		err = a.output.WriteEntry(outBytes)
		if err != nil {
			logger.Warnw("unable to write output", "error", err)
		}

	}()
}

func (a *Analyzer) validateBid(ctx context.Context, bidCtx *types.BidContext, bid *types.Bid) (*InvalidBid, error) {
	if bid == nil {
		return nil, nil
	}

	invalidBidErr := &InvalidBid{
		Context: map[string]interface{}{
			ErrTypeKey:    types.ValidationErr,
			RelayerPubKey: bidCtx.RelayPublicKey,
			SlotKey:       bidCtx.Slot,
		},
	}

	blockNumber, err := bid.BlockNumber()
	if err != nil {
		invalidBidErr.Reason = fmt.Sprintf("failed to get bid block number: %s", err)
		return invalidBidErr, nil
	}

	gasLimit, err := bid.GasLimit()
	if err != nil {
		invalidBidErr.Reason = fmt.Sprintf("failed to get bid gas limit: %s", err)
		return invalidBidErr, nil
	}

	gasUsed, err := bid.GasUsed()
	if err != nil {
		invalidBidErr.Reason = fmt.Sprintf("failed to get bid gas used: %s", err)
		return invalidBidErr, nil
	}

	timestamp, err := bid.Timestamp()
	if err != nil {
		invalidBidErr.Reason = fmt.Sprintf("failed to get bid timestamp: %s", err)
		return invalidBidErr, nil
	}

	publicKey, err := bid.Builder()
	if err != nil {
		invalidBidErr.Reason = fmt.Sprintf("failed to get bid public key: %s", err)
		return invalidBidErr, nil
	}

	baseFeePerGas, err := bid.BaseFeePerGas()
	if err != nil {
		invalidBidErr.Reason = fmt.Sprintf("failed to get bid base fee per gas: %s", err)
		return invalidBidErr, nil
	}

	prevRandao, err := bid.Random()
	if err != nil {
		invalidBidErr.Reason = fmt.Sprintf("failed to get bid random: %s", err)
		return invalidBidErr, nil
	}

	parentHash, err := bid.ParentHash()
	if err != nil {
		invalidBidErr.Reason = fmt.Sprintf("failed to get bid parent hash: %s", err)
		return invalidBidErr, nil
	}

	signature, err := bid.Signature()
	if err != nil {
		invalidBidErr.Reason = fmt.Sprintf("failed to get bid signature: %s", err)
		return invalidBidErr, nil
	}

	defer a.outputValidationError(invalidBidErr)
	if bidCtx.RelayPublicKey != types.PublicKey(publicKey) {
		invalidBidErr.Reason = "incorrect public key from relay"
		invalidBidErr.Context[ExpectedKey] = bidCtx.RelayPublicKey
		invalidBidErr.Context[ActualKey] = types.PublicKey(publicKey)
		return invalidBidErr, nil
	}

	validSignature, err := crypto.VerifySignature(bid, a.consensusClient.SignatureDomainForBuilder(), publicKey[:], signature[:])
	if err != nil {
		return nil, err
	}
	if !validSignature {
		invalidBidErr.Reason = "relay public key does not match signature"
		// No actual and expected when signatures don't match
		return invalidBidErr, nil
	}

	if bidCtx.ParentHash != parentHash {
		invalidBidErr.Reason = "invalid parent hash"
		invalidBidErr.Context[ExpectedKey] = bidCtx.ParentHash
		invalidBidErr.Context[ActualKey] = parentHash
		return invalidBidErr, nil
	}

	registration, err := store.GetLatestValidatorRegistration(ctx, a.store, &bidCtx.ProposerPublicKey)
	if err != nil {
		return nil, err
	}
	if registration != nil {
		gasLimitPreference := registration.Message.GasLimit

		// NOTE: need transaction set for possibility of payment transaction
		// so we defer analysis of fee recipient until we have the full payload

		valid, err := a.validateGasLimit(ctx, gasLimit, gasLimitPreference, blockNumber)
		if err != nil {
			return nil, err
		}
		if !valid {
			invalidBidErr.Reason = "invalid gas limit"
			invalidBidErr.Context[ExpectedKey] = gasLimitPreference
			invalidBidErr.Context[ActualKey] = gasLimit
			return invalidBidErr, nil
		}
	}

	expectedRandomness, err := a.consensusClient.GetRandomnessForProposal(phase0.Slot(bidCtx.Slot))
	if err != nil {
		return nil, err
	}
	if expectedRandomness != prevRandao {
		invalidBidErr.Context[ExpectedKey] = expectedRandomness
		invalidBidErr.Context[ActualKey] = prevRandao
		return invalidBidErr, nil
	}

	expectedBlockNumber, err := a.consensusClient.GetBlockNumberForProposal(phase0.Slot(bidCtx.Slot))
	if err != nil {
		return nil, err
	}
	if expectedBlockNumber != blockNumber {
		invalidBidErr.Reason = "invalid block number"
		invalidBidErr.Context[ExpectedKey] = expectedBlockNumber
		invalidBidErr.Context[ActualKey] = blockNumber
		return invalidBidErr, nil
	}

	if gasUsed > gasLimit {
		invalidBidErr.Reason = "gas used is higher than gas limit"
		invalidBidErr.Context[ExpectedKey] = gasLimit
		invalidBidErr.Context[ActualKey] = gasUsed
		return invalidBidErr, nil
	}

	expectedTimestamp := a.clock.SlotInSeconds(phase0.Slot(bidCtx.Slot))
	if expectedTimestamp != int64(timestamp) {
		invalidBidErr.Reason = "invalid timestamp"
		invalidBidErr.Context[ExpectedKey] = expectedTimestamp
		invalidBidErr.Context[ActualKey] = timestamp
		return invalidBidErr, nil
	}

	expectedBaseFee, err := a.consensusClient.GetBaseFeeForProposal(phase0.Slot(bidCtx.Slot))
	if err != nil {
		return nil, err
	}

	baseFee := uint256.NewInt(0)
	baseFee.SetFromBig(baseFeePerGas)

	if !expectedBaseFee.Eq(baseFee) {
		invalidBidErr.Reason = "invalid base fee"
		invalidBidErr.Context[ExpectedKey] = expectedBaseFee
		invalidBidErr.Context[ActualKey] = baseFee
		return invalidBidErr, err
	}

	return nil, nil
}

func (a *Analyzer) processBid(ctx context.Context, event *data.BidEvent) {
	logger := a.logger.Sugar()

	bidCtx := event.Context
	bid := event.Bid

	err := a.store.PutBid(ctx, bidCtx, bid)
	if err != nil {
		logger.Warnf("could not store bid: %+v", event)
		return
	}

	result, err := a.validateBid(ctx, bidCtx, bid)
	if err != nil {
		logger.Warnf("could not validate bid with error %+v: %+v, %+v", err, bidCtx, bid)
		return
	}

	// TODO scope faults by coordinate
	// TODO persist analysis results
	relayID := bidCtx.RelayPublicKey
	a.faultsLock.Lock()
	faults := a.faults[relayID]
	if bid != nil {
		faults.TotalBids += 1
	}
	if result != nil {
		switch result.Type {
		case InvalidBidConsensusType:
			faults.ConsensusInvalidBids += 1
		case InvalidBidIgnoredPreferencesType:
			faults.IgnoredPreferencesBids += 1
		default:
			logger.Warnf("could not interpret bid analysis result: %+v, %+v", event, result)
			return
		}
	}
	a.faultsLock.Unlock()
	if result != nil {
		logger.Debugf("invalid bid: %+v, %+v", result, event)
	} else {
		logger.Debugf("found valid bid: %+v, %+v", bidCtx, bid)
	}
}

// Process incoming validator registrations
// This data has already been validated by the sender of the event
func (a *Analyzer) processValidatorRegistration(ctx context.Context, event data.ValidatorRegistrationEvent) {
	logger := a.logger.Sugar()

	registrations := event.Registrations
	logger.Debugf("received %d validator registrations", len(registrations))
	for _, registration := range registrations {
		err := a.store.PutValidatorRegistration(ctx, &registration)
		if err != nil {
			logger.Warnf("could not store validator registration: %+v", registration)
			return
		}
	}
}

func (a *Analyzer) processAuctionTranscript(ctx context.Context, event data.AuctionTranscriptEvent) {
	logger := a.logger.Sugar()

	logger.Debugf("received transcript: %+v", event.Transcript)

	transcript := event.Transcript

	bid := transcript.Bid
	signedBlindedBeaconBlock := &transcript.Acceptance

	proposerIndex, err := signedBlindedBeaconBlock.ProposerIndex()
	if err != nil {
		logger.Warnw("could not get proposer index from acceptance", "error", err, "acceptance", signedBlindedBeaconBlock)
		return
	}

	slot, err := signedBlindedBeaconBlock.Slot()
	if err != nil {
		logger.Warnw("could not get slot from acceptance", "error", err, "acceptance", signedBlindedBeaconBlock)
		return
	}

	publicKey, err := bid.Builder()
	if err != nil {
		logger.Warnw("could not get public key from bid", "error", err, "bid", bid)
		return
	}

	parentHash, err := bid.ParentHash()
	if err != nil {
		logger.Warnw("could not get parent hash from bid", "error", err, "bid", bid)
		return
	}

	signature, err := signedBlindedBeaconBlock.Signature()
	if err != nil {
		logger.Warnw("could not get signature from acceptance", "error", err, "acceptance", signedBlindedBeaconBlock)
		return
	}

	// Verify signature first, to avoid doing unnecessary work in the event this is a "bad" transcript
	proposerPublicKey, err := a.consensusClient.GetPublicKeyForIndex(ctx, uint64(proposerIndex))
	if err != nil {
		logger.Warnw("could not find public key for validator index", "error", err)
		return
	}

	domain := a.consensusClient.SignatureDomain(slot)
	valid, err := crypto.VerifySignature(signedBlindedBeaconBlock, domain, proposerPublicKey[:], signature[:])
	if err != nil {
		logger.Warnw("error verifying signature from proposer; could not determine authenticity of transcript", "error", err, "bid", bid, "acceptance", signedBlindedBeaconBlock)
		return
	}
	if !valid {
		logger.Warnw("signature from proposer was invalid; could not determine authenticity of transcript", "error", err, "bid", bid, "acceptance", signedBlindedBeaconBlock)
		return
	}

	bidCtx := &types.BidContext{
		Slot:              uint64(slot),
		ParentHash:        parentHash,
		ProposerPublicKey: *proposerPublicKey,
		RelayPublicKey:    types.PublicKey(publicKey),
	}
	existingBid, err := a.store.GetBid(ctx, bidCtx)
	if err != nil {
		logger.Warnw("could not find existing bid, will continue full analysis", "context", bidCtx)

		// TODO: process bid as well as rest of transcript
	}

	if existingBid != nil && *existingBid != transcript.Bid {
		logger.Warnw("provided bid from transcript did not match existing bid, will continue full analysis", "context", bidCtx)

		// TODO: process bid as well as rest of transcript
	}

	// TODO also store bid if missing?
	err = a.store.PutAcceptance(ctx, bidCtx, signedBlindedBeaconBlock)
	if err != nil {
		logger.Warnf("could not store bid acceptance data: %+v", event)
		return
	}

	// verify later w/ full payload:
	// (claimed) Value, including fee recipient
	// expectedFeeRecipient := registration.Message.FeeRecipient
	// if expectedFeeRecipient != header.FeeRecipient {
	// 	return &InvalidBid{
	// 		Reason: "invalid fee recipient",
	// 		Type:   InvalidBidIgnoredPreferencesType,
	// 		Context: map[string]interface{}{
	// 			"expected fee recipient":  expectedFeeRecipient,
	// 			"fee recipient in header": header.FeeRecipient,
	// 		},
	// 	}, nil
	// }

	// BlockHash
	// StateRoot
	// ReceiptsRoot
	// LogsBloom
	// TransactionsRoot

	// TODO save analysis results

}

func (a *Analyzer) Run(ctx context.Context) error {
	logger := a.logger.Sugar()

	for {
		select {
		case event := <-a.events:
			switch event := event.Payload.(type) {
			case *data.BidEvent:
				a.processBid(ctx, event)
			case data.ValidatorRegistrationEvent:
				a.processValidatorRegistration(ctx, event)
			case data.AuctionTranscriptEvent:
				a.processAuctionTranscript(ctx, event)
			default:
				logger.Warnf("unknown event type %T for event %+v!", event, event)
			}
		case <-ctx.Done():
			return nil
		}
	}
}
