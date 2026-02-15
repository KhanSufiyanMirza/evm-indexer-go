-- name: CreateERC20Transfer :one
INSERT INTO erc20_transfers (tx_hash, log_index, from_address, to_address, value, block_number, token_address)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (tx_hash, log_index) DO NOTHING
RETURNING tx_hash, log_index, from_address, to_address, value, block_number, token_address;

-- name: GetERC20Transfer :one
SELECT tx_hash, log_index, from_address, to_address, value, block_number, token_address
FROM erc20_transfers
WHERE tx_hash = $1 AND log_index = $2;

-- name: ListERC20TransfersByTxHash :many
SELECT tx_hash, log_index, from_address, to_address, value, block_number, token_address
FROM erc20_transfers
WHERE tx_hash = $1
ORDER BY log_index ASC
LIMIT $2 OFFSET $3;

-- name: BatchCreateERC20Transfer :batchexec
INSERT INTO erc20_transfers (tx_hash, log_index, from_address, to_address, value, block_number, token_address)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (tx_hash, log_index) DO NOTHING;

-- name: CountERC20Transfers :one
SELECT COUNT(*) as count
FROM erc20_transfers;

-- name: DeleteERC20TransfersFromHeight :exec
DELETE FROM erc20_transfers
WHERE block_number > $1;

-- name: MarkERC20TransfersReorgedRange :exec
UPDATE erc20_transfers
SET is_canonical = FALSE, reorg_detected_at = NOW()
WHERE block_number > $1;