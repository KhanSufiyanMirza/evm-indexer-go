package indexer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	lastProcessedBlock := startBlock - 1

	for num := startBlock; num <= endBlock; num++ {
		select {
		case <-ctx.Done():
			return lastProcessedBlock, nil
		default:
		}

		// Use a separate context with a timeout to ensure the current block finishes processing
		// even if the shutdown signal is received mid-processing, but with a bounded deadline.
		opCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

		previousBlock, err := i.store.GetBlockByNumber(opCtx, num-1)
		isFirstRun := false
		if err != nil {
			if errors.Is(err, storage.ErrBlockNotFound) {
				isFirstRun = true
			} else {
				slog.Error("Failed to get previous block", "block", num-1, "error", err)
				cancel()
				return lastProcessedBlock, fmt.Errorf("failed to get previous block %d: %w", num-1, err)
			}
		}

		// 1. Fetch
		block, err := i.fetcher.Fetch(opCtx, uint64(num))
		if err != nil {
			slog.Error("Failed to fetch block", "block", num, "error", err)
			cancel()
			return lastProcessedBlock, fmt.Errorf("failed to fetch block %d: %w", num, err)
		}

		if !isFirstRun && previousBlock.Hash != block.ParentHash().String() {
			slog.Warn("Reorg detected", "block", num, "dbHash", previousBlock.Hash, "parentHash", block.ParentHash().String())

			// 1. Find Common Ancestor
			ancestorBlockNumber, err := i.findCommonAncestor(opCtx, num-1)
			if err != nil {
				cancel()
				return lastProcessedBlock, fmt.Errorf("failed to find common ancestor: %w", err)
			}
			slog.Info("Found common ancestor", "block", ancestorBlockNumber)

			// 2. Rollback
			err = i.store.MarkBlockReorgedRange(opCtx, ancestorBlockNumber)
			if err != nil {
				cancel()
				return lastProcessedBlock, fmt.Errorf("failed to rollback from block %d: %w", ancestorBlockNumber, err)
			}
			slog.Info("Rolled back data above block", "block", ancestorBlockNumber)

			// 3. Resume
			// Set loop variable `num` to ancestorBlockNumber.
			// The loop increment will make it ancestorBlockNumber + 1 for the next iteration.
			lastProcessedBlock = ancestorBlockNumber
			num = ancestorBlockNumber
			cancel()
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
			slog.Error("Failed to save block", "block", num, "error", err)
			cancel()
			return lastProcessedBlock, fmt.Errorf("failed to save block %d: %w", num, err)
		}

		// 3. Insert ERC20 Transfers (batch)
		erc20Transfers, err := i.fetcher.GetERC20TransfersInRange(opCtx, block.NumberU64(), block.NumberU64())
		if err != nil {
			slog.Error("Failed to get ERC20 transfers", "block", num, "error", err)
			cancel()
			return lastProcessedBlock, fmt.Errorf("failed to get ERC20 transfers for block %d: %w", num, err)
		}
		batchParams := make([]sqlc.BatchCreateERC20TransferParams, 0, len(erc20Transfers))
		for _, transferLog := range erc20Transfers {
			from, to, value, ok := gateway.DecodeERC20TransferLog(transferLog)
			if !ok {
				// NOTE: ERC721 Transfer event has 4 topics and ERC20 Transfer event has 3 topics both have same signature
				// so we can't differentiate between them just by signature
				continue
			}
			batchParams = append(batchParams, sqlc.BatchCreateERC20TransferParams{
				TxHash:       transferLog.TxHash.String(),
				LogIndex:     int32(transferLog.Index),
				BlockNumber:  int64(num),
				FromAddress:  from.Hex(),
				ToAddress:    to.Hex(),
				Value:        pgtype.Numeric{Int: value, Valid: true},
				TokenAddress: transferLog.Address.Hex(),
			})
		}
		// Batch insert uses ON CONFLICT DO NOTHING for idempotency.
		err = i.store.SaveERC20TransferBatch(opCtx, batchParams)
		if err != nil {
			slog.Error("Failed to save ERC20 transfers", "block", num, "error", err)
			cancel()
			return lastProcessedBlock, fmt.Errorf("failed to save ERC20 Transfers for block %d: %w", num, err)
		}
		slog.Info("Indexed ERC20 transfers", "block", num, "count", len(batchParams))

		// 3. Mark Processed (Guard)
		// This step is conceptually mostly for tracking or if we had downstream jobs.
		// Since we process sequentially here, the "SaveBlock" already essentially checkpoints us.
		// However, updating `processed_at` allows us to differentiate "inserted but crashed" vs "fully done".
		err = i.store.MarkBlockProcessed(opCtx, num)
		if err != nil {
			slog.Error("Failed to mark block as processed", "block", num, "error", err)
			cancel()
			return lastProcessedBlock, fmt.Errorf("failed to mark block %d as processed: %w", num, err)
		}

		lastProcessedBlock = num
		slog.Info("Successfully indexed block", "block", num)
		slog.Info("----------------- -----------------")
		cancel()
	}
	return lastProcessedBlock, nil
}
func (i *Indexer) RunFinalizer(ctx context.Context, safeBlockDepth uint64) error {
	ticker := time.NewTicker(time.Second * 12)
	defer ticker.Stop()

	var lastFinalizedBlock int64

	for {
		select {
		case <-ctx.Done():
			slog.Info("Finalizer shutting down", "lastFinalizedBlock", lastFinalizedBlock)
			return nil
		case <-ticker.C:
			opCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			blockNumber, err := i.fetcher.GetBlockNumberWithRetry(opCtx)
			if err != nil {
				slog.Error("Failed to get block number", "error", err)
				cancel()
				continue
			}
			finalityBlockNumber := int64(blockNumber) - int64(safeBlockDepth)
			if finalityBlockNumber <= 0 || finalityBlockNumber <= lastFinalizedBlock {
				cancel()
				continue
			}
			err = i.store.MarkBlockFinalized(opCtx, finalityBlockNumber)
			if err != nil {
				slog.Error("Failed to mark block as finalized", "block", finalityBlockNumber, "error", err)
				cancel()
				continue
			}
			slog.Info("Finalized blocks", "upTo", finalityBlockNumber, "advanced", finalityBlockNumber-lastFinalizedBlock)
			lastFinalizedBlock = finalityBlockNumber
			cancel()
		}
	}
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
			return 0, fmt.Errorf("failed to fetch canonical block %d: %w", current, err)
		}

		// 2. Get local block
		dbBlock, err := i.store.GetBlockByNumber(ctx, current)
		if err != nil {
			// If we don't have this block, it can't be an ancestor?
			// Or maybe we haven't indexed it yet?
			// But we are walking BACK from a block we presumably have.
			// If we fail to get it, it's a critical error.
			return 0, fmt.Errorf("failed to get db block %d: %w", current, err)
		}

		// 3. Compare
		if canonicalBlock.Hash().String() == dbBlock.Hash {
			// Match found! This is the common ancestor.
			return current, nil
		}

		// Mismatch, keep going back
		slog.Warn("Block hash mismatch", "block", current, "canonical", canonicalBlock.Hash().String(), "db", dbBlock.Hash)
		current--
		depth++
	}

	return 0, fmt.Errorf("no common ancestor found down to block 0")
}
