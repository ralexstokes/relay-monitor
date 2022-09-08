package data

import (
	"context"

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

func (c *Collector) collectBidFromRelay(ctx context.Context, relay *builder.Client, slot types.Slot) (*BidEvent, error) {
	parentHash, err := c.consensusClient.GetParentHash(ctx, slot)
	if err != nil {
		return nil, err
	}
	publicKey, err := c.consensusClient.GetProposerPublicKey(ctx, slot)
	if err != nil {
		return nil, err
	}
	bid, exists, err := relay.GetBid(slot, parentHash, *publicKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	bidCtx := types.BidContext{
		Slot:              slot,
		ParentHash:        parentHash,
		ProposerPublicKey: *publicKey,
		RelayPublicKey:    relay.PublicKey,
	}
	event := &BidEvent{Context: &bidCtx, Bid: bid}
	return event, nil
}

func (c *Collector) collectFromRelay(ctx context.Context, relay *builder.Client) {
	logger := c.logger.Sugar()

	relayID := relay.PublicKey

	slots := c.clock.TickSlots(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case slot := <-slots:
			payload, err := c.collectBidFromRelay(ctx, relay, slot)
			if err != nil {
				logger.Warnw("could not get bid from relay", "error", err, "relayPublicKey", relayID, "slot", slot)
				// TODO implement some retry logic...
				continue
			}
			if payload == nil {
				// No bid for this slot, continue
				// TODO consider trying again...
				continue
			}
			logger.Debugw("got bid", "relay", relayID, "context", payload.Context, "bid", payload.Bid)
			// TODO what if this is slow
			c.events <- Event{Payload: payload}
		}
	}
}

func (c *Collector) syncExecutionHeads(ctx context.Context) {
	logger := c.logger.Sugar()

	heads := c.consensusClient.StreamHeads(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case head := <-heads:
			_, err := c.consensusClient.FetchExecutionHash(ctx, head.Slot)
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

func (c *Collector) collectConsensusData(ctx context.Context) {
	go c.syncExecutionHeads(ctx)
	go c.syncProposers(ctx)
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
