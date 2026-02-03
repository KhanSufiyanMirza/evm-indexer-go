package storage

import (
	"context"
	"errors"
	"log"

	"github.com/KhanSufiyanMirza/evm-indexer-go/db/sqlc"
	"github.com/jackc/pgx/v5"
)

type Store struct {
	sqlc.Store
}

func NewStore(store sqlc.Store) *Store {
	return &Store{
		Store: store,
	}
}

// SaveBlock attempts to insert a block.
// If the block already exists (pgx.ErrNoRows due to ON CONFLICT DO NOTHING),
// it returns nil (treating it as success - Idempotency).
func (s *Store) SaveBlock(ctx context.Context, params sqlc.CreateBlockParams) error {
	_, err := s.CreateBlock(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("Block %d already exists (idempotent skip)", params.Number)
			return nil
		}
		return err
	}
	return nil
}
func (s *Store) SaveERC20Transfer(ctx context.Context, params sqlc.CreateERC20TransferParams) error {
	_, err := s.CreateERC20Transfer(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("ERC20 Transfer %s-%d already exists (idempotent skip)", params.TxHash, params.LogIndex)
			return nil
		}
		return err
	}
	return nil
}

func (s *Store) MarkBlockProcessed(ctx context.Context, blockNumber int64) error {
	return s.Store.MarkBlockProcessed(ctx, blockNumber)
}

func (s *Store) GetLatestBlockNumber(ctx context.Context) (int64, error) {
	return s.Store.GetLatestBlockNumber(ctx)
}

func (s *Store) GetLatestProcessedBlockNumber(ctx context.Context) (int64, error) {
	return s.Store.GetLatestProcessedBlockNumber(ctx)
}
