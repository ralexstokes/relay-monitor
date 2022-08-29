package types

import (
	"github.com/flashbots/go-boost-utils/types"
	primitives "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

type PublicKey = types.PublicKey

type Hash = types.Hash

type Bid = types.SignedBuilderBid

type Root = types.Root

type ValidatorIndex = uint64

type Coordinate struct {
	Slot primitives.Slot
	Root Root
}
