package data

import (
	"context"
	"fmt"

	"github.com/avast/retry-go"
	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

type Collector struct {
	logger          *zap.Logger
	relays          []*builder.Client
	clock           *consensus.Clock
	consensusClient *consensus.Client
	events          chan<- Event
}

func NewCollector(zapLogger *zap.Logger, relays []*builder.Client, clock *consensus.Clock, consensusClient *consensus.Client, events chan<- Event) *Collector {
	return &Collector{
		logger:          zapLogger,
		relays:          relays,
		clock:           clock,
		consensusClient: consensusClient,
		events:          events,
	}
}

// collectBidFromRelay does most of the work for collecting the bid from the relay and getting
// necessary "context" for the bid from the connected consensus client. If any step fails and
// results in an error, or if the call to get the bid from the relay (via the `getHeader`
// endpoint) does not return a bid, the error will be handled by the "retrier" function that will
// effectivelly schedule another call to collectBidFromRelay to perform the same steps.
func (collector *Collector) collectBidFromRelay(ctx context.Context, relay *builder.Client, slot types.Slot) (*BidEvent, error) {
	logger := collector.logger.Sugar()

	parentHash, err := collector.consensusClient.GetParentHash(ctx, slot)
	if err != nil {
		return nil, err
	}

	publicKey, err := collector.consensusClient.GetProposerPublicKey(ctx, slot)
	if err != nil {
		return nil, err
	}

	// Request a bid via the `getHeader()` endpoint.
	bid, err := relay.GetBid(slot, parentHash, *publicKey)
	if err != nil {
		return nil, err
	}

	// If there were no errors returned but the bid is `nil`, then the relay does not have a bid for
	// this slot. The 'retrier' function will retry regardless.
	if bid == nil {
		return nil, nil
	}

	bidCtx := types.BidContext{
		Slot:              slot,
		ParentHash:        parentHash,
		ProposerPublicKey: *publicKey,
		RelayPublicKey:    relay.PublicKey,
	}

	logger.Debugw("collected bid", "relay", relay.PublicKey, "context", bidCtx, "bid", bid)

	bidEvent := &BidEvent{
		Context: &bidCtx,
		Bid:     bid,
	}

	// Send in the event to get processed / analyzed.
	// TODO: instead pipe this to DB and have the analyzer read from DB (WIP)
	collector.events <- Event{Payload: bidEvent}

	return bidEvent, nil
}

// tryCollectBidFromRelay is a wrapper around collectBidFromRelay that handles the retry logic.
func (collector *Collector) tryCollectBidFromRelay(ctx context.Context, relay *builder.Client, slot types.Slot) {
	logger := collector.logger.Sugar()

	// Try to fetch the bid from client. If unavailable, setup a retry.
	retry.Do(
		func() error {
			var err error

			// Try to collect the bid.
			bid, err := collector.collectBidFromRelay(ctx, relay, slot)
			if err != nil {
				logger.Warnw("could not get bid from relay", "slot", slot, "error", err, "relay", relay.PublicKey, "retrying", RetryDelay)
				return err
			}
			// There is no error but also no bid, so we need to retry still.
			if bid == nil {
				logger.Warnw("relay does not have bid", "slot", slot, "relay", relay.PublicKey, "retrying", RetryDelay)
				return fmt.Errorf("relay did not have bid for slot")
			}

			// If no error, then no need to re-try, the bid has been collected.
			return nil
		},
		CollectorDelayType,
		CollectorRetryAttempts,
		CollectorRetryDelay,
	)
}

// collectFromRelay is the main loop for collecting bids from a relay. It will try to collect a bid
// from the relay for each slot.
func (collector *Collector) collectFromRelay(ctx context.Context, relay *builder.Client) {
	logger := collector.logger.Sugar()

	// A stream of slots.
	slots := collector.clock.TickSlots(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case slot := <-slots:
			// New slot -- try to collect a bid from the relay.
			logger.Infow("new slot, try to collect bid", "slot", slot, "relay", relay.PublicKey)

			go collector.tryCollectBidFromRelay(ctx, relay, slot)
		}
	}
}

func (c *Collector) syncBlocks(ctx context.Context) {
	logger := c.logger.Sugar()

	heads := c.consensusClient.StreamHeads(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case head := <-heads:
			err := c.consensusClient.FetchBlock(ctx, head.Slot)
			if err != nil {
				logger.Warnf("could not fetch latest execution hash for slot %d: %v", head.Slot, err)
			}
		}
	}
}

func (c *Collector) syncProposers(ctx context.Context) {
	logger := c.logger.Sugar()

	epochs := c.clock.TickEpochs(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case epoch := <-epochs:
			err := c.consensusClient.FetchProposers(ctx, epoch+1)
			if err != nil {
				logger.Warnf("could not load consensus state for epoch %d: %v", epoch, err)
			}
		}
	}
}

func (c *Collector) syncValidators(ctx context.Context) {
	logger := c.logger.Sugar()

	epochs := c.clock.TickEpochs(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case epoch := <-epochs:
			err := c.consensusClient.FetchValidators(ctx)
			if err != nil {
				logger.Warnf("could not load validators in epoch %d: %v", epoch, err)
			}
		}
	}
}

// TODO refactor this into a separate component as the list of duties is growing outside the "collector" abstraction
func (c *Collector) collectConsensusData(ctx context.Context) {
	go c.syncBlocks(ctx)
	go c.syncProposers(ctx)
	go c.syncValidators(ctx)
}

func (c *Collector) Run(ctx context.Context) error {
	logger := c.logger.Sugar()

	for _, relay := range c.relays {
		relayID := relay.PublicKey
		logger.Infof("monitoring relay %s", relayID)

		go c.collectFromRelay(ctx, relay)
	}
	go c.collectConsensusData(ctx)

	<-ctx.Done()
	return nil
}
