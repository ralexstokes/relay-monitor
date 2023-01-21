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
