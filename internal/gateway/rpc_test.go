package gateway

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
)

func TestERC20TransferEventHash(t *testing.T) {

	expectedHashV2 := "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	if expectedHashV2 != strings.Split(erc20TransferEventHash.String(), "0x")[1] {
		t.Errorf("ERC20 Transfer Event hash (Keccak256) mismatch. Expected %s, got %s", expectedHashV2, erc20TransferEventHash.String())
	}
}
func TestGetLogsInRange(t *testing.T) {
	client, err := ethclient.Dial("https://eth.llamarpc.com")
	if err != nil {
		log.Fatalf("Failed to dial RPC: %v", err)
	}
	defer client.Close()
	fetcher := NewBlockFetcher(client)
	endBlock, err := fetcher.GetBlockNumberWithRetry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get latest block number: %v", err)
	}
	startBlock := endBlock - 2

	logs, err := fetcher.GetLogsInRange(
		context.Background(),
		startBlock,
		endBlock,
	)
	if err != nil {
		t.Fatalf("Failed to get logs in range: %v", err)
	}

	t.Logf("Fetched %d logs between blocks %d and %d", len(logs), startBlock, endBlock)
	for i, log := range logs {
		t.Logf("Log %d: %+v", i, log)
	}

}
func TestGetERC20TransfersInRange(t *testing.T) {
	client, err := ethclient.Dial("https://eth.llamarpc.com")
	if err != nil {
		log.Fatalf("Failed to dial RPC: %v", err)
	}
	defer client.Close()
	fetcher := NewBlockFetcher(client)
	endBlock, err := fetcher.GetBlockNumberWithRetry(context.Background())
	if err != nil {
		t.Fatalf("Failed to get latest block number: %v", err)
	}
	startBlock := endBlock - 2

	logs, err := fetcher.GetERC20TransfersInRange(
		context.Background(),
		startBlock,
		endBlock,
	)
	if err != nil {
		t.Fatalf("Failed to get ERC20 Transfer logs in range: %v", err)
	}

	t.Logf("Fetched %d ERC20 Transfer logs between blocks %d and %d", len(logs), startBlock, endBlock)
	for i, log := range logs {
		from, to, value, err := DecodeERC20TransferLog(log)
		if err != nil {
			t.Logf("Failed to decode ERC20 Transfer Log %d: %v", i, err)
		} else {
			t.Logf("ERC20 Transfer Log %d: from=%s, to=%s, value=%s", i, from, to, value)
		}
		logKey := fmt.Sprintf("%s-%d", log.TxHash.String(), log.Index)
		t.Logf("Log Key %d: %s", i, logKey)

	}

}
