package store

import (
	"context"
	"database/sql"
	"encoding/hex"
	"strings"

	"github.com/iov-one/weave/cmd/bnsd/x/account"
	"github.com/iov-one/weave/cmd/bnsd/x/username"

	"github.com/lib/pq"

	sq "github.com/Masterminds/squirrel"
	"github.com/iov-one/block-metrics/pkg/models"
	"github.com/iov-one/weave/errors"
)

// NewStore returns a store that provides an access to our database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type Store struct {
	db *sql.DB
}

var validatorNames = map[string]string{
	"61B819A0BCF4E65AF8B6ED3AB287935074B8C7E3": "Cosmostation",
	"3B4F5C11663DC10A6D32403F945DED42AE1DD362": "StakeWith.Us",
	"8411B44F2FF2CE6A3143121EB5EEC1A23FCF2631": "HashQuark",
	"058078082E8ED2431EA61E69657BE27F0D7456FA": "Node A Team",
	"4A6CDCD260D1527CD1F89ECB5BA3A160FAB3B5F7": "Forbole",
	"A811EB8C0E76991BA241278886625CF081EFF874": "01node.com",
	"4C74A4E2156493E5FB329BE619C188519629CCE3": "IRISnet-Bianjie",
	"89F2E7F9BB4BE83924454234E56302ACA94AE2DA": "ChainLayer",
	"DF97841F98E18B02F670C62830E13C40FFCE9D1E": "syncnode",
	"A5F88A83C831E6D84C83EED33870F4015D0FE94A": "Stake Capital",
}

// InsertValidator adds a validator information into the database. It returns
// the newly created validator ID on success.
// This method returns ErrConflict if the validator cannot be inserted due to
// conflicting data.
func (s *Store) InsertValidator(ctx context.Context, publicKey, address []byte) (int64, error) {
	name := validatorNames[strings.ToUpper(hex.EncodeToString(address))]

	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO validators (public_key, address, name)
		VALUES ($1, $2, $3)
		RETURNING id
	`, publicKey, address, name).Scan(&id)
	return id, castPgErr(err)
}

// ValidatorAddressID returns an ID of a validator with given address. It
// returns ErrNotFound if no such address is present in the database.
func (s *Store) ValidatorAddressID(ctx context.Context, address []byte) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM validators WHERE address = $1
		LIMIT 1
	`, address).Scan(&id)
	return id, castPgErr(err)
}

func (s *Store) InsertBlock(ctx context.Context, b models.Block) error {
	if len(b.ParticipantIDs) == 0 {
		return errors.Wrap(ErrConflict, "no participants on block")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "cannot create transaction")
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO blocks (block_height, block_hash, block_time, proposer_id, messages, fee_frac)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, b.Height, b.Hash, b.Time.UTC(), b.ProposerID, pq.Array(b.Messages), b.FeeFrac)
	if err != nil {
		return wrapPgErr(err, "insert block")
	}

	for _, part := range b.ParticipantIDs {
		_, err = tx.ExecContext(ctx, `
		INSERT INTO block_participations (validated, block_id, validator_id)
		VALUES (true, $1, $2)
		`, b.Height, part)
		if err != nil {
			return wrapPgErr(err, "insert block participant")
		}
	}

	for _, missed := range b.MissingIDs {
		_, err = tx.ExecContext(ctx, `
		INSERT INTO block_participations (validated, block_id, validator_id)
		VALUES (false, $1, $2)
		`, b.Height, missed)
		if err != nil {
			return wrapPgErr(err, "insert block participant")
		}
	}

	for _, transaction := range b.Transactions {
		_, err := tx.ExecContext(ctx, `
		INSERT INTO transactions(transaction_hash, block_id, message)
		VALUES($1, $2, $3)`, transaction.Hash, b.Height, transaction.Message)
		if err != nil {
			return wrapPgErr(err, "insert transaction")
		}
	}

	err = tx.Commit()

	_ = tx.Rollback()

	return wrapPgErr(err, "commit block tx")
}

