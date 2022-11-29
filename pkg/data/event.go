package data

import (
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

type Event struct {
	Payload any
}

type BidEvent struct {
	Context *types.BidContext
	// A `nil` `Bid` indicates absence for the given `Context`
	Bid *types.Bid
}

type ValidatorRegistrationEvent struct {
	Registrations []types.SignedValidatorRegistration
}

type AuctionTranscriptEvent struct {
	Transcript *types.AuctionTranscript
}

type Output struct {
	Timestamp time.Time
	Rtt       uint64
	Relay     string
	Region    string
	Bid       BidEvent
}
