package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	mev_boost_relay_types "github.com/flashbots/mev-boost-relay/database"
	"github.com/flashbots/mev-boost-relay/database/vars"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/ralexstokes/relay-monitor/pkg/types"
	"go.uber.org/zap"
)

type PostgresStore struct {
	DB *sqlx.DB

	nstmtInsertBid        *sqlx.NamedStmt
	nstmtInsertAcceptance *sqlx.NamedStmt
	nstmtInsertAnalysis   *sqlx.NamedStmt

	logger *zap.SugaredLogger
}

func NewPostgresStore(dsn string, zapLogger *zap.Logger) (*PostgresStore, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.DB.SetMaxOpenConns(50)
	db.DB.SetMaxIdleConns(10)
	db.DB.SetConnMaxIdleTime(0)

	if os.Getenv("DB_DONT_APPLY_SCHEMA") == "" {
		_, err = db.Exec(schema)
		if err != nil {
			return nil, err
		}
	}

	store := &PostgresStore{DB: db, logger: zapLogger.Sugar()} //nolint:exhaustruct
	err = store.prepareNamedQueries()
	return store, err
}

func (store *PostgresStore) prepareNamedQueries() (err error) {
	// Insert bid.
	query := `INSERT INTO ` + TableBids + `
	(slot, parent_hash, relay_pubkey, proposer_pubkey, block_hash, builder_pubkey, proposer_fee_recipient, gas_used, gas_limit, value, bid, was_accepted, signature) VALUES
	(:slot, :parent_hash, :relay_pubkey, :proposer_pubkey, :block_hash, :builder_pubkey, :proposer_fee_recipient, :gas_used, :gas_limit, :value, :bid, :was_accepted, :signature) 
    RETURNING id`
	store.nstmtInsertBid, err = store.DB.PrepareNamed(query)
	if err != nil {
		return err
	}

	// Insert acceptance (of the bid).
	query = `INSERT INTO ` + TableAcceptances + `
	(signed_blinded_beacon_block, slot, parent_hash, relay_pubkey, proposer_pubkey, signature) VALUES
	(:signed_blinded_beacon_block, :slot, :parent_hash, :relay_pubkey, :proposer_pubkey, :signature) 
	RETURNING id`
	store.nstmtInsertAcceptance, err = store.DB.PrepareNamed(query)
	if err != nil {
		return err
	}

	// Insert analysis (of the bid).
	query = `INSERT INTO ` + TableBidsAnalysis + `
	(slot, parent_hash, relay_pubkey, proposer_pubkey, category, reason) VALUES
	(:slot, :parent_hash, :relay_pubkey, :proposer_pubkey, :category, :reason) 
	RETURNING id`
	store.nstmtInsertAnalysis, err = store.DB.PrepareNamed(query)

	return err
}

func (store *PostgresStore) Close() error {
	return store.DB.Close()
}

func (store *PostgresStore) PutBid(ctx context.Context, bidCtx *types.BidContext, bid *types.Bid) error {
	// Convert into a format that works better with the DB.
	bidEntry, err := types.BidWithContextToBidEntry(bidCtx, bid)
	if err != nil {
		return err
	}

	// Insert into DB.
	err = store.nstmtInsertBid.QueryRow(bidEntry).Scan(&bidEntry.ID)
	if err != nil {
		return err
	}
	store.logger.Info("saved bid to db", zap.Uint64("slot", bidCtx.Slot), zap.String("parent_hash", bidCtx.ParentHash.String()))

	return nil
}

func (store *PostgresStore) GetBid(ctx context.Context, bidCtx *types.BidContext) (*types.Bid, error) {
	// Fetch bid based on the "bid context".
	query := `SELECT bid, signature
	FROM ` + TableBids + `
	WHERE slot=$1 AND parent_hash=$2 AND relay_pubkey=$3 AND proposer_pubkey=$4`

	bidEntry := &types.BidEntry{}
	err := store.DB.Get(bidEntry, query, bidCtx.Slot, bidCtx.ParentHash.String(), bidCtx.RelayPublicKey.String(), bidCtx.ProposerPublicKey.String())
	if err != nil {
		return nil, err
	}
	store.logger.Info("fetched bid from db", zap.Uint64("slot", bidCtx.Slot), zap.String("parent_hash", bidCtx.ParentHash.String()))

	return types.BidEntryToBid(bidEntry)
}

func (store *PostgresStore) PutAcceptance(ctx context.Context, bidCtx *types.BidContext, acceptance *types.SignedBlindedBeaconBlock) error {
	// Convert into a format that works better with the DB.
	acceptanceEntry, err := types.AcceptanceWithContextToAcceptanceEntry(bidCtx, acceptance)
	if err != nil {
		return err
	}

	// Insert into DB.
	err = store.nstmtInsertAcceptance.QueryRow(acceptanceEntry).Scan(&acceptanceEntry.ID)
	if err != nil {
		return err
	}
	store.logger.Info("saved acceptance to db", zap.Uint64("slot", bidCtx.Slot), zap.String("parent_hash", bidCtx.ParentHash.String()))

	return nil
}

