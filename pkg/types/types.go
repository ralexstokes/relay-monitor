package types

import (
	consensus "github.com/umbracle/go-eth-consensus"
	"github.com/umbracle/go-eth-consensus/http"
)

type (
	Slot  = uint64
	Epoch = uint64
)

type PublicKey = [48]byte

type Hash = [32]byte

type Bid = http.SignedBuilderBid

type Root = [32]byte

type ValidatorIndex = uint64

type Coordinate struct {
	Slot Slot
	Root Root
}

type SignedValidatorRegistration = http.SignedValidatorRegistration

type AuctionTranscript struct {
	Bid        Bid                                `json:"bid"`
	Acceptance consensus.SignedBlindedBeaconBlock `json:"acceptance"`
}

type BidContext struct {
	Slot              Slot      `json:"slot"`
	ParentHash        Hash      `json:"parent_hash"`
	ProposerPublicKey PublicKey `json:"proposer_public_key"`
	RelayPublicKey    PublicKey `json:"relay_public_key"`
}

type SignedBlindedBeaconBlock = consensus.SignedBlindedBeaconBlock
