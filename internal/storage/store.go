package storage

import (
	"context"
	"errors"
	"log"

	"github.com/KhanSufiyanMirza/evm-indexer-go/db/sqlc"
	"github.com/cenkalti/backoff/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Store struct {
	sqlc.Store
}

func NewStore(store sqlc.Store) *Store {
	return &Store{
		Store: store,
	}
}

var (
	ErrBlockNotFound = errors.New("block not found")
)

// SaveBlock attempts to insert a block.
// If the block already exists (pgx.ErrNoRows due to ON CONFLICT DO NOTHING),
// it returns nil (treating it as success - Idempotency).
func (s *Store) SaveBlock(ctx context.Context, params sqlc.CreateBlockParams) error {
	// uncomment count and log line to see retry attempts and error
	// count := 0
	_, err := retry(ctx, func() (sqlc.CreateBlockRow, error) {
		res, err := s.CreateBlock(ctx, params)
		if err != nil {
			// count++
			// log.Printf("Error inserting block %d, attempt %d: %v", params.Number, count, err)
			if errors.Is(err, pgx.ErrNoRows) {
				log.Printf("Block %d already exists (idempotent skip)", params.Number)
				return sqlc.CreateBlockRow{}, nil
			}
			if isConstraintViolation(err) {
				return sqlc.CreateBlockRow{}, backoff.Permanent(err)
			}
			return sqlc.CreateBlockRow{}, err
		}
		return res, nil
	})
	return err
}

func (s *Store) SaveERC20Transfer(ctx context.Context, params sqlc.CreateERC20TransferParams) error {
	// uncomment count and log line to see retry attempts and error
	// count := 0
	_, err := retry(ctx, func() (sqlc.CreateERC20TransferRow, error) {
		res, err := s.CreateERC20Transfer(ctx, params)
		if err != nil {
			// count++
			// log.Printf("Error inserting ERC20 Transfer %s-%d, attempt %d: %v", params.TxHash, params.LogIndex, count, err)
			if errors.Is(err, pgx.ErrNoRows) {
				log.Printf("ERC20 Transfer %s-%d already exists (idempotent skip)", params.TxHash, params.LogIndex)
				return sqlc.CreateERC20TransferRow{}, nil
			}
			if isConstraintViolation(err) {
				return sqlc.CreateERC20TransferRow{}, backoff.Permanent(err)
			}
			return sqlc.CreateERC20TransferRow{}, err
		}
		return res, nil
	})
	return err
}

func (s *Store) MarkBlockProcessed(ctx context.Context, blockNumber int64) error {
	_, err := retry(ctx, func() (bool, error) {
		err := s.Store.MarkBlockProcessed(ctx, blockNumber)
		if err != nil {
			return false, err
		}
		return true, nil
	})
	return err
}

func (s *Store) GetLatestBlockNumber(ctx context.Context) (int64, error) {
	return retry(ctx, func() (int64, error) {
		blockNo, err := s.Store.GetLatestBlockNumber(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, backoff.Permanent(ErrBlockNotFound)
		}
		return blockNo, err
	})
}
func (s *Store) GetBlockByNumber(ctx context.Context, blockNumber int64) (sqlc.GetBlockByNumberRow, error) {
	return retry(ctx, func() (sqlc.GetBlockByNumberRow, error) {
		blockNo, err := s.Store.GetBlockByNumber(ctx, blockNumber)
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.GetBlockByNumberRow{}, backoff.Permanent(ErrBlockNotFound)
		}
		return blockNo, err
	})
}

func (s *Store) GetLatestProcessedBlockNumber(ctx context.Context) (int64, error) {
	return retry(ctx, func() (int64, error) {
		blockNo, err := s.Store.GetLatestProcessedBlockNumber(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, backoff.Permanent(ErrBlockNotFound)
		}
		return blockNo, err
	})
}

func retry[T any](ctx context.Context, op func() (T, error)) (T, error) {
	return backoff.Retry(ctx, op)
}

func isConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 23xxx are integrity constraint violation
		// 23505 is unique_violation
		// 23503 is foreign_key_violation
		// 23502 is not_null_violation
		// 23514 is check_violation
		// 23P01 is exclusion_violation
		return len(pgErr.Code) >= 2 && pgErr.Code[:2] == "23"
	}
	return false
}

func (s *Store) DeleteBlockRange(ctx context.Context, fromBlock int64) error {
	// We delete in reverse order of dependencies:
	// 1. ERC20 Transfers (refer to blocks)
	// 2. Blocks
	// Note: If you have more tables, add them here.

	// 1. Delete ERC20 Transfers
	_, err := retry(ctx, func() (bool, error) {
		err := s.Store.ExecTx(ctx, func(querier *sqlc.Queries) error {
			err := querier.DeleteERC20TransfersFromHeight(ctx, fromBlock)
			if err != nil {
				return err
			}

			err = querier.DeleteBlocksFromHeight(ctx, fromBlock)
			return err
		})
		if err != nil {
			return false, err
		}
		return true, nil
	})

	return err
}
func (s *Store) MarkBlockReorgedRange(ctx context.Context, fromBlock int64) error {
	// We mark in reverse order of dependencies:
	// 1. ERC20 Transfers (refer to blocks)
	// 2. Blocks
	// Note: If you have more tables, add them here.

	// 1. Mark ERC20 Transfers
	_, err := retry(ctx, func() (bool, error) {
		err := s.Store.ExecTx(ctx, func(querier *sqlc.Queries) error {
			err := querier.MarkBlockReorgedRange(ctx, fromBlock)
			if err != nil {
				return err
			}

			err = querier.MarkERC20TransfersReorgedRange(ctx, fromBlock)
			return err
		})
		if err != nil {
			return false, err
		}
		return true, nil
	})

	return err
}