func (store *PostgresStore) PutValidatorRegistration(ctx context.Context, registration *types.SignedValidatorRegistration) error {
	validatorRegistrationEntry := mev_boost_relay_types.SignedValidatorRegistrationToEntry(*registration)

	// Insert into DB. Use the same query as `mev-boost-relay` that contains a check
	// for the timestamp and gas limit / fee recipient validator preferences to avoid
	// unnecessary inserts.
	//
	// https://github.com/flashbots/mev-boost-relay/blob/main/database/database.go#L119
	query := `WITH latest_registration AS (
		SELECT DISTINCT ON (pubkey) pubkey, fee_recipient, timestamp, gas_limit, signature FROM ` + vars.TableValidatorRegistration + ` WHERE pubkey=:pubkey ORDER BY pubkey, timestamp DESC limit 1
	)
	INSERT INTO ` + vars.TableValidatorRegistration + ` (pubkey, fee_recipient, timestamp, gas_limit, signature)
	SELECT :pubkey, :fee_recipient, :timestamp, :gas_limit, :signature
	WHERE NOT EXISTS (
		SELECT 1 from latest_registration WHERE pubkey=:pubkey AND :timestamp <= latest_registration.timestamp OR (:fee_recipient = latest_registration.fee_recipient AND :gas_limit = latest_registration.gas_limit)
	);`
	_, err := store.DB.NamedExec(query, validatorRegistrationEntry)
	if err != nil {
		return err
	}
	store.logger.Info("saved validator registration to db", zap.String("pubkey", validatorRegistrationEntry.Pubkey))

	return nil
}

func (store *PostgresStore) GetValidatorRegistrations(ctx context.Context, publicKey *types.PublicKey) ([]*types.SignedValidatorRegistration, error) {
	// Fetch all validator registrations for a given 'publicKey'.
	query := `SELECT pubkey, fee_recipient, timestamp, gas_limit, signature
	FROM ` + vars.TableValidatorRegistration + `
	WHERE pubkey=$1
	ORDER BY pubkey, timestamp DESC;`

	var entries []*mev_boost_relay_types.ValidatorRegistrationEntry
	err := store.DB.Select(&entries, query, publicKey.String())
	if err != nil {
		return nil, err
	}
	store.logger.Info("fetched validator registrations from db", zap.String("pubkey", publicKey.String()))

	return types.ValidatorRegistrationEntriesToSignedValidatorRegistrations(entries)
}

