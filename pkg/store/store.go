package store

import (
	"context"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

// Interface for the state "storer" that a relay monitor can use for storing data.
type Storer interface {
	PutBid(context.Context, *types.BidContext, *types.Bid) error
	PutValidatorRegistration(context.Context, *types.SignedValidatorRegistration) error
	PutAcceptance(context.Context, *types.BidContext, *types.SignedBlindedBeaconBlock) error

	GetBid(context.Context, *types.BidContext) (*types.Bid, error)

	// `GetValidatorRegistrations` returns all known registrations for the validator's public key, sorted by timestamp (increasing).
	GetValidatorRegistrations(context.Context, *types.PublicKey) ([]*types.SignedValidatorRegistration, error)
	// `GetLatestValidatorRegistration` returns the latest known registration for the validator's public key.
	GetLatestValidatorRegistration(context.Context, *types.PublicKey) (*types.SignedValidatorRegistration, error)
}
