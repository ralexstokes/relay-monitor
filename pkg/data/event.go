package data

import "github.com/ralexstokes/relay-monitor/pkg/types"

type Event struct {
	Payload any
}

type BidEvent struct {
	Context *types.BidContext
	// A `nil` `Bid` indicates absence for the given `Context`
	Bid *types.VersionedBid
}

type ValidatorRegistrationEvent struct {
	Registrations []types.SignedValidatorRegistration
}

type AuctionTranscriptEvent struct {
	Transcript *types.AuctionTranscript
}