func (store *PostgresStore) GetLatestValidatorRegistration(ctx context.Context, publicKey *types.PublicKey) (*types.SignedValidatorRegistration, error) {
	// Fetch the latest registration for a given 'publicKey'.
	query := `SELECT DISTINCT ON (pubkey) pubkey, fee_recipient, timestamp, gas_limit, signature
	FROM ` + vars.TableValidatorRegistration + `
	WHERE pubkey=$1
	ORDER BY pubkey, timestamp DESC;`

	entry := &mev_boost_relay_types.ValidatorRegistrationEntry{}
	err := store.DB.Get(entry, query, publicKey.String())

	if errors.Cause(err) == sql.ErrNoRows {
		store.logger.Info("no validator registrations yet for this pubkey", zap.String("pubkey", publicKey.String()))
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	store.logger.Info("fetched latest validator registration from db", zap.String("pubkey", publicKey.String()))

	return types.ValidatorRegistrationEntryToSignedValidatorRegistration(entry)
}

func (store *PostgresStore) GetCountValidatorsRegistrations(ctx context.Context) (count uint, err error) {
	query := `SELECT COUNT(*) FROM ` + TableValidatorRegistration + `;`
	row := store.DB.QueryRow(query)
	err = row.Scan(&count)
	return count, err
}

func (store *PostgresStore) GetCountValidators(ctx context.Context) (count uint, err error) {
	query := `SELECT COUNT(*) FROM (SELECT DISTINCT pubkey FROM ` + TableValidatorRegistration + `) AS temp;`
	row := store.DB.QueryRow(query)
	err = row.Scan(&count)
	return count, err
}

func (store *PostgresStore) PutBidAnalysis(ctx context.Context, bidCtx *types.BidContext, invalidBid *types.InvalidBid) error {
	// Convert into a format that works better with the DB.
	analysisEntry, err := types.InvalidBidToAnalysisEntry(bidCtx, invalidBid)
	if err != nil {
		return err
	}

	// Insert into DB.
	err = store.nstmtInsertAnalysis.QueryRow(analysisEntry).Scan(&analysisEntry.ID)
	if err != nil {
		return err
	}
	store.logger.Info("saved analysis to db", zap.Uint64("slot", bidCtx.Slot), zap.String("parent_hash", bidCtx.ParentHash.String()))

	return nil
}

func (store *PostgresStore) GetCountAnalysisLookbackSlots(ctx context.Context, lookbackSlots uint64, filter *types.AnalysisQueryFilter) (count uint64, err error) {
	query := `SELECT COUNT(*) FROM ` + TableBidsAnalysis + `
	WHERE slot >= (SELECT MAX(slot) - ` + strconv.FormatUint(lookbackSlots, 10) + ` FROM ` + TableBidsAnalysis + `)`

	// Add an optional category filter.
	query = BuildCategoryFilterClause(query, filter)

	row := store.DB.QueryRow(query)
	err = row.Scan(&count)

	store.logger.Infow("query executed: count analysis within slots", "query", query, "count", count)

	return count, err
}

func (store *PostgresStore) GetCountAnalysisLookbackDuration(ctx context.Context, lookbackDuration time.Duration, filter *types.AnalysisQueryFilter) (count uint64, err error) {
	query := `SELECT COUNT(*) FROM ` + TableBidsAnalysis + `
	WHERE inserted_at >= NOW() - INTERVAL '` + fmt.Sprintf("%.0f minutes", lookbackDuration.Minutes()) + `'`

	// Add an optional category filter.
	query = BuildCategoryFilterClause(query, filter)

	row := store.DB.QueryRow(query)
	err = row.Scan(&count)

	store.logger.Infow("query executed: count analysis within duration", "query", query, "count", count)

	return count, err
}

func (store *PostgresStore) GetCountAnalysisWithinSlotBounds(ctx context.Context, relayPubkey string, slotBounds *types.SlotBounds, filter *types.AnalysisQueryFilter) (count uint64, err error) {
	query := `SELECT COUNT(*) FROM ` + TableBidsAnalysis + `
	WHERE relay_pubkey = '` + relayPubkey + `'`

	// Add a bounds filter.
	query = BuildSlotBoundsFilterClause(query, slotBounds)

	// Add an optional category filter.
	query = BuildCategoryFilterClause(query, filter)

	row := store.DB.QueryRow(query)
	err = row.Scan(&count)

	store.logger.Infow("query executed: count analysis within slot bounds", "query", query, "count", count)

	return count, err
}

func (store *PostgresStore) PutRelay(ctx context.Context, relay *types.Relay) error {
	// Convert into a format that works better with the DB.
	entry, err := types.RelayToRelayEntry(relay)
	if err != nil {
		return err
	}

	// Only insert if the relay does not already exist.
	query := `WITH stored_relay AS (
		SELECT DISTINCT ON (pubkey) pubkey FROM ` + TableRelays + ` WHERE pubkey=:pubkey
	)
	INSERT INTO ` + TableRelays + ` (pubkey, hostname, endpoint)
	SELECT :pubkey, :hostname, :endpoint
	WHERE NOT EXISTS (
		SELECT 1 from stored_relay WHERE pubkey=:pubkey
	);`

	// Insert into DB.
	_, err = store.DB.NamedExec(query, entry)
	if err != nil {
		return err
	}
	store.logger.Info("saved relay record to db", zap.String("pubkey", entry.Pubkey), zap.String("hostname", relay.Hostname))

	return nil
}

func (store *PostgresStore) GetRelay(ctx context.Context, publicKey *types.PublicKey) (*types.Relay, error) {
	query := `SELECT pubkey, hostname, endpoint FROM ` + TableRelays + ` WHERE pubkey=$1;`

	entry := &types.RelayEntry{}
	err := store.DB.Get(entry, query, publicKey.String())
	if err != nil {
		return nil, err
	}
	store.logger.Info("fetched relay from db", zap.String("pubkey", publicKey.String()))

	return types.RelayEntryToRelay(entry)
}

func (store *PostgresStore) GetRelays(ctx context.Context) ([]*types.Relay, error) {
	query := `SELECT pubkey, hostname, endpoint FROM ` + TableRelays + `;`

	var entries []*types.RelayEntry
	err := store.DB.Select(&entries, query)
	if err != nil {
		return nil, err
	}
	store.logger.Info("fetched relays from db")

	return types.RelayEntriesToRelays(entries)
}

func (store *PostgresStore) GetRecordsAnalysisWithinSlotBounds(ctx context.Context, relayPubkey string, slotBounds *types.SlotBounds, filter *types.AnalysisQueryFilter) ([]*types.Record, error) {
	query := `SELECT slot, parent_hash, proposer_pubkey FROM ` + TableBidsAnalysis + `
	WHERE relay_pubkey = '` + relayPubkey + `'`

	// Add a bounds filter.
	query = BuildSlotBoundsFilterClause(query, slotBounds)

	// Add an optional category filter.
	query = BuildCategoryFilterClause(query, filter)

	// Add an order by clause.
	query = query + ` ORDER BY slot DESC`

	// Add a limit clause.
	// TODO: make a limit
	query = query + ` LIMIT ` + strconv.FormatUint(100, 10)

	records := make([]*types.Record, 0)
	err := store.DB.Select(&records, query)
	if err != nil {
		return nil, err
	}

	store.logger.Infow("query executed: get records of analysis within slot bounds", "query", query, "count", len(records))

	return records, nil
}
