package store

import (
	"context"
	"time"

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
	// `GetCountValidatorsRegistrations`returns the total number of valid registrations processed.
	GetCountValidatorsRegistrations(ctx context.Context) (uint, error)
	// `GetCountValidators`returns the number of validators that have successfully submitted at least one registration.
	GetCountValidators(ctx context.Context) (uint, error)

	// TODO: look to refactor the way analysis is stored by referencing the bid in TableBids
	// with a foreing key / etc. in order to not store all 4 context fields that act as the
	// unique ID, since this is way too much data duplication.
	// Need to refactor Bid insert/fetch to return the ID.

	// Something like this?
	//
	// -- reference to the bid entry
	// bid_id                  bigint NOT NULL,

	// CONSTRAINT fk_bid
	// 		FOREIGN KEY(bid_id)
	// 			REFERENCES ` + TableBids + `(id)
	//
	// For now use the bid context and do a JOIN on these with the Bids table if want more Bid data
	// fields.
	PutBidAnalysis(context.Context, *types.BidContext, *types.InvalidBid) error

	PutRelay(context.Context, *types.Relay) error
	GetRelay(context.Context, *types.PublicKey) (*types.Relay, error)
	GetRelays(context.Context) ([]*types.Relay, error)

	// Analysis metrics getters.
	GetCountAnalysisLookbackSlots(ctx context.Context, lookbackSlots uint64, filter *types.AnalysisQueryFilter) (uint64, error)
	GetCountAnalysisLookbackDuration(ctx context.Context, lookbackDuration time.Duration, filter *types.AnalysisQueryFilter) (uint64, error)

	// Stats and record getters within slot bounds.
	GetCountAnalysisWithinSlotBounds(ctx context.Context, relayPubkey string, slotBounds *types.SlotBounds, filter *types.AnalysisQueryFilter) (uint64, error)
	GetRecordsAnalysisWithinSlotBounds(ctx context.Context, relayPubkey string, slotBounds *types.SlotBounds, filter *types.AnalysisQueryFilter) ([]*types.Record, error)
}
