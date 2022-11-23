package analysis

import "github.com/ralexstokes/relay-monitor/pkg/types"

type RelayStats = map[types.PublicKey]*Stats

type FaultRecord struct {
	ConsensusInvalidBids   uint `json:"consensus_invalid_bids"`
	IgnoredPreferencesBids uint `json:"ignored_preferences_bids"`

	PaymentInvalidBids       uint `json:"payment_invalid_bids"`
	MalformedPayloads        uint `json:"malformed_payloads"`
	ConsensusInvalidPayloads uint `json:"consensus_invalid_payloads"`
	UnavailablePayloads      uint `json:"unavailable_payloads"`
}

type Stats struct {
	TotalBids uint `json:"total_bids"`

	Faults map[types.BidContext]FaultRecord `json:"faults"`
}
