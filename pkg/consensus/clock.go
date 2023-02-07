package consensus

import (
	"context"
	"fmt"
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

func (c *Clock) SlotInSeconds(slot types.Slot) int64 {
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
			now := time.Now().Unix()
			currentSlot := c.CurrentSlot(now)
			ch <- currentSlot
			nextSlot := currentSlot + 1
			nextSlotStart := c.SlotInSeconds(nextSlot)
			duration := time.Duration(nextSlotStart - now)
			select {
			case <-time.After(duration * time.Second):
			case <-ctx.Done():
				close(ch)
				return
			}
		}
	}()
	return ch
}

// Tick multiple times per slot, tickPerSlot MUST be less than 13
func (c *Clock) MultiTickSlots(ctx context.Context, tickPerSlot int) chan types.Slot {
	ch := make(chan types.Slot, 1)
	go func() {
		now := time.Now().Unix()
		currentSlot := c.CurrentSlot(now)
		ch <- currentSlot
		nextSlot := currentSlot + 1
		nextSlotStart := c.SlotInSeconds(nextSlot)
		duration := time.Duration(nextSlotStart - now)
		select {
		case <-time.After(duration * time.Second):
			c.multiTickSlots(ctx, tickPerSlot, ch)
		case <-ctx.Done():
			close(ch)
			return
		}
	}()
	return ch
}

// Tick multiple times per slot, then wait until the next slot
func (c *Clock) multiTickSlots(ctx context.Context, tickPerSlot int, ch chan types.Slot) {
	duration := c.secondsPerSlot / uint64(tickPerSlot)
	go func() {
		for {
			now := time.Now().Unix()
			currentSlot := c.CurrentSlot(now)

			// Send a slot right away, then sleep before sending again
			ch <- currentSlot
			for i := 0; i < tickPerSlot-1; i++ {
				select {
				case <-time.After(time.Duration(duration) * time.Second):
					fmt.Printf("===== current_slot_start: %d, current_time: %d next_slot_start: %d slot: %d, expected_slot: %d \n", c.SlotInSeconds(currentSlot), time.Now().Unix(), c.SlotInSeconds(currentSlot+1), currentSlot, c.CurrentSlot(time.Now().Unix()))
					ch <- currentSlot
				case <-ctx.Done():
					close(ch)
					return
				}
			}

			// Sleep the remainder of the time until the next slot
			now = time.Now().Unix()
			nextSlot := currentSlot + 1
			nextSlotStart := c.SlotInSeconds(nextSlot)
			duration := time.Duration(nextSlotStart - now)
			if duration > 0 {
				select {
				case <-time.After(duration * time.Second):
				case <-ctx.Done():
					close(ch)
					return
				}
			}
		}
	}()
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
		close(ch)
	}()
	return ch
}
