package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/KhanSufiyanMirza/evm-indexer-go/db/sqlc"
	"github.com/cenkalti/backoff/v5"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	RpcUrl     = "RPC_URL"
	StartBlock = "START_BLOCK"
)

func main() {
	rawurl, exist := os.LookupEnv(RpcUrl)
	if !exist || rawurl == "" {
		log.Println("RPC_URL is missing!!! so continuing with default RPC")
		rawurl = "https://eth.llamarpc.com"
	}

	client, err := ethclient.Dial(rawurl)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer client.Close()

	latestBlockNumberOnchain, err := client.BlockNumber(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Latest Block No: %d \n", latestBlockNumberOnchain)
	fetchBlockByNumer := getBlockFetcherWithRetry(client)
	latestBlock, err := fetchBlockByNumer(
		context.Background(),
		latestBlockNumberOnchain)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("latestBlock.Number(): %v\n", latestBlock.Number())
	fmt.Printf("latestBlock.Hash(): %v\n", latestBlock.Hash())
	fmt.Printf("latestBlock.ParentHash(): %v\n", latestBlock.ParentHash())
	fmt.Printf("latestBlock.Transactions().Len(): %v\n", latestBlock.Transactions().Len())

	store, err := sqlc.NewStore()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer store.Close()
	err = store.Ping(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("Connected to DB successfully")
	processedLastBlock, err := store.GetLatestBlockNumber(context.Background())
	if err != nil {
		startBlock, err := getStartBlock()
		if err != nil {
			log.Fatal("checkpointer is missing...!")
		}
		processedLastBlock = int64(startBlock) - 1
	}

	// blocks, err := fetch10SqBlocks(lastestBlockNumberOnchain, client)
	// if err != nil {
	// 	log.Fatal("failed to fetch 10 seq blocks")
	// }
	// log.Println("10 blocks Fetched successfully..!")
	//
	// err = store.ExecTx(context.Background(), func(q *sqlc.Queries) error {
	// 	for _, b := range blocks {
	// 		_, err := q.CreateBlock(context.Background(), sqlc.CreateBlockParams{
	// 			Hash:       b.Hash().String(),
	// 			Number:     b.Number().Int64(),
	// 			ParentHash: b.ParentHash().String(),
	// 			Timestamp:  time.Unix(int64(b.Time()), 0),
	// 		})
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// 	return nil
	// })
	//
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }
	log.Printf("LastProcessed Block : %d and LatestBlock on chain : %d", processedLastBlock, latestBlockNumberOnchain)
	log.Println("HIT ENTER or N + ENTER to abort")
	var action string
	fmt.Scanln(&action)
	if strings.ToLower(action) == "n" {
		log.Println("aborting...!")
		return
	}
	ctx := context.Background()
	for i := processedLastBlock + 1; i <= int64(latestBlockNumberOnchain); i++ {
		block, err := fetchBlockByNumer(ctx, uint64(i))
		if err != nil {
			log.Fatal(err.Error())
		}
		fmt.Printf("block.Number(): %v\n", block.Number())
		fmt.Printf("block.Hash(): %v\n", block.Hash())

		_, err = store.CreateBlock(ctx, sqlc.CreateBlockParams{
			Hash:       block.Hash().String(),
			Number:     block.Number().Int64(),
			ParentHash: block.ParentHash().String(),
			Timestamp:  time.Unix(int64(block.Time()), 0),
		})
		if err != nil {
			log.Fatal(err.Error(), "block failed to record", block.Number().String())
		}
		log.Printf("Inserted block record successfully : %d", i)
	}
	log.Printf("Insert All blocks from %d to %d ...!", processedLastBlock+1, latestBlockNumberOnchain)
}

func getBlockFetcherWithRetry(client *ethclient.Client) func(ctx context.Context, blockNumber uint64) (*types.Block, error) {

	return func(ctx context.Context, blockNumber uint64) (*types.Block, error) {
		st := time.Now()
		defer func() {
			log.Printf("Time taken to fetch block %d : %s", blockNumber, time.Since(st))
		}()
		count := 1
		block, err := backoff.Retry(ctx, func() (*types.Block, error) {
			log.Println("Fetching block number :", blockNumber, " attempt:", count)
			block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))

			if err != nil && !isRetryableRPCError(err) {
				var rpcErr rpc.Error
				if errors.As(err, &rpcErr) {
					log.Printf("Non-retryable RPC error encountered. Code: %d, Message: %s", rpcErr.ErrorCode(), rpcErr.Error())
				}

				return nil, backoff.Permanent(err)
			}
			count++
			return block, err
		}, backoff.WithMaxTries(5))

		return block, err
	}
}

var retryableCodes = map[int]bool{
	-32001: true, // resource not found (node lag)
	-32002: true, // resource unavailable
	-32005: true, // rate limit
	-32603: true, // internal error
	-32016: true, // over rate limit

}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	switch {
	case strings.Contains(msg, "timeout"):
		return true
	case strings.Contains(msg, "context deadline"):
		return true
	case strings.Contains(msg, "429"):
		return true
	case strings.Contains(msg, "504"):
		return true
	case strings.Contains(msg, "header not found"):
		return true
	default:
		return false
	}
}

func isRetryableRPCError(err error) bool {
	// Fixme: improve detection of rpc.Error as we are using json-rpc under the hood of ethclient it seems it is not detected properly
	// and it falls back to custom error detection
	var rpcErr rpc.Error // go-ethereum rpc error
	if errors.As(err, &rpcErr) {
		return retryableCodes[rpcErr.ErrorCode()]
	}
	log.Println("Not a go-ethereum rpc error, checking custom RPCError...", err.Error())
	if isRetryableError(err) {
		return true
	}
	return false
}
func fetch10SqBlocks(lastestBlockNumber uint64, client *ethclient.Client) ([]*types.Block, error) {
	var blocks = make([]*types.Block, 0, 10)
	ctx := context.Background()
	for i := lastestBlockNumber - 10; i <= lastestBlockNumber; i++ {
		b, err := client.BlockByNumber(ctx, big.NewInt(int64(i)))
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
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
