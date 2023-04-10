package store

import (
	"encoding/json"
	"fmt"

	"github.com/attestantio/go-builder-client/spec"
	boostTypes "github.com/flashbots/go-boost-utils/types"
	mev_boost_relay_types "github.com/flashbots/mev-boost-relay/database"
	"github.com/ralexstokes/relay-monitor/pkg/types"
)

// Wrapper around `mev-boost-relay` converter util function of validator registration entry (DB) to a signed validator registration.
func ValidatorRegistrationEntryToSignedValidatorRegistration(entry *mev_boost_relay_types.ValidatorRegistrationEntry) (*boostTypes.SignedValidatorRegistration, error) {
	return entry.ToSignedValidatorRegistration()
}

// ValidatorRegistrationEntryToSignedValidatorRegistration converts a list of validator registration entries to a list of signed validator registrations.
func ValidatorRegistrationEntriesToSignedValidatorRegistrations(entries []*mev_boost_relay_types.ValidatorRegistrationEntry) (registrations []*boostTypes.SignedValidatorRegistration, err error) {
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
func AcceptanceWithContextToAcceptanceEntry(bidCtx *types.BidContext, acceptance *types.VersionedAcceptance) (*AcceptanceEntry, error) {
	_acceptance, err := json.Marshal(acceptance)
	if err != nil {
		return nil, err
	}

	signature, err := acceptance.Signature()
	if err != nil {
		return nil, err
	}

	return &AcceptanceEntry{
		SignedBlindedBeaconBlock: mev_boost_relay_types.NewNullString(string(_acceptance)),

		// Bid "context" data.
		Slot:           uint64(bidCtx.Slot),
		ParentHash:     bidCtx.ParentHash.String(),
		RelayPubkey:    bidCtx.RelayPublicKey.String(),
		ProposerPubkey: bidCtx.ProposerPublicKey.String(),

		Signature: signature.String(),
	}, nil
}

// BidEntryToSignedBid converts a signed builder bid to a bid entry.
func BidWithContextToBidEntry(bidCtx *types.BidContext, bid *types.VersionedBid) (*BidEntry, error) {
	builderBid := bid.Bid

	_bid, err := json.Marshal(builderBid)
	if err != nil {
		return nil, err
	}

	blockHash, err := bid.BlockHash()
	if err != nil {
		return nil, err
	}
	builderPubkey, err := bid.Builder()
	if err != nil {
		return nil, err
	}
	proposerFeeRecipient, err := bid.FeeRecipient()
	if err != nil {
		return nil, err
	}
	gasUsed, err := bid.GasUsed()
	if err != nil {
		return nil, err
	}
	gasLimit, err := bid.GasLimit()
	if err != nil {
		return nil, err
	}
	value, err := bid.Value()
	if err != nil {
		return nil, err
	}
	signature, err := bid.Signature()
	if err != nil {
		return nil, err
	}

	return &BidEntry{
		// Bid "context" data.
		Slot:           uint64(bidCtx.Slot),
		ParentHash:     bidCtx.ParentHash.String(),
		RelayPubkey:    bidCtx.RelayPublicKey.String(),
		ProposerPubkey: bidCtx.ProposerPublicKey.String(),

		// Bidtrace data (public data).
		BlockHash:            blockHash.String(),
		BuilderPubkey:        builderPubkey.String(),
		ProposerFeeRecipient: proposerFeeRecipient.String(),

		GasUsed:  gasUsed,
		GasLimit: gasLimit,
		Value:    value.ToBig().String(),

		Bid:         string(_bid),
		WasAccepted: false,

		Signature: signature.String(),
	}, nil
}

// BidEntryToBid converts a bid entry to a signed builder bid.
func BidEntryToBid(bidEntry *BidEntry) (*types.VersionedBid, error) {
	builderBid := &spec.VersionedSignedBuilderBid{}

	// JSON parse the BuilderBid.
	err := json.Unmarshal([]byte(bidEntry.Bid), builderBid)
	if err != nil {
		return nil, err
	}

	return &types.VersionedBid{
		Bid: builderBid,
	}, nil
}

// InvalidBidToAnalysisEntry converts an invalid bid to an analysis entry.
func InvalidBidToAnalysisEntry(bidCtx *types.BidContext, invalidBid *types.InvalidBid) (*AnalysisEntry, error) {
	if bidCtx == nil {
		return nil, fmt.Errorf("no bid context for analysis entry")
	}

	// Pre-fill analysis entry with context.
	analysisEntry := &AnalysisEntry{
		// Bid "context" data.
		Slot:           uint64(bidCtx.Slot),
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
		analysisEntry.Category = types.ValidBidCategory
	}

	return analysisEntry, nil
}

// RelayToRelayEntry converts a relay struct to a relay entry.
func RelayToRelayEntry(relay *types.Relay) (*RelayEntry, error) {
	return &RelayEntry{
		Pubkey:   relay.Pubkey.String(),
		Hostname: relay.Hostname,
		Endpoint: relay.Endpoint,
	}, nil
}

// RelayEntryToRelay converts a relay entry to a relay struct.
func RelayEntryToRelay(relayEntry *RelayEntry) (*types.Relay, error) {
	pubkey, err := types.BLSPubKeyFromHexString(relayEntry.Pubkey)
	if err != nil {
		return nil, err
	}

	return &types.Relay{
		Pubkey:   pubkey,
		Hostname: relayEntry.Hostname,
		Endpoint: relayEntry.Endpoint,
	}, nil
}

// RelayEntriesToRelays converts a list of relay entries to a list of relay structs.
func RelayEntriesToRelays(relayEntries []*RelayEntry) (relays []*types.Relay, err error) {
	for _, entry := range relayEntries {
		relay, err := RelayEntryToRelay(entry)
		if err != nil {
			return nil, err
		}
		relays = append(relays, relay)
	}
	return relays, nil
}
