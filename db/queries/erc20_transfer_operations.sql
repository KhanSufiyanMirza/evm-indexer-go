-- name: CreateERC20Transfer :one
INSERT INTO erc20_transfers (tx_hash, log_index, from_address, to_address, value, block_number)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (tx_hash, log_index) DO NOTHING
RETURNING tx_hash, log_index, from_address, to_address, value, block_number;

-- name: GetERC20Transfer :one
SELECT tx_hash, log_index, from_address, to_address, value, block_number
FROM erc20_transfers
WHERE tx_hash = $1 AND log_index = $2;

-- name: ListERC20TransfersByTxHash :many
SELECT tx_hash, log_index, from_address, to_address, value, block_number
FROM erc20_transfers
WHERE tx_hash = $1
ORDER BY log_index ASC
LIMIT $2 OFFSET $3;

-- name: CountERC20Transfers :one
SELECT COUNT(*) as count
FROM erc20_transfers;

-- name: DeleteERC20TransfersFromHeight :exec
DELETE FROM erc20_transfers
WHERE block_number > $1;