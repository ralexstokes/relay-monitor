package types

import (
	"github.com/flashbots/go-boost-utils/types"
)

type (
	Slot  = uint64
	Epoch = uint64
)

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
	Slot              Slot
	ParentHash        Hash
	ProposerPublicKey PublicKey
	RelayPublicKey    PublicKey
}

type SignedBlindedBeaconBlock = types.SignedBlindedBeaconBlock
