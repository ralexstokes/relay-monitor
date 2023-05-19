package store

import (
	"github.com/flashbots/mev-boost-relay/common"
)

var (
	// Prefix for all tables.
	tableBase = common.GetEnv("DB_TABLE_PREFIX", "dev")

	TableValidatorRegistration = tableBase + "_validator_registration"
	TableBids                  = tableBase + "_bids"
	TableAcceptances           = tableBase + "_acceptances"
	TableRelays                = tableBase + "_relays"

	TableBidsAnalysis = tableBase + "_bids_analysis"
)

var schema = `
CREATE TABLE IF NOT EXISTS ` + TableValidatorRegistration + ` (
	id          bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	inserted_at timestamp NOT NULL default current_timestamp,

	pubkey        varchar(98) NOT NULL,
	fee_recipient varchar(42) NOT NULL,
	timestamp     bigint NOT NULL,
	gas_limit     bigint NOT NULL,
	signature     text NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS ` + TableValidatorRegistration + `_pubkey_timestamp_uidx ON ` + TableValidatorRegistration + `(pubkey, timestamp DESC);

CREATE TABLE IF NOT EXISTS ` + TableBids + ` (
	id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	inserted_at timestamp NOT NULL default current_timestamp,

	-- bid "context" data
	slot                    bigint NOT NULL,
	parent_hash             varchar(66) NOT NULL,
	relay_pubkey            varchar(98) NOT NULL,
	proposer_pubkey         varchar(98) NOT NULL,

	-- bidtrace data (public data about a bid)
	block_hash              varchar(66) NOT NULL,
	builder_pubkey          varchar(98) NOT NULL,
	proposer_fee_recipient  varchar(42) NOT NULL,

	gas_limit               bigint NOT NULL,
	gas_used                bigint NOT NULL,
	value                   NUMERIC(48, 0),

	bid                     json NOT NULL,
	was_accepted            boolean NOT NULL,

	signature               text NOT NULL
);

CREATE INDEX IF NOT EXISTS ` + TableBids + `_slot_idx ON ` + TableBids + `("slot");
CREATE INDEX IF NOT EXISTS ` + TableBids + `_parenthash_idx ON ` + TableBids + `("parent_hash");
CREATE INDEX IF NOT EXISTS ` + TableBids + `_relaypubkey_idx ON ` + TableBids + `("relay_pubkey");
CREATE INDEX IF NOT EXISTS ` + TableBids + `_proposerpubkey_idx ON ` + TableBids + `("proposer_pubkey");

CREATE INDEX IF NOT EXISTS ` + TableBids + `_blockhash_idx ON ` + TableBids + `("block_hash");
CREATE INDEX IF NOT EXISTS ` + TableBids + `_insertedat_idx ON ` + TableBids + `("inserted_at");
CREATE INDEX IF NOT EXISTS ` + TableBids + `_builderpubkey_idx ON ` + TableBids + `("builder_pubkey");

CREATE TABLE IF NOT EXISTS ` + TableAcceptances + ` (
	id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	inserted_at timestamp NOT NULL default current_timestamp,

	signed_blinded_beacon_block json,

	-- bid acceptance "context" data
	slot                    bigint NOT NULL,
	parent_hash             varchar(66) NOT NULL,
	relay_pubkey            varchar(98) NOT NULL,
	proposer_pubkey         varchar(98) NOT NULL,

	signature               text NOT NULL,

	UNIQUE (slot, proposer_pubkey, parent_hash)
);

CREATE INDEX IF NOT EXISTS ` + TableAcceptances + `_slot_idx ON ` + TableAcceptances + `("slot");
CREATE INDEX IF NOT EXISTS ` + TableAcceptances + `_parenthash_idx ON ` + TableAcceptances + `("parent_hash");
CREATE INDEX IF NOT EXISTS ` + TableAcceptances + `_relaypubkey_idx ON ` + TableAcceptances + `("relay_pubkey");
CREATE INDEX IF NOT EXISTS ` + TableAcceptances + `_proposerpubkey_idx ON ` + TableAcceptances + `("proposer_pubkey");

CREATE INDEX IF NOT EXISTS ` + TableAcceptances + `_insertedat_idx ON ` + TableAcceptances + `("inserted_at");

CREATE TABLE IF NOT EXISTS ` + TableBidsAnalysis + ` (
	id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	inserted_at timestamp NOT NULL default current_timestamp,

	-- bid "context" data
	slot                    bigint NOT NULL,
	parent_hash             varchar(66) NOT NULL,
	relay_pubkey            varchar(98) NOT NULL,
	proposer_pubkey         varchar(98) NOT NULL,

	category                bigint,
	reason                  text
);

CREATE INDEX IF NOT EXISTS ` + TableBidsAnalysis + `_slot_idx ON ` + TableBidsAnalysis + `("slot");
CREATE INDEX IF NOT EXISTS ` + TableBidsAnalysis + `_parenthash_idx ON ` + TableBidsAnalysis + `("parent_hash");
CREATE INDEX IF NOT EXISTS ` + TableBidsAnalysis + `_relaypubkey_idx ON ` + TableBidsAnalysis + `("relay_pubkey");
CREATE INDEX IF NOT EXISTS ` + TableBidsAnalysis + `_proposerpubkey_idx ON ` + TableBidsAnalysis + `("proposer_pubkey");

CREATE INDEX IF NOT EXISTS ` + TableBidsAnalysis + `category_idx ON ` + TableBidsAnalysis + `("category");

CREATE TABLE IF NOT EXISTS ` + TableRelays + ` (
	id          bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
	inserted_at timestamp NOT NULL default current_timestamp,

	pubkey        varchar(98) NOT NULL,
	hostname      text NOT NULL,
	endpoint      text NOT NULL
);
`
