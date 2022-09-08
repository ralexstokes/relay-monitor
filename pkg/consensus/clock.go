package consensus

import (
	"context"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

type Clock struct {
	genesisTime    uint64
	secondsPerSlot uint64
	slotsPerEpoch  uint64
}

func NewClock(genesisTime, secondsPerSlot, slotsPerEpoch uint64) *Clock {
	return &Clock{
		genesisTime:    genesisTime,
		secondsPerSlot: secondsPerSlot,
		slotsPerEpoch:  slotsPerEpoch,
	}
}

func (c *Clock) slotInSeconds(slot types.Slot) int64 {
	return int64(slot*c.secondsPerSlot + c.genesisTime)
}

func (c *Clock) CurrentSlot(currentTime int64) types.Slot {
	diff := currentTime - int64(c.genesisTime)
	// TODO better handling of pre-genesis
	if diff < 0 {
		return 0
	}
	return types.Slot(diff / int64(c.secondsPerSlot))
}

func (c *Clock) EpochForSlot(slot types.Slot) types.Epoch {
	return slot / c.slotsPerEpoch
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
			now := time.Now().Unix()
			currentSlot := c.CurrentSlot(now)
			ch <- currentSlot
			nextSlot := currentSlot + 1
			nextSlotStart := c.slotInSeconds(nextSlot)
			duration := time.Duration(nextSlotStart - now)
			time.Sleep(duration * time.Second)
		}
	}()
	return ch
}

func (c *Clock) TickEpochs(ctx context.Context) chan types.Epoch {
	ch := make(chan types.Epoch, 1)
	go func() {
		slots := c.TickSlots(ctx)
		currentSlot := <-slots
		currentEpoch := currentSlot / c.slotsPerEpoch
		ch <- currentEpoch
		for slot := range slots {
			epoch := slot / c.slotsPerEpoch
			if epoch > currentEpoch {
				currentEpoch = epoch
				ch <- currentEpoch
			}
		}
	}()
	return ch
}
