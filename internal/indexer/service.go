package indexer

import (
	"context"
	"log"
	"time"

	"github.com/KhanSufiyanMirza/evm-indexer-go/db/sqlc"
	"github.com/KhanSufiyanMirza/evm-indexer-go/internal/gateway"
	"github.com/KhanSufiyanMirza/evm-indexer-go/internal/storage"
)

type Indexer struct {
	fetcher gateway.BlockFetcher
	store   *storage.Store
}

func NewIndexer(fetcher gateway.BlockFetcher, store *storage.Store) *Indexer {
	return &Indexer{
		fetcher: fetcher,
		store:   store,
	}
}

func (i *Indexer) Run(ctx context.Context, startBlock, endBlock int64) error {
	for num := startBlock; num <= endBlock; num++ {
		// 1. Fetch
		block, err := i.fetcher.Fetch(ctx, uint64(num))
		if err != nil {
			log.Printf("Failed to fetch block %d: %v", num, err)
			return err
		}

		// 2. Insert Block
		// Note: CreateBlock uses ON CONFLICT DO NOTHING.
		// If it exists, we get sql.ErrNoRows (handled by store.SaveBlock).
		// This is idempotent.
		err = i.store.SaveBlock(ctx, sqlc.CreateBlockParams{
			Hash:       block.Hash().String(),
			Number:     block.Number().Int64(),
			ParentHash: block.ParentHash().String(),
			Timestamp:  time.Unix(int64(block.Time()), 0),
		})
		if err != nil {
			log.Printf("Failed to save block %d: %v", num, err)
			return err
		}

		// 3. Mark Processed (Guard)
		// This step is conceptually mostly for tracking or if we had downstream jobs.
		// Since we process sequentially here, the "SaveBlock" already essentially checkpoints us.
		// However, updating `processed_at` allows us to differentiate "inserted but crashed" vs "fully done".
		err = i.store.MarkBlockProcessed(ctx, num)
		if err != nil {
			log.Printf("Failed to mark block %d as processed: %v", num, err)
			return err
		}

		log.Printf("Successfully indexed block %d \n", num)
		log.Println("--------------------------------")
	}
	return nil
}
