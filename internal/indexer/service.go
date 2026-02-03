package indexer

import (
	"context"
	"log"
	"time"

	"github.com/KhanSufiyanMirza/evm-indexer-go/db/sqlc"
	"github.com/KhanSufiyanMirza/evm-indexer-go/internal/gateway"
	"github.com/KhanSufiyanMirza/evm-indexer-go/internal/storage"
	"github.com/jackc/pgx/v5/pgtype"
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

		// 3. Insert ERC20 Transfers
		erc20Transfers, err := i.fetcher.GetERC20TransfersInRange(ctx, block.NumberU64(), block.NumberU64())
		if err != nil {
			log.Printf("Failed to get ERC20 transfers for block %d: %v", num, err)
			return err
		}
		for _, transferLog := range erc20Transfers {
			from, to, value, err := gateway.DecodeERC20TransferLog(transferLog)
			if err != nil {
				log.Printf("Failed to decode ERC20 Transfer Log for block %d: %v", num, err)
				return err
			}
			err = i.store.SaveERC20Transfer(ctx, sqlc.CreateERC20TransferParams{
				TxHash:      transferLog.TxHash.String(),
				LogIndex:    int32(transferLog.Index),
				BlockNumber: int64(num),
				FromAddress: from.Hex(),
				ToAddress:   to.Hex(),
				Value:       pgtype.Numeric{Int: value, Valid: true},
			})
			if err != nil {
				log.Printf("Failed to save ERC20 Transfer for block %d: %v", num, err)
				return err
			}
		}
		log.Printf("Successfully indexed ERC20 Transfers for block %d and count: %d \n", num, len(erc20Transfers))

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
