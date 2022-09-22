package store

import (
	"context"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

type Storer interface {
	PutBid(context.Context, *types.BidContext, *types.Bid) error
	PutValidatorRegistration(context.Context, *types.SignedValidatorRegistration) error
	PutAcceptance(context.Context, *types.BidContext, *types.SignedBlindedBeaconBlock) error
}

type MemoryStore struct {
	bids          map[types.BidContext]types.Bid
	registrations map[types.PublicKey]*types.SignedValidatorRegistration
	acceptances   map[types.BidContext]types.SignedBlindedBeaconBlock
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		bids:          make(map[types.BidContext]types.Bid),
		registrations: make(map[types.PublicKey]*types.SignedValidatorRegistration),
	}
}

func (s *MemoryStore) PutBid(ctx context.Context, bidCtx *types.BidContext, bid *types.Bid) error {
	s.bids[*bidCtx] = *bid
	return nil
}

func (s *MemoryStore) PutValidatorRegistration(ctx context.Context, registration *types.SignedValidatorRegistration) error {
	publicKey := registration.Message.Pubkey
	s.registrations[publicKey] = registration
	return nil
}

func (s *MemoryStore) PutAcceptance(ctx context.Context, bidCtx *types.BidContext, acceptance *types.SignedBlindedBeaconBlock) error {
	s.acceptances[*bidCtx] = *acceptance
	return nil
}
