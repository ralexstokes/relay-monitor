package consensus

import (
	"context"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/types"
	"github.com/umbracle/go-eth-consensus/chaintime"
)

type Clock struct {
	chaintime *chaintime.Chaintime
}

func NewClock(genesisTime, secondsPerSlot, slotsPerEpoch uint64) *Clock {
	return &Clock{
		chaintime: chaintime.New(time.Time{}, secondsPerSlot, slotsPerEpoch),
	}
}

func (c *Clock) CurrentSlot(currentTime int64) types.Slot {
	return c.chaintime.CurrentSlot().Number
}

func (c *Clock) EpochForSlot(slot types.Slot) types.Epoch {
	return c.chaintime.Slot(slot).Epoch
}

func (c *Clock) TickSlots(ctx context.Context) chan types.Slot {
	ch := make(chan types.Slot, 1)
	go func() {
		for {
			// TODO do not block if we are in the middle of the sleep at the end of this loop...
			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
			}

			slot := c.chaintime.CurrentSlot()
			ch <- slot.Number

			nextSlot := c.chaintime.Slot(slot.Number + 1)
			<-nextSlot.C().C
		}
	}()
	return ch
}

func (c *Clock) TickEpochs(ctx context.Context) chan types.Epoch {
	ch := make(chan types.Epoch, 1)
	go func() {
		for {
			epoch := c.chaintime.CurrentEpoch()
			ch <- epoch.Number

			nextEpoch := c.chaintime.Epoch(epoch.Number + 1)
			<-nextEpoch.C().C
		}
	}()
	return ch
}
