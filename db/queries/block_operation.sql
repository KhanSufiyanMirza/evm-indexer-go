-- name: CreateBlock :one
INSERT INTO blocks (hash, number, parent_hash, timestamp)
VALUES ($1, $2, $3, $4)
ON CONFLICT (hash, number) DO UPDATE SET is_canonical = TRUE, reorg_detected_at = NULL
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
WHERE number = $1 AND is_canonical = TRUE;

-- name: ListBlocks :many
SELECT id, hash, number, parent_hash, timestamp
FROM blocks
WHERE is_canonical = TRUE
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
SELECT MAX(number)::Bigint FROM blocks WHERE is_canonical = TRUE HAVING MAX(number) IS NOT NULL;

-- name: GetLatestProcessedBlockNumber :one
SELECT MAX(number)::Bigint FROM blocks WHERE processed_at IS NOT NULL AND is_canonical = TRUE HAVING MAX(number) IS NOT NULL;

-- name: MarkBlockProcessed :exec
UPDATE blocks
SET processed_at = NOW()
WHERE number = $1 AND processed_at IS NULL;

-- name: DeleteBlocksFromHeight :exec
DELETE FROM blocks
WHERE number > $1;

-- name: MarkBlockReorgedRange :exec
UPDATE blocks
SET is_canonical = FALSE, reorg_detected_at = NOW()
WHERE number > $1;

-- name: MarkBlockFinalized :exec
UPDATE blocks
SET status = 'FINALIZED'
WHERE number <= $1 AND is_canonical = TRUE AND status != 'FINALIZED';