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
)

const (
	RpcUrl                = "RPC_URL"
	StartBlock            = "START_BLOCK"
	SafeBlockDepth        = "SAFE_BLOCK_DEPTH"
	defaultSafeBlockDepth = 12
	IngestionBlockDepth   = "INGESTION_BLOCK_DEPTH"
	Continuous            = "CONTINUOUS"
	BlockPollInterval     = "BLOCK_POLL_INTERVAL"
	defaultPollInterval   = 12 * time.Second // ~Ethereum block time
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
	ingestionBlockDepth, err := getIngestionBlockDepth()
	if err != nil {
		slog.Error("Failed to get block depth", "error", err)
		os.Exit(1)
	}
	slog.Info("block depth configured for ingestion", "depth", ingestionBlockDepth)

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
	end := int64(latestBlockNumberOnchain - ingestionBlockDepth)
	slog.Info("Indexing range determined", "lastProcessed", processedLastBlock, "latestOnchain", latestBlockNumberOnchain, "diff", latestBlockNumberOnchain-uint64(processedLastBlock))

	runContinuous := getContinuous()
	pollInterval := getBlockPollInterval()
	if runContinuous {
		slog.Info("Continuous mode enabled", "pollInterval", pollInterval)
	}
	slog.Info("Starting indexing", "from", start, "to", end)
	slog.Info("---------------------------------------------")

	// 5. Run Indexer
	idx := indexer.NewIndexer(fetcher, storageStore)

	// We use signal.NotifyContext to handle graceful shutdown in background it
	// spawns a new goroutine to wait for a signal and returns a context that is
	// canceled when the signal is received.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	// run finality modelling in background
	safeBlockDepth, err := getSafeBlockDepth()
	if err != nil {
		slog.Error("Failed to get safe block depth", "error", err)
		os.Exit(1)
	}
	go func() {
		err = idx.RunFinalizer(ctx, safeBlockDepth)
		if err != nil {
			slog.Error("Finalizer stopped with error", "error", err)
		}
	}()
	// run indexer
	startTime := time.Now()
	lastProcessedBlock, err := idx.Run(ctx, start, end)
	if err != nil {
		slog.Error("Indexer stopped with error", "error", err)
	}
	slog.Info("Indexing complete", "blocksIndexed", lastProcessedBlock-start+1, "duration", time.Since(startTime))

	// 6. Continuous mode: keep polling for new blocks until shutdown
	if runContinuous && err == nil {
		slog.Info("Entering continuous mode; polling for new blocks")
		for {
			select {
			case <-ctx.Done():
				slog.Info("Shutdown signal received, exiting continuous mode")
				return
			case <-time.After(pollInterval):
			}
			latest, err := fetcher.GetBlockNumberWithRetry(ctx)
			if err != nil {
				slog.Error("Failed to get latest block in continuous mode", "error", err)
				continue
			}
			start = lastProcessedBlock + 1
			end = int64(latest) - int64(ingestionBlockDepth)
			if start > end {
				slog.Debug("No new blocks to index", "lastProcessed", lastProcessedBlock, "latest", latest)
				continue
			}
			slog.Info("New blocks available", "from", start, "to", end)
			lastProcessedBlock, err = idx.Run(ctx, start, end)
			if err != nil {
				slog.Error("Indexer stopped with error in continuous mode", "error", err)
				continue
			}
			slog.Info("Caught up", "lastProcessed", lastProcessedBlock, "blocksIndexed", lastProcessedBlock-start+1)
		}
	}
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
func getIngestionBlockDepth() (uint64, error) {

	blockDepthStr, exist := os.LookupEnv(IngestionBlockDepth)
	if !exist {
		return defaultSafeBlockDepth, nil // default safe block depth is 12
	}
	blockDepth, err := strconv.ParseUint(blockDepthStr, 10, 64)
	if err != nil {
		slog.Warn("Failed to parse block depth, using default safe block depth", "error", err, "default", defaultSafeBlockDepth)
		return defaultSafeBlockDepth, nil // default safe block depth is 12
	}
	return blockDepth, nil
}

func getContinuous() bool {
	s, _ := os.LookupEnv(Continuous)
	switch s {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func getBlockPollInterval() time.Duration {
	s, exist := os.LookupEnv(BlockPollInterval)
	if !exist || s == "" {
		return defaultPollInterval
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		slog.Warn("Invalid BLOCK_POLL_INTERVAL, using default", "error", err, "default", defaultPollInterval)
		return defaultPollInterval
	}
	if d < time.Second {
		d = time.Second
	}
	return d
}

func getSafeBlockDepth() (uint64, error) {

	blockDepthStr, exist := os.LookupEnv(SafeBlockDepth)
	if !exist {
		return defaultSafeBlockDepth, nil // default safe block depth is 12
	}
	blockDepth, err := strconv.ParseUint(blockDepthStr, 10, 64)
	if err != nil {
		slog.Warn("Failed to parse block depth, using default safe block depth", "error", err, "default", defaultSafeBlockDepth)
		return defaultSafeBlockDepth, nil // default safe block depth is 12
	}
	return blockDepth, nil
}
