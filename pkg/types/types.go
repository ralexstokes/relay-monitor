package types

import (
	"fmt"
	"strconv"

	"github.com/attestantio/go-builder-client/spec"
	"github.com/attestantio/go-eth2-client/api"
	consensusspec "github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	boostTypes "github.com/flashbots/go-boost-utils/types"
	"github.com/holiman/uint256"
)

type (
	Slot                        = phase0.Slot
	Epoch                       = uint64
	Uint256                     = uint256.Int
	ValidatorIndex              = phase0.ValidatorIndex
	PublicKey                   = phase0.BLSPubKey
	Hash                        = phase0.Hash32
	Root                        = phase0.Root
	SignedValidatorRegistration = boostTypes.SignedValidatorRegistration
	SignedBlindedBeaconBlock    = api.VersionedSignedBlindedBeaconBlock
)

// SlotFromString converts a slot string in base 10 to a slot.
func SlotFromString(slotString string) (phase0.Slot, error) {
	startSlotValue, err := strconv.ParseUint(slotString, 10, 64)
	if err != nil {
		return 0, err
	}
	return phase0.Slot(startSlotValue), nil
}

// BLSPubKeyFromHexString converts a BLS public key hex string to a BLS public key.
func BLSPubKeyFromHexString(hexString string) (phase0.BLSPubKey, error) {
	var publicKey boostTypes.PublicKey
	err := publicKey.UnmarshalText([]byte(hexString))
	if err != nil {
		return phase0.BLSPubKey{}, err
	}
	return phase0.BLSPubKey(publicKey), nil
}

// VerionedAcceptance is a wrapper around VersionedSignedBlindedBeaconBlock that implements additional
// methods to make getting data easier.
type VersionedAcceptance struct {
	Block *api.VersionedSignedBlindedBeaconBlock
}

// Slot returns the slot of the signed beacon block.
func (a *VersionedAcceptance) Slot() (phase0.Slot, error) {
	if a == nil {
		return 0, fmt.Errorf("nil struct")
	}
	if a.Block == nil {
		return 0, fmt.Errorf("nil block")
	}
	return a.Block.Slot()
}

// ParentRoot returns the parent root of the signed beacon block.
func (a *VersionedAcceptance) ParentRoot() (phase0.Root, error) {
	if a == nil {
		return phase0.Root{}, fmt.Errorf("nil struct")
	}
	if a.Block == nil {
		return phase0.Root{}, fmt.Errorf("nil block")
	}
	return a.Block.ParentRoot()
}

// Signature returns the signature of the signed beacon block.
func (a *VersionedAcceptance) Signature() (phase0.BLSSignature, error) {
	if a == nil {
		return phase0.BLSSignature{}, fmt.Errorf("nil struct")
	}
	if a.Block == nil {
		return phase0.BLSSignature{}, fmt.Errorf("nil block")
	}
	switch a.Block.Version {
	case consensusspec.DataVersionBellatrix:
		if a.Block.Bellatrix == nil {
			return phase0.BLSSignature{}, fmt.Errorf("no data")
		}
		return a.Block.Bellatrix.Signature, nil
	case consensusspec.DataVersionCapella:
		if a.Block.Capella == nil {
			return phase0.BLSSignature{}, fmt.Errorf("no data")
		}
		return a.Block.Capella.Signature, nil
	default:
		return phase0.BLSSignature{}, fmt.Errorf("unsupported version")
	}
}

// ProposerIndex returns the proposer index of the signed beacon block.
func (a *VersionedAcceptance) ProposerIndex() (phase0.ValidatorIndex, error) {
	if a == nil {
		return 0, fmt.Errorf("nil struct")
	}
	if a.Block == nil {
		return 0, fmt.Errorf("nil block")
	}
	switch a.Block.Version {
	case consensusspec.DataVersionBellatrix:
		if a.Block.Bellatrix == nil {
			return 0, fmt.Errorf("no data")
		}
		if a.Block.Bellatrix.Message == nil {
			return 0, fmt.Errorf("no message")
		}
		return a.Block.Bellatrix.Message.ProposerIndex, nil
	case consensusspec.DataVersionCapella:
		if a.Block.Capella == nil {
			return 0, fmt.Errorf("no data")
		}
		if a.Block.Capella.Message == nil {
			return 0, fmt.Errorf("no message")
		}
		return a.Block.Capella.Message.ProposerIndex, nil
	default:
		return 0, fmt.Errorf("unsupported version")
	}
}

