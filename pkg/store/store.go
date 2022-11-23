package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

type Storer interface {
	PutBid(context.Context, *types.BidContext, *types.Bid) error
	PutValidatorRegistration(context.Context, *types.SignedValidatorRegistration) error
	PutAcceptance(context.Context, *types.BidContext, *types.SignedBlindedBeaconBlock) error

	GetBid(context.Context, *types.BidContext) (*types.Bid, error)
	// `GetValidatorRegistrations` returns all known registrations for the validator's public key, sorted by timestamp (increasing).
	GetValidatorRegistrations(context.Context, *types.PublicKey) ([]types.SignedValidatorRegistration, error)
}

type MemoryStore struct {
	sync.RWMutex

	bids          map[types.BidContext]*types.Bid
	registrations map[types.PublicKey][]types.SignedValidatorRegistration
	acceptances   map[types.BidContext]types.SignedBlindedBeaconBlock
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		bids:          make(map[types.BidContext]*types.Bid),
		registrations: make(map[types.PublicKey][]types.SignedValidatorRegistration),
		acceptances:   make(map[types.BidContext]types.SignedBlindedBeaconBlock),
	}
}

func (s *MemoryStore) PutBid(ctx context.Context, bidCtx *types.BidContext, bid *types.Bid) error {
	s.Lock()
	defer s.Unlock()

	s.bids[*bidCtx] = bid
	return nil
}

func (s *MemoryStore) GetBid(ctx context.Context, bidCtx *types.BidContext) (*types.Bid, error) {
	s.RLock()
	defer s.RUnlock()

	bid, ok := s.bids[*bidCtx]
	if !ok {
		return nil, fmt.Errorf("could not find bid for %+v", bidCtx)
	}
	return bid, nil
}

func (s *MemoryStore) PutValidatorRegistration(ctx context.Context, registration *types.SignedValidatorRegistration) error {
	s.Lock()
	defer s.Unlock()

	publicKey := registration.Message.Pubkey
	registrations := s.registrations[publicKey]
	registrations = append(registrations, *registration)
	s.registrations[publicKey] = registrations
	return nil
}

func (s *MemoryStore) PutAcceptance(ctx context.Context, bidCtx *types.BidContext, acceptance *types.SignedBlindedBeaconBlock) error {
	s.Lock()
	defer s.Unlock()

	s.acceptances[*bidCtx] = *acceptance
	return nil
}

func (s *MemoryStore) GetValidatorRegistrations(ctx context.Context, publicKey *types.PublicKey) ([]types.SignedValidatorRegistration, error) {
	s.RLock()
	defer s.RUnlock()

	return s.registrations[*publicKey], nil
}
