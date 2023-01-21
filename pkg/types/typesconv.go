package types

import (
	"encoding/json"

	"github.com/flashbots/go-boost-utils/types"
	mev_boost_relay_types "github.com/flashbots/mev-boost-relay/database"
)

// Wrapper around `mev-boost-relay` converter util function of validator registration entry (DB) to a signed validator registration.
func ValidatorRegistrationEntryToSignedValidatorRegistration(entry *mev_boost_relay_types.ValidatorRegistrationEntry) (*types.SignedValidatorRegistration, error) {
	return entry.ToSignedValidatorRegistration()
}

func ValidatorRegistrationEntriesToSignedValidatorRegistrations(entries []*mev_boost_relay_types.ValidatorRegistrationEntry) (registrations []*types.SignedValidatorRegistration, err error) {
	// Go through all entries and try to convert each to SignedValidatorRegistration.
	for _, entry := range entries {
		registration, err := ValidatorRegistrationEntryToSignedValidatorRegistration(entry)
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, registration)
	}
	return registrations, nil
}

func AcceptanceWithContextToAcceptanceEntry(bidCtx *BidContext, acceptance *types.SignedBlindedBeaconBlock) (*AcceptanceEntry, error) {
	_acceptance, err := json.Marshal(acceptance)
	if err != nil {
		return nil, err
	}

	return &AcceptanceEntry{
		SignedBlindedBeaconBlock: mev_boost_relay_types.NewNullString(string(_acceptance)),

		// Bid "context" data.
		Slot:           bidCtx.Slot,
		ParentHash:     bidCtx.ParentHash.String(),
		RelayPubkey:    bidCtx.RelayPublicKey.String(),
		ProposerPubkey: bidCtx.ProposerPublicKey.String(),

		Signature: acceptance.Signature.String(),
	}, nil
}

func BidWithContextToBidEntry(bidCtx *BidContext, bid *Bid) (*BidEntry, error) {
	builderBid := bid.Message
	signature := bid.Signature

	_bid, err := json.Marshal(builderBid)
	if err != nil {
		return nil, err
	}

	return &BidEntry{
		// Bid "context" data.
		Slot:           bidCtx.Slot,
		ParentHash:     bidCtx.ParentHash.String(),
		RelayPubkey:    bidCtx.RelayPublicKey.String(),
		ProposerPubkey: bidCtx.ProposerPublicKey.String(),

		// Bidtrace data (public data).
		BlockHash:            builderBid.Header.BlockHash.String(),
		BuilderPubkey:        builderBid.Pubkey.String(),
		ProposerFeeRecipient: builderBid.Header.FeeRecipient.String(),

		GasUsed:  builderBid.Header.GasUsed,
		GasLimit: builderBid.Header.GasLimit,
		Value:    builderBid.Value.String(),

		Bid:         string(_bid),
		WasAccepted: false,

		Signature: signature.String(),
	}, nil
}

func BidEntryToBid(bidEntry *BidEntry) (*Bid, error) {
	builderBid := &types.BuilderBid{}

	// JSON parse the BuilderBid.
	err := json.Unmarshal([]byte(bidEntry.Bid), builderBid)
	if err != nil {
		return nil, err
	}

	// Parse out the signature.
	signature, err := types.HexToSignature(bidEntry.Signature)
	if err != nil {
		return nil, err
	}

	return &Bid{
		Message:   builderBid,
		Signature: signature,
	}, nil
}
