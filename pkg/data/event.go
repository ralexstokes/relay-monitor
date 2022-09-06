package data

import "github.com/ralexstokes/relay-monitor/pkg/types"

type Event struct {
	Payload any
}

type BidEvent struct {
	Relay types.PublicKey
	Bid   *types.Bid
}

type ValidatorRegistrationEvent struct {
	Registrations []types.SignedValidatorRegistration
}

type AuctionTranscriptEvent struct {
	Transcript *types.AuctionTranscript
}
