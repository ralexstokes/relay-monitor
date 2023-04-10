package store

import (
	"database/sql"
	"time"

	"github.com/ralexstokes/relay-monitor/pkg/types"
)

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

type AnalysisEntry struct {
	ID         int64     `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	// Bid analysis "context" data
	Slot           uint64 `db:"slot"`
	ParentHash     string `db:"parent_hash"`
	RelayPubkey    string `db:"relay_pubkey"`
	ProposerPubkey string `db:"proposer_pubkey"`

	Category types.AnalysisCategory `db:"category"`
	Reason   string                 `db:"reason"`
}

type RelayEntry struct {
	ID         int64     `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	Pubkey   string `db:"pubkey"`
	Hostname string `db:"hostname"`
	Endpoint string `db:"endpoint"`
}