func (s *Store) InsertAccount(ctx context.Context, a *account.RegisterAccountMsg) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "cannot create transaction")
	}

	query :=
		`INSERT INTO accounts(domain, name, owner, broker)
		  VALUES ($1, $2, $3, $4)
  		  RETURNING id`
	stmt, err := s.db.Prepare(query)
	if err != nil {
		return wrapPgErr(err, "insert account query")
	}
	defer stmt.Close()

	var accountID int
	if err := stmt.QueryRow(a.Domain, a.Name, a.Owner.String(), a.Broker.String()).Scan(&accountID); err != nil {
		return wrapPgErr(err, "insert account")
	}

	for _, target := range a.Targets {
		_, err = tx.ExecContext(ctx, `
		INSERT INTO account_targets (account_id, blockchain_id, address)
		VALUES ($1, $2, $3)
		`, accountID, target.BlockchainID, target.Address)
		if err != nil {
			return wrapPgErr(err, "insert account targets")
		}
	}

	err = tx.Commit()

	_ = tx.Rollback()

	return wrapPgErr(err, "commit account")
}

func (s *Store) ReplaceAccountTargets(ctx context.Context, a *account.ReplaceAccountTargetsMsg) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "cannot create transaction")
	}

	var accountID int
	if err := s.db.QueryRow(`SELECT id FROM accounts WHERE domain = $1 AND name = $2`, a.Domain, a.Name).Scan(&accountID); err != nil {
		return wrapPgErr(err, "cannot get account ID")
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM account_targets WHERE account_id = $1`, accountID); err != nil {
		return wrapPgErr(err, "update account delete query")

	}

	for _, target := range a.NewTargets {
		_, err = tx.ExecContext(ctx, `
		INSERT INTO account_targets (account_id, blockchain_id, address)
		VALUES ($1, $2, $3)
		`, accountID, target.BlockchainID, target.Address)
		if err != nil {
			return wrapPgErr(err, "insert account targets")
		}
	}

	err = tx.Commit()

	_ = tx.Rollback()

	return wrapPgErr(err, "commit account")
}

func (s *Store) InsertUsername(ctx context.Context, a *username.RegisterTokenMsg) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "cannot create transaction")
	}

	query :=
		`INSERT INTO usernames(name)
		  VALUES ($1)
  		  RETURNING id`
	stmt, err := s.db.Prepare(query)
	if err != nil {
		return wrapPgErr(err, "insert username query")
	}
	defer stmt.Close()

	var usernameID int
	if err := stmt.QueryRow(a.Username).Scan(&usernameID); err != nil {
		return wrapPgErr(err, "insert account")
	}

	for _, target := range a.Targets {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO username_targets (username_id, blockchain_id, address)
			VALUES ($1, $2, $3)
			`, usernameID, target.BlockchainID, target.Address)
		if err != nil {
			return wrapPgErr(err, "insert account targets")
		}
	}

	err = tx.Commit()

	_ = tx.Rollback()

	return wrapPgErr(err, "commit account")
}

func (s *Store) ReplaceUsernameTargets(ctx context.Context, a *username.ChangeTokenTargetsMsg) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "cannot create transaction")
	}

	var usernameID int
	if err := s.db.QueryRow(`SELECT id FROM usernames WHERE name = $1`, a.Username).Scan(&usernameID); err != nil {
		return wrapPgErr(err, "cannot get account ID")
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM username_targets WHERE username_id = $1`, usernameID); err != nil {
		return wrapPgErr(err, "update username delete query")

	}

	for _, target := range a.NewTargets {
		_, err = tx.ExecContext(ctx, `
		INSERT INTO username_targets (username_id, blockchain_id, address)
		VALUES ($1, $2, $3)
		`, usernameID, target.BlockchainID, target.Address)
		if err != nil {
			return wrapPgErr(err, "insert username targets")
		}
	}

	err = tx.Commit()

	_ = tx.Rollback()

	return wrapPgErr(err, "commit account")
}

// LoadLastNBlock returns the last blocks with given count.
// ErrNotFound is returned if no blocks exist.
// ErrLimit is returned if allowed limit is exceeded
// Note that it doesn't load the validators by default
func (s *Store) LastNBlock(ctx context.Context, limit, after int) ([]*models.Block, error) {
	// max number of blocks that is allowed to retrieved is 100
	if limit > 100 {
		return nil, errors.Wrapf(ErrLimit, "limit exceeded")
	}

	var rows *sql.Rows
	var err error
	if after == 0 {
		rows, err = s.db.QueryContext(ctx, `
		SELECT block_height, block_hash, block_time, proposer_id, messages, fee_frac
		FROM blocks
		ORDER BY block_height DESC
		LIMIT $1
	`, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
		SELECT block_height, block_hash, block_time, proposer_id, messages, fee_frac
		FROM blocks
		WHERE block_height < $1
		ORDER BY block_height DESC
		LIMIT $2
	`, after, limit)
	}
	defer rows.Close()

	if err != nil {
		err = castPgErr(err)
		if errors.ErrNotFound.Is(err) {
			return nil, errors.Wrap(err, "no blocks")
		}
		return nil, errors.Wrap(castPgErr(err), "cannot select block")
	}

	var blocks []*models.Block

	for rows.Next() {
		var b models.Block
		err := rows.Scan(&b.Height, &b.Hash, &b.Time, &b.ProposerID, pq.Array(&b.Messages), &b.FeeFrac)
		if err != nil {
			err = castPgErr(err)
			if errors.ErrNotFound.Is(err) {
				return nil, errors.Wrap(err, "no blocks")
			}
			return nil, errors.Wrap(castPgErr(err), "cannot select block")

		}
		txs, err := s.LoadTxsInBlock(ctx, b.Height)
		if err != nil && !errors.ErrNotFound.Is(err) {
			return nil, err
		}
		b.Transactions = txs

		// normalize it here, as not always stored like this in the db
		b.Time = b.Time.UTC()
		b.ParticipantIDs, b.MissingIDs, err = s.loadParticipants(ctx, b.Height)
		if err != nil {
			return nil, err
		}

		// insert validator name to the response
		name, err := s.validatorNameFromProposerID(ctx, b.ProposerID)
		if err != nil {
			return nil, err
		}
		b.ProposerName = name

		blocks = append(blocks, &b)
	}
	if len(blocks) == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "no blocks")
	}
	return blocks, nil
}

