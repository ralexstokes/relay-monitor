package store

import (
	"context"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

func GetLatestValidatorRegistration(ctx context.Context, store Storer, publicKey *types.PublicKey) (*types.SignedValidatorRegistration, error) {
	registrations, err := store.GetValidatorRegistrations(ctx, publicKey)
	if err != nil {
		return nil, err
	}

	if len(registrations) == 0 {
		return nil, nil
	} else {
		currentRegistration := &registrations[len(registrations)-1]
		return currentRegistration, nil
	}
}
