package types

import (
	"encoding/json"
	"fmt"

	"github.com/flashbots/go-boost-utils/types"
	mev_boost_relay_types "github.com/flashbots/mev-boost-relay/database"
)

// Wrapper around `mev-boost-relay` converter util function of validator registration entry (DB) to a signed validator registration.
func ValidatorRegistrationEntryToSignedValidatorRegistration(entry *mev_boost_relay_types.ValidatorRegistrationEntry) (*types.SignedValidatorRegistration, error) {
	return entry.ToSignedValidatorRegistration()
}

// ValidatorRegistrationEntryToSignedValidatorRegistration converts a list of validator registration entries to a list of signed validator registrations.
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

// AcceptanceEntryToSignedBlindedBeaconBlock converts a signed blinded beacon block to an acceptance entry.
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

// BidEntryToSignedBid converts a signed builder bid to a bid entry.
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

// BidEntryToBid converts a bid entry to a signed builder bid.
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

// InvalidBidToAnalysisEntry converts an invalid bid to an analysis entry.
func InvalidBidToAnalysisEntry(bidCtx *BidContext, invalidBid *InvalidBid) (*AnalysisEntry, error) {
	if bidCtx == nil {
		return nil, fmt.Errorf("no bid context for analysis entry")
	}

	// Pre-fill analysis entry with context.
	analysisEntry := &AnalysisEntry{
		// Bid "context" data.
		Slot:           bidCtx.Slot,
		ParentHash:     bidCtx.ParentHash.String(),
		RelayPubkey:    bidCtx.RelayPublicKey.String(),
		ProposerPubkey: bidCtx.ProposerPublicKey.String(),
	}

	// If the 'invalidBid' is defined, then set the category and the reason, otherwise the bid
	// must be valid so set the analysis category to reflect a valid bid status.
	if invalidBid != nil {
		analysisEntry.Category = invalidBid.Category
		analysisEntry.Reason = string(invalidBid.Reason)
	} else {
		analysisEntry.Category = ValidBidCategory
	}

	return analysisEntry, nil
}

// RelayToRelayEntry converts a relay struct to a relay entry.
func RelayToRelayEntry(relay *Relay) (*RelayEntry, error) {
	return &RelayEntry{
		Pubkey:   relay.Pubkey.String(),
		Hostname: relay.Hostname,
		Endpoint: relay.Endpoint,
	}, nil
}

// RelayEntryToRelay converts a relay entry to a relay struct.
func RelayEntryToRelay(relayEntry *RelayEntry) (*Relay, error) {
	pubkey, err := types.HexToPubkey(relayEntry.Pubkey)
	if err != nil {
		return nil, err
	}

	return &Relay{
		Pubkey:   pubkey,
		Hostname: relayEntry.Hostname,
		Endpoint: relayEntry.Endpoint,
	}, nil
}

// RelayEntriesToRelays converts a list of relay entries to a list of relay structs.
func RelayEntriesToRelays(relayEntries []*RelayEntry) (relays []*Relay, err error) {
	for _, entry := range relayEntries {
		relay, err := RelayEntryToRelay(entry)
		if err != nil {
			return nil, err
		}
		relays = append(relays, relay)
	}
	return relays, nil
}