func (s *Store) validatorNameFromProposerID(ctx context.Context, proposerID int64) (string, error) {
	var vadr []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT validators.address
		FROM blocks
		INNER JOIN validators ON blocks.proposer_id=validators.id
		AND blocks.proposer_id=$1
	`, proposerID).Scan(&vadr)
	if err != nil {
		return "", err
	}

	return validatorNames[strings.ToUpper(hex.EncodeToString(vadr))], nil
}

// LatestBlock returns the block with the greatest high value. This method
// returns ErrNotFound if no block exist.
// Note that it doesn't load the validators by default
func (s *Store) LatestBlock(ctx context.Context) (*models.Block, error) {
	blocks, err := s.LastNBlock(ctx, 1, 0)
	if err != nil {
		return nil, err
	}
	return blocks[0], nil
}

// LoadBlock returns the block with the given block height from the database.
// This method returns ErrNotFound if no block exist.
// Note that it doesn't load the validators by default
//
// TODO: de-duplicate LatestBlock() code
func (s *Store) LoadBlock(ctx context.Context, blockHeight int64) (*models.Block, error) {
	var b models.Block

	err := s.db.QueryRowContext(ctx, `
		SELECT block_height, block_hash, block_time, proposer_id, messages, fee_frac
		FROM blocks
		WHERE block_height = $1
	`, blockHeight).Scan(&b.Height, &b.Hash, &b.Time, &b.ProposerID, pq.Array(&b.Messages), &b.FeeFrac)

	if err != nil {
		err = castPgErr(err)
		if errors.ErrNotFound.Is(err) {
			return nil, errors.Wrap(err, "no blocks")
		}
		return nil, errors.Wrap(castPgErr(err), "cannot select block")
	}

	txs, err := s.LoadTxsInBlock(ctx, b.Height)
	if err != nil && !errors.ErrNotFound.Is(err) {
		return nil, err
	}
	b.Transactions = txs

	b.ProposerName, err = s.validatorNameFromProposerID(ctx, b.ProposerID)
	if err != nil {
		return nil, err
	}

	// normalize it here, as not always stored like this in the db
	b.Time = b.Time.UTC()
	b.ParticipantIDs, b.MissingIDs, err = s.loadParticipants(ctx, b.Height)
	return &b, err
}

func (s *Store) LoadBlockByHash(ctx context.Context, blockHash string) (*models.Block, error) {
	var b models.Block

	err := s.db.QueryRowContext(ctx, `
		SELECT block_height, block_hash, block_time, proposer_id, messages, fee_frac
		FROM blocks
		WHERE block_hash=$1
	`, blockHash).Scan(&b.Height, &b.Hash, &b.Time, &b.ProposerID, pq.Array(&b.Messages), &b.FeeFrac)

	if err != nil {
		err = castPgErr(err)
		if errors.ErrNotFound.Is(err) {
			return nil, errors.Wrap(err, "no blocks")
		}
		return nil, errors.Wrap(castPgErr(err), "cannot select block")
	}

	txs, err := s.LoadTxsInBlock(ctx, b.Height)
	if err != nil && !errors.ErrNotFound.Is(err) {
		return nil, err
	}
	b.Transactions = txs

	b.ProposerName, err = s.validatorNameFromProposerID(ctx, b.ProposerID)
	if err != nil {
		return nil, err
	}

	// normalize it here, as not always stored like this in the db
	b.Time = b.Time.UTC()
	b.ParticipantIDs, b.MissingIDs, err = s.loadParticipants(ctx, b.Height)
	return &b, err
}

func (s *Store) LoadBlockByHeight(ctx context.Context, blockHeight string) (*models.Block, error) {
	var b models.Block

	err := s.db.QueryRowContext(ctx, `
		SELECT block_height, block_hash, block_time, proposer_id, messages, fee_frac
		FROM blocks
		WHERE block_height=$1
	`, blockHeight).Scan(&b.Height, &b.Hash, &b.Time, &b.ProposerID, pq.Array(&b.Messages), &b.FeeFrac)

	if err != nil {
		err = castPgErr(err)
		if errors.ErrNotFound.Is(err) {
			return nil, errors.Wrap(err, "no block found by height")
		}
		return nil, errors.Wrap(castPgErr(err), "cannot select block")
	}

	txs, err := s.LoadTxsInBlock(ctx, b.Height)
	if err != nil && !errors.ErrNotFound.Is(err) {
		return nil, err
	}
	b.Transactions = txs

	b.ProposerName, err = s.validatorNameFromProposerID(ctx, b.ProposerID)
	if err != nil {
		return nil, err
	}

	// normalize it here, as not always stored like this in the db
	b.Time = b.Time.UTC()
	b.ParticipantIDs, b.MissingIDs, err = s.loadParticipants(ctx, b.Height)
	return &b, err
}

// LoadTx
func (s *Store) LoadTx(ctx context.Context, txHash string) (*models.Transaction, error) {
	var tx models.Transaction

	err := s.db.QueryRowContext(ctx, `
		SELECT transaction_hash, block_id, message
		FROM transactions
		WHERE transaction_hash=$1
	`, txHash).Scan(&tx.Hash, &tx.BlockID, &tx.Message)
	if err == nil {
		return &tx, nil
	}

	err = castPgErr(err)
	if errors.ErrNotFound.Is(err) {
		return nil, errors.Wrap(err, "no transaction")
	}
	return nil, errors.Wrap(castPgErr(err), "cannot select transaction")
}

// LoadLatestNTx
func (s *Store) LoadLatestNTx(ctx context.Context, n int) ([]*models.Transaction, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT transaction_hash, block_id, message
		FROM transactions
		ORDER BY block_id DESC
		LIMIT $1
	`, n)
	defer rows.Close()

	if err != nil {
		err = castPgErr(err)
		if errors.ErrNotFound.Is(err) {
			return nil, errors.Wrap(err, "no transaction")
		}
		return nil, errors.Wrap(castPgErr(err), "cannot select transaction")
	}

	var txs []*models.Transaction

	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(&tx.Hash, &tx.BlockID, &tx.Message)
		if err != nil {
			err = castPgErr(err)
			if errors.ErrNotFound.Is(err) {
				return nil, errors.Wrap(err, "no transactions")
			}

			return nil, errors.Wrap(castPgErr(err), "cannot select transaction")
		}

		txs = append(txs, &tx)
	}

	return txs, nil
}