// Message returns the message of the signed beacon block.
func (a *VersionedAcceptance) Message() (boostTypes.HashTreeRoot, error) {
	if a == nil {
		return nil, fmt.Errorf("nil struct")
	}
	if a.Block == nil {
		return nil, fmt.Errorf("nil block")
	}
	switch a.Block.Version {
	case consensusspec.DataVersionBellatrix:
		if a.Block.Bellatrix == nil {
			return nil, fmt.Errorf("no data")
		}
		return a.Block.Bellatrix.Message, nil
	case consensusspec.DataVersionCapella:
		if a.Block.Capella == nil {
			return nil, fmt.Errorf("no data")
		}
		return a.Block.Capella.Message, nil
	default:
		return nil, fmt.Errorf("unsupported version")
	}
}

// VersionedBid is a wrapper around VersionedSignedBuilderBid that implements additional
// methods to make getting data easier.
type VersionedBid struct {
	Bid *spec.VersionedSignedBuilderBid
}

// Builder returns the BLS public key of the bid builder. Coming from the relay this is
// the BLS public key of the relay.
func (b *VersionedBid) Builder() (phase0.BLSPubKey, error) {
	if b == nil {
		return phase0.BLSPubKey{}, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return phase0.BLSPubKey{}, fmt.Errorf("nil bid")
	}
	return b.Bid.Builder()
}

// Signature returns the signature of the signed builder bid.
func (b *VersionedBid) Signature() (phase0.BLSSignature, error) {
	if b == nil {
		return phase0.BLSSignature{}, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return phase0.BLSSignature{}, fmt.Errorf("nil bid")
	}
	return b.Bid.Signature()
}

// ParentHash returns the parent hash of the signed builder bid.
func (b *VersionedBid) ParentHash() (phase0.Hash32, error) {
	if b == nil {
		return phase0.Hash32{}, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return phase0.Hash32{}, fmt.Errorf("nil bid")
	}
	return b.Bid.ParentHash()
}

// FeeRecipient returns the fee recipient of the signed builder bid.
func (b *VersionedBid) FeeRecipient() (bellatrix.ExecutionAddress, error) {
	if b == nil {
		return bellatrix.ExecutionAddress{}, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return bellatrix.ExecutionAddress{}, fmt.Errorf("nil bid")
	}
	return b.Bid.FeeRecipient()
}

// Value returns the value of the signed builder bid.
func (b *VersionedBid) Value() (*uint256.Int, error) {
	if b == nil {
		return nil, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return nil, fmt.Errorf("nil bid")
	}
	return b.Bid.Value()
}

// Timestamp returns the timestamp of the signed builder bid.
func (b *VersionedBid) Timestamp() (uint64, error) {
	if b == nil {
		return 0, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return 0, fmt.Errorf("nil bid")
	}
	return b.Bid.Timestamp()
}

// Message returns the message of the signed builder bid.
func (b *VersionedBid) Message() (boostTypes.HashTreeRoot, error) {
	if b == nil {
		return nil, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return nil, fmt.Errorf("nil bid")
	}
	switch b.Bid.Version {
	case consensusspec.DataVersionBellatrix:
		if b.Bid.Bellatrix == nil {
			return nil, fmt.Errorf("no data")
		}
		if b.Bid.Bellatrix.Message == nil {
			return nil, fmt.Errorf("no message")
		}
		return b.Bid.Bellatrix.Message, nil
	case consensusspec.DataVersionCapella:
		if b.Bid.Capella == nil {
			return nil, fmt.Errorf("no data")
		}
		if b.Bid.Capella.Message == nil {
			return nil, fmt.Errorf("no message")
		}
		return b.Bid.Capella.Message, nil
	default:
		return nil, fmt.Errorf("unsupported version")
	}
}

