package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/KhanSufiyanMirza/evm-indexer-go/db/sqlc"
	"github.com/KhanSufiyanMirza/evm-indexer-go/internal/gateway"
	"github.com/KhanSufiyanMirza/evm-indexer-go/internal/indexer"
	"github.com/KhanSufiyanMirza/evm-indexer-go/internal/storage"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	RpcUrl         = "RPC_URL"
	StartBlock     = "START_BLOCK"
	SafeBlockDepth = "SAFE_BLOCK_DEPTH"
)

func main() {
	// 1. Setup Database using existing sqlc.NewStore
	sqlcStore, err := sqlc.NewStore()
	if err != nil {
		slog.Error("Failed to create store", "error", err)
		os.Exit(1)
	}
	defer sqlcStore.Close()

	if err := sqlcStore.Ping(context.Background()); err != nil {
		slog.Error("Failed to ping DB", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to DB successfully")

	storageStore := storage.NewStore(sqlcStore)

	// 2. Setup Eth Client
	rawurl, exist := os.LookupEnv(RpcUrl)
	if !exist || rawurl == "" {
		slog.Warn("RPC_URL is missing, continuing with default RPC")
		rawurl = "https://eth.llamarpc.com"
	}

	client, err := ethclient.Dial(rawurl)
	if err != nil {
		slog.Error("Failed to dial RPC", "error", err)
		os.Exit(1)
	}
	defer client.Close()
	safeBlockDepth, err := getSafeBlockDepth()
	if err != nil {
		slog.Error("Failed to get safe block depth", "error", err)
		os.Exit(1)
	}
	slog.Info("Safe block depth configured", "depth", safeBlockDepth)

	// 3. Setup Fetcher
	fetcher := gateway.NewBlockFetcher(client)

	// 4. Determine Range
	latestBlockNumberOnchain, err := fetcher.GetBlockNumberWithRetry(context.Background())
	if err != nil {
		slog.Error("Failed to get latest block", "error", err)
		os.Exit(1)
	}
	slog.Info("Latest onchain block", "block", latestBlockNumberOnchain)

	processedLastBlock, err := storageStore.GetLatestProcessedBlockNumber(context.Background())
	if err != nil && !errors.Is(err, storage.ErrBlockNotFound) {
		slog.Error("Failed to get latest processed block number", "error", err)
		os.Exit(1)
	}

	if processedLastBlock == 0 {
		startBlock, err := getStartBlock()
		if err != nil {
			slog.Error("Failed to get start block", "error", err)
			os.Exit(1)
		}
		processedLastBlock = int64(startBlock) - 1
	}

	start := processedLastBlock + 1
	end := int64(latestBlockNumberOnchain - safeBlockDepth)
	slog.Info("Indexing range determined", "lastProcessed", processedLastBlock, "latestOnchain", latestBlockNumberOnchain, "diff", latestBlockNumberOnchain-uint64(processedLastBlock))
	// fmt.Println("Hit Enter to continue or Ctrl+C to exit")
	// fmt.Scanln()
	slog.Info("Starting indexing", "from", start, "to", end)
	slog.Info("---------------------------------------------")

	// 5. Run Indexer
	idx := indexer.NewIndexer(fetcher, storageStore)

	// We use signal.NotifyContext to handle graceful shutdown in background it
	// spawns a new goroutine to wait for a signal and returns a context that is
	// canceled when the signal is received.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startTime := time.Now()
	lastProcessedBlock, err := idx.Run(ctx, start, end)
	if err != nil {
		slog.Error("Indexer stopped with error", "error", err)
	}
	slog.Info("Indexing complete", "blocksIndexed", lastProcessedBlock-start+1, "duration", time.Since(startTime))
}

func getStartBlock() (uint64, error) {
	startBlockStr, exist := os.LookupEnv(StartBlock)
	if !exist {
		return 0, fmt.Errorf("%s", "start block missing..!")
	}
	startBlock, err := strconv.ParseUint(startBlockStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return startBlock, nil
}
func getSafeBlockDepth() (uint64, error) {
	var defaultBlockDepth uint64 = 12
	safeBlockDepthStr, exist := os.LookupEnv(SafeBlockDepth)
	if !exist {
		return defaultBlockDepth, nil // default safe block depth is 12
	}
	safeBlockDepth, err := strconv.ParseUint(safeBlockDepthStr, 10, 64)
	if err != nil {
		slog.Warn("Failed to parse safe block depth, using default", "error", err, "default", defaultBlockDepth)
		return defaultBlockDepth, nil // default safe block depth is 12
	}
	return safeBlockDepth, nil
}