// LoadTxsInBlock
func (s *Store) LoadTxsInBlock(ctx context.Context, blockHeight int64) ([]models.Transaction, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT transaction_hash, block_id, message
		FROM transactions
		WHERE block_id=$1
	`, blockHeight)
	defer rows.Close()

	if err != nil {
		err = castPgErr(err)
		if errors.ErrNotFound.Is(err) {
			return nil, errors.Wrap(err, "no txs")
		}
		return nil, errors.Wrap(castPgErr(err), "cannot select txs")
	}

	var txs []models.Transaction

	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(&tx.Hash, &tx.BlockID, &tx.Message)
		if err != nil {
			err = castPgErr(err)
			if errors.ErrNotFound.Is(err) {
				return nil, errors.Wrap(err, "no tx")
			}
			return nil, errors.Wrap(castPgErr(err), "cannot select tx")
		}
		txs = append(txs, tx)
	}

	if len(txs) == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "no txs")
	}

	return txs, nil
}

func (s *Store) LoadTxsByParams(ctx context.Context, source, dest, memo string) ([]models.Transaction, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query := psql.Select("transaction_hash, block_id, message").From("transactions").Limit(100)

	if source != "" {
		query = query.Where("message->'details'->>'source' = ?", source)
	}
	if dest != "" {
		query = query.Where("message->'details'->>'destination' = ?", dest)
	}
	if memo != "" {
		query = query.Where("message->'details'->>'memo' = ?", memo)
	}

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		err = castPgErr(err)
		if errors.ErrNotFound.Is(err) {
			return nil, errors.Wrap(err, "no txs")
		}
		return nil, errors.Wrap(castPgErr(err), "cannot select txs")
	}

	var txs []models.Transaction

	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(&tx.Hash, &tx.BlockID, &tx.Message)
		if err != nil {
			err = castPgErr(err)
			if errors.ErrNotFound.Is(err) {
				return nil, errors.Wrap(err, "no tx")
			}
			return nil, errors.Wrap(castPgErr(err), "cannot select tx")
		}
		txs = append(txs, tx)
	}

	if len(txs) == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "no txs")
	}

	return txs, nil
}

// LoadTxsByMemo
func (s *Store) LoadTxsByMemo(ctx context.Context, memo string) ([]models.Transaction, error) {
	rows, err := s.db.QueryContext(ctx, `
			SELECT transaction_hash, block_id, message
			FROM transactions
			AND message -> 'details' ->> 'memo' = $1
		`, memo)
	defer rows.Close()

	if err != nil {
		err = castPgErr(err)
		if errors.ErrNotFound.Is(err) {
			return nil, errors.Wrap(err, "no txs")
		}
		return nil, errors.Wrap(castPgErr(err), "cannot select txs")
	}

	var txs []models.Transaction

	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(&tx.Hash, &tx.BlockID, &tx.Message)
		if err != nil {
			err = castPgErr(err)
			if errors.ErrNotFound.Is(err) {
				return nil, errors.Wrap(err, "no tx")
			}
			return nil, errors.Wrap(castPgErr(err), "cannot select tx")
		}
		txs = append(txs, tx)
	}

	if len(txs) == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "no txs")
	}

	return txs, nil
}

// loadParticipants will load the participants for the given block and update the structure.
// Automatically called as part of Load/LatestBlock to give you the full info
func (s *Store) loadParticipants(ctx context.Context, blockHeight int64) (participants []int64, missing []int64, err error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT validator_id, validated
		FROM block_participations
		WHERE block_id = $1
	`, blockHeight)
	if err != nil {
		err = wrapPgErr(err, "query participants")
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var pid int64
		var validated bool
		if err = rows.Scan(&pid, &validated); err != nil {
			err = wrapPgErr(rows.Err(), "scanning participants")
			return nil, nil, err
		}
		if validated {
			participants = append(participants, pid)
		} else {
			missing = append(missing, pid)
		}
	}

	err = wrapPgErr(rows.Err(), "scanning participants")
	return
}

