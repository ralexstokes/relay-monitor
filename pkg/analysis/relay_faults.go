package analysis

import "github.com/ralexstokes/relay-monitor/pkg/types"

type FaultRecord = map[types.PublicKey]*Faults

type Faults struct {
	Stats *FaultStats `json:"stats"`
	Misc  *Misc       `json:"misc"`
}

type FaultStats struct {
	TotalBids                uint `json:"total_bids"`
	MalformedBids            uint `json:"malformed_bids"`
	ConsensusInvalidBids     uint `json:"consensus_invalid_bids"`
	PaymentInvalidBids       uint `json:"payment_invalid_bids"`
	IgnoredPreferencesBids   uint `json:"ignored_preferences_bids"`
	MalformedPayloads        uint `json:"malformed_payloads"`
	ConsensusInvalidPayloads uint `json:"consensus_invalid_payloads"`
	UnavailablePayloads      uint `json:"unavailable_payloads"`
}

type Misc struct {
	Endpoint string `json:"endpoint"`
}
