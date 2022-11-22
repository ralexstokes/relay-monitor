package analysis

import (
	"context"
	"sync"

	"github.com/holiman/uint256"
	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/crypto"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"github.com/ralexstokes/relay-monitor/pkg/store"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

const (
	GasLimitBoundDivisor = 1024
)

type Config struct {
	SignatureDomain crypto.Domain
}

type Analyzer struct {
	config *Config
	logger *zap.Logger

	events <-chan data.Event

	store           store.Storer
	consensusClient *consensus.Client
	clock           *consensus.Clock

	faults     FaultRecord
	faultsLock sync.Mutex
}

func NewAnalyzer(config *Config, logger *zap.Logger, relays []*builder.Client, events <-chan data.Event, store store.Storer, consensusClient *consensus.Client, clock *consensus.Clock) *Analyzer {
	faults := make(FaultRecord)
	for _, relay := range relays {
		faults[relay.PublicKey] = &Faults{}
	}
	return &Analyzer{
		config:          config,
		logger:          logger,
		events:          events,
		store:           store,
		consensusClient: consensusClient,
		clock:           clock,
		faults:          faults,
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

// borrowed from `flashbots/go-boost-utils`
func reverse(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	for i := len(dst)/2 - 1; i >= 0; i-- {
		opp := len(dst) - 1 - i
		dst[i], dst[opp] = dst[opp], dst[i]
	}
	return dst
}

func (a *Analyzer) validateBid(ctx context.Context, bidCtx *types.BidContext, bid *types.Bid) (*InvalidBid, error) {
	if bid == nil {
		return nil, nil
	}

	if bidCtx.RelayPublicKey != bid.Message.Pubkey {
		return &InvalidBid{
			Reason: "incorrect public key from relay",
		}, nil
	}

	validSignature, err := crypto.VerifySignature(bid.Message, a.config.SignatureDomain, bid.Message.Pubkey[:], bid.Signature[:])
	if err != nil {
		return nil, err
	}

	if !validSignature {
		return &InvalidBid{
			Reason: "invalid signature",
		}, nil
	}

	header := bid.Message.Header

	if bidCtx.ParentHash != header.ParentHash {
		return &InvalidBid{
			Reason: "invalid parent hash",
		}, nil
	}

	registration, err := store.GetLatestValidatorRegistration(ctx, a.store, &bidCtx.ProposerPublicKey)
	if err != nil {
		return nil, err
	}
	if registration != nil {
		gasLimitPreference := registration.Message.GasLimit

		// NOTE: need transaction set for possibility of payment transaction
		// so we defer analysis of fee recipient until we have the full payload

		valid, err := a.validateGasLimit(ctx, header.GasLimit, gasLimitPreference, header.BlockNumber)
		if err != nil {
			return nil, err
		}
		if !valid {
			return &InvalidBid{
				Reason: "invalid gas limit",
				Type:   InvalidBidIgnoredPreferencesType,
			}, nil
		}
	}

	expectedRandomness, err := a.consensusClient.GetRandomnessForProposal(bidCtx.Slot)
	if err != nil {
		return nil, err
	}
	if expectedRandomness != header.Random {
		return &InvalidBid{
			Reason: "invalid random value",
		}, nil
	}

	expectedBlockNumber, err := a.consensusClient.GetBlockNumberForProposal(bidCtx.Slot)
	if err != nil {
		return nil, err
	}
	if expectedBlockNumber != header.BlockNumber {
		return &InvalidBid{
			Reason: "invalid block number",
		}, nil
	}

	if header.GasUsed > header.GasLimit {
		return &InvalidBid{
			Reason: "gas used is higher than gas limit",
		}, nil
	}

	expectedTimestamp := a.clock.SlotInSeconds(bidCtx.Slot)
	if expectedTimestamp != int64(header.Timestamp) {
		return &InvalidBid{
			Reason: "invalid timestamp",
		}, nil
	}

	expectedBaseFee, err := a.consensusClient.GetBaseFeeForProposal(bidCtx.Slot)
	if err != nil {
		return nil, err
	}
	baseFee := uint256.NewInt(0)
	baseFee.SetBytes(reverse(header.BaseFeePerGas[:]))
	if !expectedBaseFee.Eq(baseFee) {
		return &InvalidBid{
			Reason: "invalid base fee",
		}, nil
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

	// TODO validations on data
	// - validate bid correlates with acceptance, otherwise consider a count against the proposer

	// TODO clean up nil handling here?
	if event.Transcript == nil {
		return
	}

	transcript := event.Transcript
	bid := &transcript.Bid
	if bid == nil {
		return
	}
	if bid.Message == nil {
		return
	}
	acceptance := &transcript.Acceptance
	if acceptance == nil {
		return
	}
	if acceptance.Message == nil {
		return
	}

	// TODO correlate to existing bid

	// verify later w/ full payload:
	// w/ payload, check:
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

	// TODO save transcript
	// TODO dispatch event to analyze transcript
	// TODO validate to extent possible
	// TODO save analysis results

	// TODO implement
	// proposerPublicKey, err := a.consensusClient.GetPublicKeyForIndex(acceptance.Message.ProposerIndex)
	proposerPublicKey := types.PublicKey{}
	bidCtx := &types.BidContext{
		Slot:              acceptance.Message.Slot,
		ParentHash:        acceptance.Message.Body.ExecutionPayloadHeader.ParentHash,
		ProposerPublicKey: proposerPublicKey,
		RelayPublicKey:    bid.Message.Pubkey,
	}

	err := a.store.PutBid(ctx, bidCtx, bid)
	if err != nil {
		logger.Warnf("could not store bid: %+v", event)
		return
	}

	err = a.store.PutAcceptance(ctx, bidCtx, acceptance)
	if err != nil {
		logger.Warnf("could not store bid acceptance data: %+v", event)
		return
	}
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
