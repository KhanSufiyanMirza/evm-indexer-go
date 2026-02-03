package main

import (
	"context"
	"fmt"
	"log"
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
	RpcUrl     = "RPC_URL"
	StartBlock = "START_BLOCK"
)

func main() {
	// 1. Setup Database using existing sqlc.NewStore
	sqlcStore, err := sqlc.NewStore()
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	defer sqlcStore.Close()

	if err := sqlcStore.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}
	log.Println("Connected to DB successfully")

	storageStore := storage.NewStore(sqlcStore)

	// 2. Setup Eth Client
	rawurl, exist := os.LookupEnv(RpcUrl)
	if !exist || rawurl == "" {
		log.Println("RPC_URL is missing!!! so continuing with default RPC")
		rawurl = "https://eth.llamarpc.com"
	}

	client, err := ethclient.Dial(rawurl)
	if err != nil {
		log.Fatalf("Failed to dial RPC: %v", err)
	}
	defer client.Close()

	// 3. Setup Fetcher
	fetcher := gateway.NewBlockFetcher(client)

	// 4. Determine Range
	latestBlockNumberOnchain, err := fetcher.GetBlockNumberWithRetry(context.Background())
	if err != nil {
		log.Fatalf("Failed to get latest block: %v", err)
	}
	log.Printf("Latest Block No: %d \n", latestBlockNumberOnchain)

	processedLastBlock, err := storageStore.GetLatestProcessedBlockNumber(context.Background())
	if err != nil {
		startBlock, err := getStartBlock()
		if err != nil {
			log.Println("No previous state and no START_BLOCK env, defaulting to latest-10")
			processedLastBlock = int64(latestBlockNumberOnchain) - 10
		} else {
			processedLastBlock = int64(startBlock) - 1
		}
	}

	start := processedLastBlock + 1
	end := int64(latestBlockNumberOnchain)
	log.Printf("Processed Last Block: %d and Latest Block onchain: %d, total diff: %d \nHit Enter to continue or Ctrl+C to exit", processedLastBlock, latestBlockNumberOnchain, latestBlockNumberOnchain-uint64(processedLastBlock))
	fmt.Scanln()
	log.Printf("Starting indexing from %d to %d", start, end)

	// 5. Run Indexer
	idx := indexer.NewIndexer(fetcher, storageStore)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()
	startTime := time.Now()
	if err := idx.Run(ctx, start, end); err != nil {
		log.Printf("Indexer stopped with error: %v", err)
	} else {
		log.Println("Indexer finished successfully")
		log.Printf("Total Blocks Indexed: %d/%s", end-start+1, time.Since(startTime))
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
