package store

import (
	"context"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ralexstokes/relay-monitor/pkg/types"
)

type MemoryStore struct {
	bids          map[types.BidContext]*types.VersionedBid
	registrations map[types.PublicKey][]*types.SignedValidatorRegistration
	acceptances   map[types.BidContext]types.VersionedAcceptance
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		bids:          make(map[types.BidContext]*types.VersionedBid),
		registrations: make(map[types.PublicKey][]*types.SignedValidatorRegistration),
		acceptances:   make(map[types.BidContext]types.VersionedAcceptance),
	}
}

func (s *MemoryStore) PutBid(ctx context.Context, bidCtx *types.BidContext, bid *types.VersionedBid) error {
	s.bids[*bidCtx] = bid
	return nil
}

func (s *MemoryStore) GetBid(ctx context.Context, bidCtx *types.BidContext) (*types.VersionedBid, error) {
	bid, ok := s.bids[*bidCtx]
	if !ok {
		return nil, fmt.Errorf("could not find bid for %+v", bidCtx)
	}
	return bid, nil
}

func (s *MemoryStore) PutValidatorRegistration(ctx context.Context, registration *types.SignedValidatorRegistration) error {
	publicKey := phase0.BLSPubKey(registration.Message.Pubkey)
	registrations := s.registrations[publicKey]
	registrations = append(registrations, registration)
	s.registrations[publicKey] = registrations
	return nil
}

func (s *MemoryStore) PutAcceptance(ctx context.Context, bidCtx *types.BidContext, acceptance *types.VersionedAcceptance) error {
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

func (s *MemoryStore) GetCountValidatorsRegistrations(ctx context.Context) (uint, error) {
	var totalRegistrations uint
	for _, signedRegistrations := range s.registrations {
		totalRegistrations += uint(len(signedRegistrations))
	}
	return totalRegistrations, nil
}

func (s *MemoryStore) GetCountValidators(ctx context.Context) (uint, error) {
	return uint(len(s.registrations)), nil
}
