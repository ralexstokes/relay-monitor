package store

import (
	"context"
	"database/sql"
	"os"

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
