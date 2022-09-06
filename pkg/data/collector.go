package data

import (
	"context"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/consensus"
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

func (c *Collector) collectFromRelay(ctx context.Context, relay *builder.Client) {
	logger := c.logger.Sugar()

	relayID := relay.PublicKey
	logger.Infof("monitoring relay %s", relayID)

	slots := c.clock.TickSlots()
	for {
		select {
		case <-ctx.Done():
			return
		case slot := <-slots:
			parentHash, err := c.consensusClient.GetParentHash(slot)
			if err != nil {
				logger.Warnw("error fetching bid", "error", err)
				continue
			}
			publicKey, err := c.consensusClient.GetProposerPublicKey(slot)
			if err != nil {
				logger.Warnw("error fetching bid", "error", err)
				continue
			}
			bid, err := relay.GetBid(slot, parentHash, *publicKey)
			if err != nil {
				logger.Warnw("could not get bid from relay", "error", err, "relayPublicKey", relayID, "slot", slot, "parentHash", parentHash, "proposer", publicKey)
			} else if bid != nil {
				logger.Debugw("got bid", "value", bid.Message.Value, "header", bid.Message.Header, "publicKey", bid.Message.Pubkey, "id", relayID)
				payload := BidEvent{Relay: relayID, Bid: bid}
				// TODO what if this is slow
				c.events <- Event{Payload: payload}
			}
		}
	}
}

func (c *Collector) runSlotTasks(ctx context.Context) {
	logger := c.logger.Sugar()

	// Load data for the previous slot
	now := time.Now().Unix()
	currentSlot := c.clock.CurrentSlot(now)
	_, err := c.consensusClient.FetchExecutionHash(currentSlot - 1)
	if err != nil {
		logger.Warnf("could not fetch latest execution hash for slot %d: %v", currentSlot, err)
	}

	// Load data for the current slot
	_, err = c.consensusClient.FetchExecutionHash(currentSlot)
	if err != nil {
		logger.Warnf("could not fetch latest execution hash for slot %d: %v", currentSlot, err)
	}

	heads := c.consensusClient.StreamHeads()
	for {
		select {
		case <-ctx.Done():
			return
		case head := <-heads:
			_, err := c.consensusClient.FetchExecutionHash(head.Slot)
			if err != nil {
				logger.Warnf("could not fetch latest execution hash for slot %d: %v", head.Slot, err)
			}
		}
	}
}

func (c *Collector) runEpochTasks(ctx context.Context) {
	logger := c.logger.Sugar()

	epochs := c.clock.TickEpochs()

	// Load data for the current epoch
	epoch := <-epochs
	err := c.consensusClient.LoadData(epoch)
	if err != nil {
		logger.Warnf("could not load consensus state for epoch %d: %v", epoch, err)
	}

	// Load data for the next epoch, as we will typically do
	err = c.consensusClient.LoadData(epoch + 1)
	if err != nil {
		logger.Warnf("could not load consensus state for epoch %d: %v", epoch, err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case epoch := <-epochs:
			err := c.consensusClient.LoadData(epoch + 1)
			if err != nil {
				logger.Warnf("could not load consensus state for epoch %d: %v", epoch, err)
			}
		}
	}
}

func (c *Collector) collectConsensusData(ctx context.Context) {
	go c.runSlotTasks(ctx)
	go c.runEpochTasks(ctx)
}

func (c *Collector) Run(ctx context.Context) error {
	for _, relay := range c.relays {
		go c.collectFromRelay(ctx, relay)
	}
	go c.collectConsensusData(ctx)

	<-ctx.Done()
	return nil
}