// PrevRandao returns the RANDAO of the signed builder bid.
func (b *VersionedBid) PrevRandao() (phase0.Hash32, error) {
	if b == nil {
		return phase0.Hash32{}, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return phase0.Hash32{}, fmt.Errorf("nil bid")
	}
	switch b.Bid.Version {
	case consensusspec.DataVersionBellatrix:
		if b.Bid.Bellatrix == nil {
			return phase0.Hash32{}, fmt.Errorf("no data")
		}
		if b.Bid.Bellatrix.Message == nil {
			return phase0.Hash32{}, fmt.Errorf("no message")
		}
		if b.Bid.Bellatrix.Message.Header == nil {
			return phase0.Hash32{}, fmt.Errorf("no header")
		}
		return b.Bid.Bellatrix.Message.Header.PrevRandao, nil
	case consensusspec.DataVersionCapella:
		if b.Bid.Capella == nil {
			return phase0.Hash32{}, fmt.Errorf("no data")
		}
		if b.Bid.Capella.Message == nil {
			return phase0.Hash32{}, fmt.Errorf("no message")
		}
		if b.Bid.Capella.Message.Header == nil {
			return phase0.Hash32{}, fmt.Errorf("no header")
		}
		return b.Bid.Capella.Message.Header.PrevRandao, nil
	default:
		return phase0.Hash32{}, fmt.Errorf("unsupported version")
	}
}

// BlockNumber returns the block number of the signed builder bid.
func (b *VersionedBid) BlockNumber() (uint64, error) {
	if b == nil {
		return 0, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return 0, fmt.Errorf("nil bid")
	}
	switch b.Bid.Version {
	case consensusspec.DataVersionBellatrix:
		if b.Bid.Bellatrix == nil {
			return 0, fmt.Errorf("no data")
		}
		if b.Bid.Bellatrix.Message == nil {
			return 0, fmt.Errorf("no message")
		}
		if b.Bid.Bellatrix.Message.Header == nil {
			return 0, fmt.Errorf("no header")
		}
		return b.Bid.Bellatrix.Message.Header.BlockNumber, nil
	case consensusspec.DataVersionCapella:
		if b.Bid.Capella == nil {
			return 0, fmt.Errorf("no data")
		}
		if b.Bid.Capella.Message == nil {
			return 0, fmt.Errorf("no message")
		}
		if b.Bid.Capella.Message.Header == nil {
			return 0, fmt.Errorf("no header")
		}
		return b.Bid.Capella.Message.Header.BlockNumber, nil
	default:
		return 0, fmt.Errorf("unsupported version")
	}
}

// BlockHash returns the block hash of the signed builder bid.
func (b *VersionedBid) BlockHash() (phase0.Hash32, error) {
	if b == nil {
		return phase0.Hash32{}, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return phase0.Hash32{}, fmt.Errorf("nil bid")
	}
	switch b.Bid.Version {
	case consensusspec.DataVersionBellatrix:
		if b.Bid.Bellatrix == nil {
			return phase0.Hash32{}, fmt.Errorf("no data")
		}
		if b.Bid.Bellatrix.Message == nil {
			return phase0.Hash32{}, fmt.Errorf("no message")
		}
		if b.Bid.Bellatrix.Message.Header == nil {
			return phase0.Hash32{}, fmt.Errorf("no header")
		}
		return b.Bid.Bellatrix.Message.Header.BlockHash, nil
	case consensusspec.DataVersionCapella:
		if b.Bid.Capella == nil {
			return phase0.Hash32{}, fmt.Errorf("no data")
		}
		if b.Bid.Capella.Message == nil {
			return phase0.Hash32{}, fmt.Errorf("no message")
		}
		if b.Bid.Capella.Message.Header == nil {
			return phase0.Hash32{}, fmt.Errorf("no header")
		}
		return b.Bid.Capella.Message.Header.BlockHash, nil
	default:
		return phase0.Hash32{}, fmt.Errorf("unsupported version")
	}
}

