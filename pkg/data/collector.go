package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	output          *output.FileOutput
	region          string
}

func NewCollector(zapLogger *zap.Logger, relays []*builder.Client, clock *consensus.Clock, consensusClient *consensus.Client, output *output.FileOutput, region string, events chan<- Event) *Collector {
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
	logger := c.logger.Sugar()

	out := &Output{
		Timestamp: time.Now(),
		Rtt:       *duration,
		Bid:       *event,
		Relay:     relay.Endpoint(),
		Region:    c.region,
	}

	outBytes, err := json.Marshal(out)
	if err != nil {
		logger.Warnw("unable to marshal outout", "error", err, "content", out)
	} else {
		outBytes = append(outBytes, []byte("\n")...)
		err = c.output.WriteEntry(outBytes)
		if err != nil {
			logger.Warnw("unable to write output", "error", err)
		}
	}
}

func (c *Collector) collectBidFromRelay(ctx context.Context, relay *builder.Client, slot types.Slot) (*BidEvent, error) {
	// logger := c.logger.Sugar()
	var duration *uint64 = new(uint64)
	// *duration = 0
	var bid *types.Bid

	bidCtx := types.BidContext{
		Slot: slot,
		// ParentHash:        parentHash,
		// ProposerPublicKey: *publicKey,
		RelayPublicKey: relay.PublicKey,
	}

	event := &BidEvent{}
	defer c.outputBid(event, duration, relay)

	parentHash, err := c.consensusClient.GetParentHash(ctx, slot)
	if err != nil {
		bidCtx.Error = err
		return nil, err
	}
	bidCtx.ParentHash = parentHash

	publicKey, err := c.consensusClient.GetProposerPublicKey(ctx, slot)
	if err != nil {
		bidCtx.Error = err
		return nil, err
	}
	bidCtx.ProposerPublicKey = *publicKey

	bid, *duration, err = relay.GetBid(slot, parentHash, *publicKey)
	if err != nil {
		bidCtx.Error = err
		return nil, err
	}
	if bid == nil {
		bidCtx.Error = fmt.Errorf("no bid returned")
		return nil, nil
	}

	event.Context = &bidCtx
	event.Bid = bid

	// event := &BidEvent{Context: &bidCtx, Bid: bid}

	// out := &Output{
	// 	Timestamp: time.Now(),
	// 	Rtt:       duration,
	// 	Bid:       *event,
	// 	Relay:     relay.Endpoint(),
	// 	Region:    c.region,
	// }

	// outBytes, err := json.Marshal(out)
	// if err != nil {
	// 	logger.Warnw("unable to marshal outout", "error", err, "content", out)
	// } else {
	// 	outBytes = append(outBytes, []byte("\n")...)
	// 	err = c.output.WriteEntry(outBytes)
	// 	if err != nil {
	// 		logger.Warnw("unable to write output", "error", err)
	// 	}
	// }

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
