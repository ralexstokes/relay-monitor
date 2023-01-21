package store

import (
	"context"
	"fmt"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

type MemoryStore struct {
	bids          map[types.BidContext]*types.Bid
	registrations map[types.PublicKey][]*types.SignedValidatorRegistration
	acceptances   map[types.BidContext]types.SignedBlindedBeaconBlock
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		bids:          make(map[types.BidContext]*types.Bid),
		registrations: make(map[types.PublicKey][]*types.SignedValidatorRegistration),
		acceptances:   make(map[types.BidContext]types.SignedBlindedBeaconBlock),
	}
}

func (s *MemoryStore) PutBid(ctx context.Context, bidCtx *types.BidContext, bid *types.Bid) error {
	s.bids[*bidCtx] = bid
	return nil
}

func (s *MemoryStore) GetBid(ctx context.Context, bidCtx *types.BidContext) (*types.Bid, error) {
	bid, ok := s.bids[*bidCtx]
	if !ok {
		return nil, fmt.Errorf("could not find bid for %+v", bidCtx)
	}
	return bid, nil
}

func (s *MemoryStore) PutValidatorRegistration(ctx context.Context, registration *types.SignedValidatorRegistration) error {
	publicKey := registration.Message.Pubkey
	registrations := s.registrations[publicKey]
	registrations = append(registrations, registration)
	s.registrations[publicKey] = registrations
	return nil
}

func (s *MemoryStore) PutAcceptance(ctx context.Context, bidCtx *types.BidContext, acceptance *types.SignedBlindedBeaconBlock) error {
	s.acceptances[*bidCtx] = *acceptance
	return nil
}

func (s *MemoryStore) GetValidatorRegistrations(ctx context.Context, publicKey *types.PublicKey) ([]*types.SignedValidatorRegistration, error) {
	return s.registrations[*publicKey], nil
}

func (s *MemoryStore) GetLatestValidatorRegistration(ctx context.Context, publicKey *types.PublicKey) (*types.SignedValidatorRegistration, error) {
	registrations, err := s.GetValidatorRegistrations(ctx, publicKey)
	if err != nil {
		return nil, err
	}

	if len(registrations) == 0 {
		return nil, nil
	} else {
		currentRegistration := registrations[len(registrations)-1]
		return currentRegistration, nil
	}
}
