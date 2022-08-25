package consensus

import (
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

type Clock struct {
	genesisTime    uint64
	slotsPerSecond uint64
	slotsPerEpoch  uint64
}

func NewClock(genesisTime uint64, slotsPerSecond uint64, slotsPerEpoch uint64) *Clock {
	return &Clock{
		genesisTime:    genesisTime,
		slotsPerSecond: slotsPerSecond,
		slotsPerEpoch:  slotsPerEpoch,
	}
}

func (c *Clock) slotInSeconds(slot types.Slot) int64 {
	return int64(slot*uint64(c.slotsPerSecond) + c.genesisTime)
}

func (c *Clock) currentSlot(currentTime int64) types.Slot {
	diff := currentTime - int64(c.genesisTime)
	return types.Slot(diff / int64(c.slotsPerSecond))
}

func (c *Clock) TickSlots() chan types.Slot {
	ch := make(chan types.Slot, 1)
	go func() {
		for {
			now := time.Now().Unix()
			currentSlot := c.currentSlot(now)
			ch <- currentSlot
			nextSlot := currentSlot + 1
			nextSlotStart := c.slotInSeconds(nextSlot)
			duration := time.Duration(nextSlotStart - now)
			time.Sleep(duration * time.Second)
		}
	}()
	return ch
}

func (c *Clock) TickEpochs() chan types.Epoch {
	ch := make(chan types.Epoch, 1)
	go func() {
		slots := c.TickSlots()
		currentSlot := <-slots
		currentEpoch := currentSlot / c.slotsPerEpoch
		ch <- currentEpoch
		for slot := range c.TickSlots() {
			epoch := slot / c.slotsPerEpoch
			if epoch > currentEpoch {
				currentEpoch = epoch
				ch <- currentEpoch
			}
		}
	}()
	return ch
}
