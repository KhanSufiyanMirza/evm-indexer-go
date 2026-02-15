package gateway

import (
	"context"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func init() {
	erc20TransferEventHash = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
}

// keccak256("Transfer(address,address,uint256)")
var erc20TransferEventHash common.Hash

type BlockFetcher interface {
	Fetch(ctx context.Context, blockNumber uint64) (*types.Block, error)
	GetBlockNumberWithRetry(ctx context.Context) (uint64, error)
	GetLogsInRange(ctx context.Context, startBlock, endBlock uint64) ([]types.Log, error)
	GetERC20TransfersInRange(ctx context.Context, startBlock, endBlock uint64) ([]types.Log, error)
}
type blockFetcher struct {
	client *ethclient.Client
}

// BlockFetcher is a function that fetches a block by its number.
// type BlockFetcher func(ctx context.Context, blockNumber uint64) (*types.Block, error)

// var retryableCodes = map[int]bool{
// 	-32001: true, // resource not found (node lag)
// 	-32002: true, // resource unavailable
// 	-32005: true, // rate limit
// 	-32603: true, // internal error
// 	-32016: true, // over rate limit
// }

// NewBlockFetcher returns a BlockFetcher that fetches blocks using the provided ethclient.Client.
func NewBlockFetcher(client *ethclient.Client) BlockFetcher {
	return &blockFetcher{client: client}
}

func (bf *blockFetcher) Fetch(ctx context.Context, blockNumber uint64) (*types.Block, error) {
	st := time.Now()
	defer func() {
		slog.Info("Block fetched", "block", blockNumber, "duration", time.Since(st))
	}()
	count := 1
	block, err := backoff.Retry(ctx, func() (*types.Block, error) {
		slog.Info("Fetching block", "block", blockNumber, "attempt", count)
		block, err := bf.client.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))

		if err != nil && !isRetryableError(err) {
			slog.Error("Non-retryable RPC error", "error", err)

			return nil, backoff.Permanent(err)
		}
		count++
		return block, err
	}, backoff.WithMaxTries(5))

	return block, err
}

func (bf *blockFetcher) GetBlockNumberWithRetry(ctx context.Context) (uint64, error) {
	st := time.Now()
	defer func() {
		slog.Info("Block number fetched", "duration", time.Since(st))
	}()
	count := 1
	blockNumber, err := backoff.Retry(ctx, func() (uint64, error) {
		slog.Info("Fetching block number", "attempt", count)
		blockNumber, err := bf.client.BlockNumber(ctx)

		if err != nil && !isRetryableError(err) {
			slog.Error("Non-retryable RPC error", "error", err)
			return 0, backoff.Permanent(err)
		}
		count++
		return blockNumber, err
	}, backoff.WithMaxTries(5))

	return blockNumber, err
}

// GetLogsInRange fetches logs from startBlock to endBlock with retry logic.
func (bf *blockFetcher) GetLogsInRange(ctx context.Context, startBlock, endBlock uint64) ([]types.Log, error) {
	st := time.Now()
	defer func() {
		slog.Info("Logs fetched", "startBlock", startBlock, "endBlock", endBlock, "duration", time.Since(st))
	}()
	count := 1
	logs, err := backoff.Retry(ctx, func() ([]types.Log, error) {
		slog.Info("Fetching logs", "startBlock", startBlock, "endBlock", endBlock, "attempt", count)
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
		}
		logs, err := bf.client.FilterLogs(ctx, query)

		if err != nil && !isRetryableError(err) {
			slog.Error("Non-retryable RPC error fetching logs", "error", err)
			return nil, backoff.Permanent(err)
		}
		count++
		return logs, err
	}, backoff.WithMaxTries(5))

	return logs, err
}

func (bf *blockFetcher) GetERC20TransfersInRange(ctx context.Context, startBlock, endBlock uint64) ([]types.Log, error) {
	st := time.Now()
	defer func() {
		slog.Info("ERC20 transfer logs fetched", "startBlock", startBlock, "endBlock", endBlock, "duration", time.Since(st))
	}()
	count := 1
	logs, err := backoff.Retry(ctx, func() ([]types.Log, error) {
		slog.Info("Fetching ERC20 transfer logs", "startBlock", startBlock, "endBlock", endBlock, "attempt", count)
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
			Topics:    [][]common.Hash{{erc20TransferEventHash}},
		}
		logs, err := bf.client.FilterLogs(ctx, query)

		if err != nil && !isRetryableError(err) {
			slog.Error("Non-retryable RPC error fetching ERC20 transfer logs", "error", err)
			return nil, backoff.Permanent(err)
		}
		count++
		return logs, err
	}, backoff.WithMaxTries(5))

	return logs, err
}
func DecodeERC20TransferLog(log types.Log) (from common.Address, to common.Address, value *big.Int, ok bool) {
	if len(log.Topics) != 3 {
		return common.Address{}, common.Address{}, nil, false
	}

	from = common.BytesToAddress(log.Topics[1].Bytes())
	to = common.BytesToAddress(log.Topics[2].Bytes())
	value = new(big.Int).SetBytes(log.Data)

	return from, to, value, true
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
	case strings.Contains(msg, "no response"):
		return true
	case strings.Contains(msg, "connection reset by peer"):
		return true
	default:
		return false
	}
}

// func isRetryableRPCError(err error) bool {
// 	// fixme: as we know we're using go-ethereum ethclient, we can't use rpc.Error type directly becasue it's json-rpc error
// 	// so we have to find a way to check if the error is retryable
// 	var rpcErr rpc.Error // go-ethereum rpc error
// 	if errors.As(err, &rpcErr) {
// 		return retryableCodes[rpcErr.ErrorCode()]
// 	}
// 	// Fallback to string checking
// 	if isRetryableError(err) {
// 		return true
// 	}
// 	return false
// }
