package data

import (
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

type Event struct {
	Payload any
}

type BidEvent struct {
	Context *types.BidContext `json:",omitempty"`
	Bid     *types.Bid        `json:"-"`
	// A `nil` `Bid` indicates absence for the given `Context`
	Message interface{} `json:"Bid,omitempty"`
}

type ValidatorRegistrationEvent struct {
	Registrations []types.SignedValidatorRegistration
}

type AuctionTranscriptEvent struct {
	Transcript *types.AuctionTranscript
}

type BidOutput struct {
	Timestamp time.Time `json:",omitempty"`
	Rtt       uint64    `json:",omitempty"`
	Relay     string    `json:",omitempty"`
	Region    string    `json:",omitempty"`
	Bid       BidEvent  `json:",omitempty"`
}

type ValidationOutput struct {
	Timestamp      time.Time      `json:",omitempty"`
	RelayPublicKey string         `json:",omitempty"`
	Slot           types.Slot     `json:",omitempty"`
	Region         string         `json:",omitempty"`
	Error          *ValidationErr `json:"error,omitempty"`
}

type ValidationErr struct {
	Type     types.ErrorType `json:"errorType,omitempty"`
	Reason   string          `json:",omitempty"`
	Expected interface{}     `json:",omitempty"`
	Actual   interface{}     `json:",omitempty"`
}
