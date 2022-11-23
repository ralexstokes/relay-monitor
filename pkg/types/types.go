package types

import (
	"bytes"
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
	ExecutionPayload            = types.ExecutionPayload
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
	Slot              Slot      `json:"slot"`
	ParentHash        Hash      `json:"parent_hash"`
	ProposerPublicKey PublicKey `json:"proposer_public_key"`
	RelayPublicKey    PublicKey `json:"relay_public_key"`
}

func (c BidContext) MarshalText() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteString(fmt.Sprintf("(%d, %s, %s)", c.Slot, c.ParentHash, c.ProposerPublicKey))
	return buf.Bytes(), nil
}
