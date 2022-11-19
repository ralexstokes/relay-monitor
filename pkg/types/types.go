package types

import (
	"github.com/flashbots/go-boost-utils/types"
	"github.com/holiman/uint256"
)

type (
	Slot  = uint64
	Epoch = uint64
)

type Uint256 = uint256.Int

type PublicKey = types.PublicKey

type Hash = types.Hash

type Bid = types.SignedBuilderBid

type Root = types.Root

type ValidatorIndex = uint64

type Coordinate struct {
	Slot Slot
	Root Root
}

type SignedValidatorRegistration = types.SignedValidatorRegistration

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

type SignedBlindedBeaconBlock = types.SignedBlindedBeaconBlock
