package analysis

import (
	"context"
	"sync"

	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"github.com/ralexstokes/relay-monitor/pkg/store"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

type Analyzer struct {
	logger *zap.Logger

	events <-chan data.Event

	store store.Storer

	faults     FaultRecord
	faultsLock sync.Mutex
}

func NewAnalyzer(logger *zap.Logger, relays []*builder.Client, events <-chan data.Event, store store.Storer) *Analyzer {
	faults := make(FaultRecord)
	for _, relay := range relays {
		faults[relay.PublicKey] = &Faults{}
	}
	return &Analyzer{
		logger: logger,
		events: events,
		store:  store,
		faults: faults,
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

func (a *Analyzer) processBid(ctx context.Context, event *data.BidEvent) {
	logger := a.logger.Sugar()

	bidCtx := event.Context
	bid := event.Bid

	err := a.store.PutBid(ctx, bidCtx, bid)
	if err != nil {
		logger.Warn("could not store bid: %+v", event)
		return
	}

	// TODO dispatch event to analyze bid

	relayID := bidCtx.RelayPublicKey
	a.faultsLock.Lock()
	faults := a.faults[relayID]
	faults.TotalBids += 1
	a.faultsLock.Unlock()
}

func (a *Analyzer) processValidatorRegistration(ctx context.Context, event data.ValidatorRegistrationEvent) {
	logger := a.logger.Sugar()

	// TODO validations on data

	registrations := event.Registrations
	for _, registration := range registrations {
		err := a.store.PutValidatorRegistration(ctx, registration)
		if err != nil {
			logger.Warn("could not store validator registration: %+v", registration)
			return
		}
	}
}

func (a *Analyzer) processAuctionTranscript(ctx context.Context, event data.AuctionTranscriptEvent) {
	logger := a.logger.Sugar()

	// TODO validations on data
	// - validate bid correlates with acceptance, otherwise consider a count against the proposer

	bid := &event.Transcript.Bid
	acceptance := &event.Transcript.Acceptance
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
		logger.Warn("could not store bid: %+v", event)
		return
	}

	err = a.store.PutAcceptance(ctx, bidCtx, acceptance)
	if err != nil {
		logger.Warn("could not store bid acceptance data: %+v", event)
		return
	}

	// TODO dispatch event to analyze transcript
}

func (a *Analyzer) Run(ctx context.Context) error {
	logger := a.logger.Sugar()

	for {
		select {
		case event := <-a.events:
			logger.Debugf("got event: %+v", event.Payload)

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
