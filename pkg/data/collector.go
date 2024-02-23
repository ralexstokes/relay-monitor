package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
	"github.com/ralexstokes/relay-monitor/pkg/output"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

type Collector struct {
	logger          *zap.Logger
	relays          []*builder.Client
	clock           *consensus.Clock
	consensusClient *consensus.Client
	events          chan<- Event
	output          *output.Output
	region          string
}

func NewCollector(zapLogger *zap.Logger, relays []*builder.Client, clock *consensus.Clock, consensusClient *consensus.Client, output *output.Output, region string, events chan<- Event) *Collector {
	return &Collector{
		logger:          zapLogger,
		relays:          relays,
		clock:           clock,
		consensusClient: consensusClient,
		events:          events,
		output:          output,
		region:          region,
	}
}

func (c *Collector) outputBid(event *BidEvent, duration *uint64, relay *builder.Client) {

	go func() {
		logger := c.logger.Sugar()

		out := &BidOutput{
			Timestamp: time.Unix(c.clock.SlotInSeconds(phase0.Slot(event.Context.Slot)), 0),
			Rtt:       *duration,
			Bid:       *event,
			Relay:     relay.Endpoint(),
			Region:    c.region,
		}

		outBytes, err := json.Marshal(out)
		if err != nil {
			logger.Warnw("unable to marshal outout", "error", err, "content", out)
		}
		err = c.output.WriteEntry(outBytes)
		if err != nil {
			logger.Warnw("unable to write output", "error", err)
		}

	}()
}

func (c *Collector) collectBidFromRelay(ctx context.Context, relay *builder.Client, slot types.Slot) (*BidEvent, error) {
	var duration *uint64 = new(uint64)
	var bid *types.Bid

	bidCtx := types.BidContext{
		Slot:           uint64(slot),
		RelayPublicKey: relay.PublicKey,
	}

	event := &BidEvent{
		Context: &bidCtx,
	}
	defer c.outputBid(event, duration, relay)

	parentHash, err := c.consensusClient.GetParentHash(ctx, slot)
	if err != nil {
		bidCtx.Error = &types.ClientError{Type: types.ParentHashErr, Code: 500, Message: "Unable to get parent hash"}
		return nil, err
	}
	bidCtx.ParentHash = parentHash

	publicKey, err := c.consensusClient.GetProposerPublicKey(ctx, slot)
	if err != nil {
		bidCtx.Error = &types.ClientError{Type: types.PubKeyErr, Code: 500, Message: "Unable to get proposer public key"}
		return nil, err
	}
	bidCtx.ProposerPublicKey = *publicKey

	bid, *duration, err = relay.GetBid(slot, parentHash, *publicKey)
	if err != nil {
		bidCtx.Error = err
		return nil, err
	}
	if bid == nil {
		bidCtx.Error = &types.ClientError{Type: types.EmptyBidError, Code: 204, Message: "No bid returned"}
		return nil, nil
	}

	event.Bid = bid
	event.Message, _ = bid.Message()

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

// TODO refactor this into a separate component as the list of duties is growing outside the "collector" abstraction
func (c *Collector) collectConsensusData(ctx context.Context) {
	go c.syncBlocks(ctx)
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
