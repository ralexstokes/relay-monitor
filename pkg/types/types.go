package types

import (
	"database/sql"
	"time"

	"github.com/flashbots/go-boost-utils/types"
	"github.com/holiman/uint256"
)

type (
	Slot                        = uint64
	Epoch                       = uint64
	ForkVersion                 = types.ForkVersion
	Uint256                     = uint256.Int
	PublicKey                   = types.PublicKey
	Hash                        = types.Hash
	Bid                         = types.SignedBuilderBid
	Root                        = types.Root
	ValidatorIndex              = uint64
	SignedValidatorRegistration = types.SignedValidatorRegistration
	SignedBlindedBeaconBlock    = types.SignedBlindedBeaconBlock
)

type Coordinate struct {
	Slot Slot
	Root Root
}

type AuctionTranscript struct {
	Bid        Bid                            `json:"bid"`
	Acceptance types.SignedBlindedBeaconBlock `json:"acceptance"`
}

type BidContext struct {
	Slot              Slot      `json:"slot"`
	ParentHash        Hash      `json:"parent_hash"`
	ProposerPublicKey PublicKey `json:"proposer_public_key"`
	RelayPublicKey    PublicKey `json:"relay_public_key"`
}

// Database types

type BidEntry struct {
	ID         int64     `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	// Bid "context" data
	Slot           uint64 `db:"slot"`
	ParentHash     string `db:"parent_hash"`
	RelayPubkey    string `db:"relay_pubkey"`
	ProposerPubkey string `db:"proposer_pubkey"`

	// Bidtrace data (public data about a bid)
	BlockHash            string `db:"block_hash"`
	BuilderPubkey        string `db:"builder_pubkey"`
	ProposerFeeRecipient string `db:"proposer_fee_recipient"`

	GasUsed  uint64 `db:"gas_used"`
	GasLimit uint64 `db:"gas_limit"`
	Value    string `db:"value"`

	Bid         string `db:"bid"`
	WasAccepted bool   `db:"was_accepted"`

	Signature string `db:"signature"`
}

type AcceptanceEntry struct {
	ID         int64     `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	SignedBlindedBeaconBlock sql.NullString `db:"signed_blinded_beacon_block"`

	// Bid acceptance "context" data
	Slot           uint64 `db:"slot"`
	ParentHash     string `db:"parent_hash"`
	RelayPubkey    string `db:"relay_pubkey"`
	ProposerPubkey string `db:"proposer_pubkey"`

	Signature string `db:"signature"`
}

type InvalidBid struct {
	Category AnalysisCategory
	Reason   AnalysisReason
	Context  map[string]interface{}
}

type AnalysisEntry struct {
	ID         int64     `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	// Bid analysis "context" data
	Slot           uint64 `db:"slot"`
	ParentHash     string `db:"parent_hash"`
	RelayPubkey    string `db:"relay_pubkey"`
	ProposerPubkey string `db:"proposer_pubkey"`

	Category AnalysisCategory `db:"category"`
	Reason   string           `db:"reason"`
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

type RelayEntry struct {
	ID         int64     `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	Pubkey   string `db:"pubkey"`
	Hostname string `db:"hostname"`
	Endpoint string `db:"endpoint"`
}

type Relay struct {
	Pubkey   types.PublicKey
	Hostname string
	Endpoint string
}

type (
	ScoreReport        = map[types.PublicKey]*Score
	FaultStatsReport   = map[types.PublicKey]*FaultStats
	FaultRecordsReport = map[types.PublicKey]*FaultRecords
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
	StartSlot *uint64 `json:"start_slot"`
	EndSlot   *uint64 `json:"end_slot"`
}
