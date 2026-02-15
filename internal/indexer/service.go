package indexer

import (
	"context"
	"errors"
	"fmt"
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

func (i *Indexer) Run(ctx context.Context, startBlock, endBlock int64) (int64, error) {
	// Use a separate context for operations to ensure the current block finishes processing
	// even if the shutdown signal is received mid-processing.
	opCtx := context.Background()

	lastProcessedBlock := startBlock - 1

	for num := startBlock; num <= endBlock; num++ {
		select {
		case <-ctx.Done():
			return lastProcessedBlock, nil
		default:
		}
		previousBlock, err := i.store.GetBlockByNumber(opCtx, num-1)
		isFirstRun := false
		if err != nil {
			if errors.Is(err, storage.ErrBlockNotFound) {
				isFirstRun = true
			} else {
				log.Printf("Failed to get previous block %d: %v", num-1, err)
				return lastProcessedBlock, fmt.Errorf("failed to get previous block %d: %v", num-1, err)
			}
		}

		// 1. Fetch
		block, err := i.fetcher.Fetch(opCtx, uint64(num))
		if err != nil {
			log.Printf("Failed to fetch block %d: %v", num, err)
			return lastProcessedBlock, fmt.Errorf("failed to fetch block %d: %v", num, err)
		}

		if !isFirstRun && previousBlock.Hash != block.ParentHash().String() {
			log.Printf("Reorg detected at block %d. Previous DB block hash %s != New block parent hash %s", num, previousBlock.Hash, block.ParentHash().String())

			// 1. Find Common Ancestor
			ancestorBlockNumber, err := i.findCommonAncestor(opCtx, num-1)
			if err != nil {
				return lastProcessedBlock, fmt.Errorf("failed to find common ancestor: %v", err)
			}
			log.Printf("Found common ancestor at block %d", ancestorBlockNumber)

			// 2. Rollback
			err = i.store.MarkBlockReorgedRange(opCtx, ancestorBlockNumber)
			if err != nil {
				return lastProcessedBlock, fmt.Errorf("failed to rollback from block %d: %v", ancestorBlockNumber, err)
			}
			log.Printf("Rolled back data > block %d", ancestorBlockNumber)

			// 3. Resume
			// Set loop variable `num` to ancestorBlockNumber.
			// The loop increment will make it ancestorBlockNumber + 1 for the next iteration.
			lastProcessedBlock = ancestorBlockNumber
			num = ancestorBlockNumber
			continue
		}
		// 2. Insert Block
		// Note: CreateBlock uses ON CONFLICT DO NOTHING.
		// If it exists, we get pgx.ErrNoRows (handled by store.SaveBlock).
		// This is idempotent.
		err = i.store.SaveBlock(opCtx, sqlc.CreateBlockParams{
			Hash:       block.Hash().String(),
			Number:     block.Number().Int64(),
			ParentHash: block.ParentHash().String(),
			Timestamp:  time.Unix(int64(block.Time()), 0),
		})
		if err != nil {
			log.Printf("Failed to save block %d: %v", num, err)
			return lastProcessedBlock, fmt.Errorf("failed to save block %d: %v", num, err)
		}

		// 3. Insert ERC20 Transfers
		erc20Transfers, err := i.fetcher.GetERC20TransfersInRange(opCtx, block.NumberU64(), block.NumberU64())
		if err != nil {
			log.Printf("Failed to get ERC20 transfers for block %d: %v", num, err)
			return lastProcessedBlock, fmt.Errorf("failed to get ERC20 transfers for block %d: %v", num, err)
		}
		count := 0
		for _, transferLog := range erc20Transfers {
			from, to, value, ok := gateway.DecodeERC20TransferLog(transferLog)
			if !ok {
				// NOTE: ERC721 Transfer event has 4 topics and ERC20 Transfer event has 3 topics both have same signature
				// so we can't differentiate between them just by signature
				// un-comment below line to log the error
				// log.Printf("Failed to decode Log for ERC20 Transfer for block %d could be ERC721 if topic count is 4, length: %d", num, len(transferLog.Topics))
				continue
			}
			count++
			// Note: CreateERC20Transfer uses ON CONFLICT DO NOTHING.
			// If it exists, we get pgx.ErrNoRows (handled by store.SaveERC20Transfer).
			// This is idempotent.
			err = i.store.SaveERC20Transfer(opCtx, sqlc.CreateERC20TransferParams{
				TxHash:       transferLog.TxHash.String(),
				LogIndex:     int32(transferLog.Index),
				BlockNumber:  int64(num),
				FromAddress:  from.Hex(),
				ToAddress:    to.Hex(),
				Value:        pgtype.Numeric{Int: value, Valid: true},
				TokenAddress: transferLog.Address.Hex(),
			})
			if err != nil {
				log.Printf("Failed to save ERC20 Transfer for block %d: %v", num, err)
				return lastProcessedBlock, fmt.Errorf("failed to save ERC20 Transfer for block %d: %v", num, err)
			}
		}
		log.Printf("Successfully indexed ERC20 Transfers for block %d and count: %d \n", num, count)

		// 3. Mark Processed (Guard)
		// This step is conceptually mostly for tracking or if we had downstream jobs.
		// Since we process sequentially here, the "SaveBlock" already essentially checkpoints us.
		// However, updating `processed_at` allows us to differentiate "inserted but crashed" vs "fully done".
		err = i.store.MarkBlockProcessed(opCtx, num)
		if err != nil {
			log.Printf("Failed to mark block %d as processed: %v", num, err)
			return lastProcessedBlock, fmt.Errorf("failed to mark block %d as processed: %v", num, err)
		}

		lastProcessedBlock = num
		log.Printf("Successfully indexed block %d \n", num)
		log.Println("--------------------------------")
	}
	return lastProcessedBlock, nil
}

// findCommonAncestor steps back from startBlock verifying checks against canonical chain
// Returns the block number of the first block that matches (Common Ancestor).
func (i *Indexer) findCommonAncestor(ctx context.Context, startBlock int64) (int64, error) {
	// We start from the block we *thought* was the tip (startBlock) and go backwards.
	// Since we called this, we know startBlock is likely invalid (or at least its successor didn't match it).
	// But it's safer to check startBlock again against canonical to be sure, and then descend.

	// Safety limit to prevent infinite loops (though usually 0 is the floor)
	const maxReorgDepth = 1000

	current := startBlock
	depth := 0

	for current >= 0 {
		if depth > maxReorgDepth {
			return 0, fmt.Errorf("reorg depth exceeded safe limit of %d blocks", maxReorgDepth)
		}

		// 1. Get canonical block
		canonicalBlock, err := i.fetcher.Fetch(ctx, uint64(current))
		if err != nil {
			return 0, fmt.Errorf("failed to fetch canonical block %d: %v", current, err)
		}

		// 2. Get local block
		dbBlock, err := i.store.GetBlockByNumber(ctx, current)
		if err != nil {
			// If we don't have this block, it can't be an ancestor?
			// Or maybe we haven't indexed it yet?
			// But we are walking BACK from a block we presumably have.
			// If we fail to get it, it's a critical error.
			return 0, fmt.Errorf("failed to get db block %d: %v", current, err)
		}

		// 3. Compare
		if canonicalBlock.Hash().String() == dbBlock.Hash {
			// Match found! This is the common ancestor.
			return current, nil
		}

		// Mismatch, keep going back
		log.Printf("Block %d mismatch: canonical %s != db %s", current, canonicalBlock.Hash().String(), dbBlock.Hash)
		current--
		depth++
	}

	return 0, fmt.Errorf("no common ancestor found down to block 0")
}
