package analysis

import (
	"context"
	"sync"

	"github.com/ralexstokes/relay-monitor/pkg/builder"
	"github.com/ralexstokes/relay-monitor/pkg/data"
	"go.uber.org/zap"
)

type Analyzer struct {
	logger *zap.Logger

	events <-chan data.Event

	faults     FaultRecord
	faultsLock sync.Mutex
}

func NewAnalyzer(logger *zap.Logger, relays []*builder.Client, events <-chan data.Event) *Analyzer {
	faults := make(FaultRecord)
	for _, relay := range relays {
		faults[relay.PublicKey] = &Faults{}
	}
	return &Analyzer{
		logger: logger,
		events: events,
		faults: faults,
	}
}

func (a *Analyzer) GetFaults() FaultRecord {
	a.faultsLock.Lock()
	defer a.faultsLock.Unlock()

	faults := make(FaultRecord)
	for relay, summary := range a.faults {
		summary := *summary
		faults[relay] = &summary
	}

	return faults
}

func (a *Analyzer) Run(ctx context.Context) error {
	logger := a.logger.Sugar()

	for {
		select {
		case event := <-a.events:
			logger.Debugf("got event: %v", event)

			switch event := event.Payload.(type) {
			case data.BidEvent:
				relayID := event.Relay
				a.faultsLock.Lock()
				faults := a.faults[relayID]
				faults.ValidBids += 1
				a.faultsLock.Unlock()
			case data.ValidatorRegistrationEvent:
				// TODO validations on data
			case data.AuctionTranscriptEvent:
				// TODO validations on data
				// inspect faults
			}
		case <-ctx.Done():
			return nil
		}
	}
}
