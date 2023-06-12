package types

import (
	"fmt"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/holiman/uint256"
)

type (
	Slot                        = uint64
	Epoch                       = uint64
	ForkVersion                 = types.ForkVersion
	Uint256                     = uint256.Int
	PublicKey                   = types.PublicKey
	Hash                        = types.Hash
	Bid                         = types.SignedBuilderBid
	Root                        = types.Root
	ValidatorIndex              = uint64
	SignedValidatorRegistration = types.SignedValidatorRegistration
	SignedBlindedBeaconBlock    = types.SignedBlindedBeaconBlock
)

type Coordinate struct {
	Slot Slot
	Root Root
}

type AuctionTranscript struct {
	Bid        Bid                            `json:"bid"`
	Acceptance types.SignedBlindedBeaconBlock `json:"acceptance"`
}

type BidContext struct {
	Slot              Slot      `json:"slot,omitempty"`
	ParentHash        Hash      `json:"parent_hash,omitempty"`
	ProposerPublicKey PublicKey `json:"proposer_public_key,omitempty"`
	RelayPublicKey    PublicKey `json:"relay_public_key,omitempty"`
	Error             error     `json:"error,omitempty"`
}

type ErrorType string

const (
	ParentHashErr ErrorType = "ParentHashError"
	PubKeyErr     ErrorType = "PublicKeyError"
	EmptyBidError ErrorType = "EmptyBidError"
	RelayError    ErrorType = "RelayError"
	ValidationErr ErrorType = "ValidationError"
)

type ClientError struct {
	Type    ErrorType `json:"errorType,omitempty"`
	Code    int       `json:"code,omitempty"`
	Message string    `json:"message,omitempty"`
}

func (e *ClientError) Error() string {
	return fmt.Sprintf("Type: %s Code: %d Message: %s", e.Type, e.Code, e.Message)
}