// GasUsed returns the gas used of the signed builder bid.
func (b *VersionedBid) GasUsed() (uint64, error) {
	if b == nil {
		return 0, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return 0, fmt.Errorf("nil bid")
	}
	switch b.Bid.Version {
	case consensusspec.DataVersionBellatrix:
		if b.Bid.Bellatrix == nil {
			return 0, fmt.Errorf("no data")
		}
		if b.Bid.Bellatrix.Message == nil {
			return 0, fmt.Errorf("no message")
		}
		if b.Bid.Bellatrix.Message.Header == nil {
			return 0, fmt.Errorf("no header")
		}
		return b.Bid.Bellatrix.Message.Header.GasUsed, nil
	case consensusspec.DataVersionCapella:
		if b.Bid.Capella == nil {
			return 0, fmt.Errorf("no data")
		}
		if b.Bid.Capella.Message == nil {
			return 0, fmt.Errorf("no message")
		}
		if b.Bid.Capella.Message.Header == nil {
			return 0, fmt.Errorf("no header")
		}
		return b.Bid.Capella.Message.Header.GasUsed, nil
	default:
		return 0, fmt.Errorf("unsupported version")
	}
}

// GasLimit returns the gas limit of the signed builder bid.
func (b *VersionedBid) GasLimit() (uint64, error) {
	if b == nil {
		return 0, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return 0, fmt.Errorf("nil bid")
	}
	switch b.Bid.Version {
	case consensusspec.DataVersionBellatrix:
		if b.Bid.Bellatrix == nil {
			return 0, fmt.Errorf("no data")
		}
		if b.Bid.Bellatrix.Message == nil {
			return 0, fmt.Errorf("no message")
		}
		if b.Bid.Bellatrix.Message.Header == nil {
			return 0, fmt.Errorf("no header")
		}
		return b.Bid.Bellatrix.Message.Header.GasLimit, nil
	case consensusspec.DataVersionCapella:
		if b.Bid.Capella == nil {
			return 0, fmt.Errorf("no data")
		}
		if b.Bid.Capella.Message == nil {
			return 0, fmt.Errorf("no message")
		}
		if b.Bid.Capella.Message.Header == nil {
			return 0, fmt.Errorf("no header")
		}
		return b.Bid.Capella.Message.Header.GasLimit, nil
	default:
		return 0, fmt.Errorf("unsupported version")
	}
}

// BaseFeeForGas returns the base fee for gas of the signed builder bid.
func (b *VersionedBid) BaseFeeForGas() ([32]byte, error) {
	if b == nil {
		return [32]byte{}, fmt.Errorf("nil struct")
	}
	if b.Bid == nil {
		return [32]byte{}, fmt.Errorf("nil bid")
	}
	switch b.Bid.Version {
	case consensusspec.DataVersionBellatrix:
		if b.Bid.Bellatrix == nil {
			return [32]byte{}, fmt.Errorf("no data")
		}
		if b.Bid.Bellatrix.Message == nil {
			return [32]byte{}, fmt.Errorf("no message")
		}
		if b.Bid.Bellatrix.Message.Header == nil {
			return [32]byte{}, fmt.Errorf("no header")
		}
		return b.Bid.Bellatrix.Message.Header.BaseFeePerGas, nil
	case consensusspec.DataVersionCapella:
		if b.Bid.Capella == nil {
			return [32]byte{}, fmt.Errorf("no data")
		}
		if b.Bid.Capella.Message == nil {
			return [32]byte{}, fmt.Errorf("no message")
		}
		if b.Bid.Capella.Message.Header == nil {
			return [32]byte{}, fmt.Errorf("no header")
		}
		return b.Bid.Capella.Message.Header.BaseFeePerGas, nil
	default:
		return [32]byte{}, fmt.Errorf("unsupported version")
	}
}

type Coordinate struct {
	Slot Slot
	Root phase0.Root
}

type InvalidBid struct {
	Category AnalysisCategory
	Reason   AnalysisReason
	Context  map[string]interface{}
}