func (s *Store) LoadAccount(ctx context.Context, name, domain string) (*models.Account, error) {
	var acc models.Account

	err := s.db.QueryRowContext(ctx, `
		SELECT id, domain, name, owner, broker
		FROM accounts
		WHERE name=$1 AND domain=$2
	`, name, domain).Scan(&acc.ID, &acc.Domain, &acc.Name, &acc.Owner, &acc.Broker)
	if err != nil {
		return nil, wrapPgErr(err, "cannot load account")
	}
	return &acc, nil
}

func (s *Store) LoadAccountTargets(ctx context.Context, name, domain string) ([]models.AccountTarget, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_targets.id, account_targets.account_id, account_targets.blockchain_id, account_targets.address
		FROM account_targets
		INNER JOIN accounts ON account_targets.account_id = accounts.id
		AND accounts.name = $1 AND accounts.domain = $2
	`, name, domain)
	defer rows.Close()

	if err != nil {
		err = castPgErr(err)
		if errors.ErrNotFound.Is(err) {
			return nil, errors.Wrap(err, "no accounts")
		}
		return nil, errors.Wrap(err, "cannot select account_targets")
	}

	var accts []models.AccountTarget

	for rows.Next() {
		var acct models.AccountTarget
		err := rows.Scan(&acct.ID, &acct.AccountID, &acct.BlockchainID, &acct.Address)
		if err != nil {
			err = castPgErr(err)
			if errors.ErrNotFound.Is(err) {
				return nil, errors.Wrap(err, "no account target")
			}
			return nil, errors.Wrap(castPgErr(err), "cannot select account target")
		}
		accts = append(accts, acct)
	}

	if len(accts) == 0 {
		return nil, errors.Wrap(errors.ErrNotFound, "no account targets")
	}

	return accts, nil
}
