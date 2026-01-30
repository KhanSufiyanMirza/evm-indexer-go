package gateway

import (
	"context"
	"errors"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// BlockFetcher is a function that fetches a block by its number.
type BlockFetcher func(ctx context.Context, blockNumber uint64) (*types.Block, error)

var retryableCodes = map[int]bool{
	-32001: true, // resource not found (node lag)
	-32002: true, // resource unavailable
	-32005: true, // rate limit
	-32603: true, // internal error
	-32016: true, // over rate limit
}

// NewBlockFetcherWithRetry returns a BlockFetcher that retries on transient errors.
func NewBlockFetcherWithRetry(client *ethclient.Client) BlockFetcher {
	return func(ctx context.Context, blockNumber uint64) (*types.Block, error) {
		st := time.Now()
		defer func() {
			log.Printf("Time taken to fetch block %d : %s", blockNumber, time.Since(st))
		}()
		count := 1
		block, err := backoff.Retry(ctx, func() (*types.Block, error) {
			log.Println("Fetching block number :", blockNumber, " attempt:", count)
			block, err := client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))

			if err != nil && !isRetryableError(err) {
				log.Printf("Non-retryable RPC error encountered, Message: %s", err.Error())

				return nil, backoff.Permanent(err)
			}
			count++
			return block, err
		}, backoff.WithMaxTries(5))

		return block, err
	}
}
func GetBlockNumberWithRetry(ctx context.Context, client *ethclient.Client) (uint64, error) {
	st := time.Now()
	defer func() {
		log.Printf("Time taken to get block number : %s", time.Since(st))
	}()
	count := 1
	blockNumber, err := backoff.Retry(ctx, func() (uint64, error) {
		log.Println("Fetching block number attempt:", count)
		blockNumber, err := client.BlockNumber(ctx)

		if err != nil && !isRetryableError(err) {
			log.Printf("Non-retryable RPC error encountered, Message: %s", err.Error())
			return 0, backoff.Permanent(err)
		}
		count++
		return blockNumber, err
	}, backoff.WithMaxTries(5))

	return blockNumber, err
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
	// fixme: as we know we're using go-ethereum ethclient, we can't use rpc.Error type directly becasue it's json-rpc error
	// so we have to find a way to check if the error is retryable
	var rpcErr rpc.Error // go-ethereum rpc error
	if errors.As(err, &rpcErr) {
		return retryableCodes[rpcErr.ErrorCode()]
	}
	// Fallback to string checking
	if isRetryableError(err) {
		return true
	}
	return false
}
