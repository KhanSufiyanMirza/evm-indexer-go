-- name: CreateBlock :one
INSERT INTO blocks (hash, number, parent_hash, timestamp)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
RETURNING id, hash, number, parent_hash, timestamp;

-- name: GetBlockByID :one
SELECT id, hash, number, parent_hash, timestamp
FROM blocks
WHERE id = $1;

-- name: GetBlockByHash :one
SELECT id, hash, number, parent_hash, timestamp
FROM blocks
WHERE hash = $1;

-- name: GetBlockByNumber :one
SELECT id, hash, number, parent_hash, timestamp
FROM blocks
WHERE number = $1;

-- name: ListBlocks :many
SELECT id, hash, number, parent_hash, timestamp
FROM blocks
ORDER BY number DESC
LIMIT $1 OFFSET $2;

-- name: UpdateBlock :one
UPDATE blocks
SET hash = $2, number = $3, parent_hash = $4, timestamp = $5
WHERE id = $1
RETURNING id, hash, number, parent_hash, timestamp;

-- name: DeleteBlock :exec
DELETE FROM blocks
WHERE id = $1;

-- name: DeleteBlockByHash :exec
DELETE FROM blocks
WHERE hash = $1;

-- name: CountBlocks :one
SELECT COUNT(*) as count
FROM blocks;

-- name: GetLatestBlockNumber :one
SELECT MAX(number)::Bigint FROM blocks;

-- name: GetLatestProcessedBlockNumber :one
SELECT MAX(number)::Bigint FROM blocks WHERE processed_at IS NOT NULL;

-- name: MarkBlockProcessed :exec
UPDATE blocks
SET processed_at = NOW()
WHERE number = $1 AND processed_at IS NULL;