package analysis

import "github.com/ralexstokes/relay-monitor/pkg/types"

type FaultRecord = map[types.PublicKey]*Faults

type Faults struct {
	ValidBids                uint `json:"valid_bids"`
	MalformedBids            uint `json:"malformed_bids"`
	ConsensusInvalidBids     uint `json:"consensus_invalid_bids"`
	PaymentInvalidBids       uint `json:"payment_invalid_bids"`
	NonconformingBids        uint `json:"nonconforming_bids"`
	MalformedPayloads        uint `json:"malformed_payloads"`
	ConsensusInvalidPayloads uint `json:"consensus_invalid_payloads"`
	UnavailablePayloads      uint `json:"unavailable_payloads"`
}