type AuctionTranscript struct {
	Bid        VersionedBid        `json:"bid"`
	Acceptance VersionedAcceptance `json:"acceptance"`
}

type BidContext struct {
	Slot              Slot             `json:"slot"`
	ParentHash        phase0.Hash32    `json:"parent_hash"`
	ProposerPublicKey phase0.BLSPubKey `json:"proposer_public_key"`
	RelayPublicKey    phase0.BLSPubKey `json:"relay_public_key"`
}

type Relay struct {
	Pubkey   phase0.BLSPubKey
	Hostname string
	Endpoint string
}

type (
	ScoreReport        = map[string]*Score
	FaultStatsReport   = map[string]*FaultStats
	FaultRecordsReport = map[string]*FaultRecords
)

type Score struct {
	Score float64 `json:"score"`
	Meta  *Meta   `json:"meta"`
}

type FaultStats struct {
	Stats *Stats `json:"stats"`
	Meta  *Meta  `json:"meta"`
}

type FaultRecords struct {
	Records *Records `json:"records"`
	Meta    *Meta    `json:"meta"`
}

type Record struct {
	Slot           uint64 `db:"slot"`
	ParentHash     string `db:"parent_hash"`
	ProposerPubkey string `db:"proposer_pubkey"`
}

type Records struct {
	ConsensusInvalidBids   []*Record `json:"consensus_invalid_bids"`
	IgnoredPreferencesBids []*Record `json:"ignored_preferences_bids"`

	PaymentInvalidBids       []*Record `json:"payment_invalid_bids"`
	MalformedPayloads        []*Record `json:"malformed_payloads"`
	ConsensusInvalidPayloads []*Record `json:"consensus_invalid_payloads"`
	UnavailablePayloads      []*Record `json:"unavailable_payloads"`
}

type Stats struct {
	TotalBids uint64 `json:"total_bids"`

	ConsensusInvalidBids   uint64 `json:"consensus_invalid_bids"`
	IgnoredPreferencesBids uint64 `json:"ignored_preferences_bids"`

	PaymentInvalidBids       uint `json:"payment_invalid_bids"`
	MalformedPayloads        uint `json:"malformed_payloads"`
	ConsensusInvalidPayloads uint `json:"consensus_invalid_payloads"`
	UnavailablePayloads      uint `json:"unavailable_payloads"`
}

type Meta struct {
	Hostname string `json:"hostname"`
}

type SlotBounds struct {
	StartSlot *Slot `json:"start_slot"`
	EndSlot   *Slot `json:"end_slot"`
}

type AnalysisCategory uint

const (
	ValidBidCategory AnalysisCategory = iota
	InvalidBidPublicKeyCategory
	InvalidBidSignatureCategory
	InvalidBidConsensusCategory
	InvalidBidIgnoredPreferencesCategory
)

type AnalysisReason string

const (
	AnalysisReasonEmpty                                  AnalysisReason = ""
	AnalysisReasonIncorrectPublicKey                     AnalysisReason = "incorrect public key from relay"
	AnalysisReasonInvalidSignature                       AnalysisReason = "invalid signature"
	AnalysisReasonInvalidParentHash                      AnalysisReason = "invalid parent hash"
	AnalysisReasonInvalidRandomValue                     AnalysisReason = "invalid random value"
	AnalysisReasonInvalidBlockNumber                     AnalysisReason = "invalid block number"
	AnalysisReasonInvalidTimestamp                       AnalysisReason = "invalid timestamp"
	AnalysisReasonInvalidBaseFee                         AnalysisReason = "invalid base fee"
	AnalysisReasonInvalidGasUsed                         AnalysisReason = "gas used is higher than gas limit"
	AnalysisReasonIgnoredValidatorPreferenceGasLimit     AnalysisReason = "ignored validator gas limit preference"
	AnalysisReasonIgnoredValidatorPreferenceFeeRecipient AnalysisReason = "ignored validator fee recipient preference"
)

type AnalysisQueryFilter struct {
	Category   AnalysisCategory
	Comparator string
}
