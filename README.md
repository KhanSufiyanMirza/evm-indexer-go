# evm-indexer-go

A high-performance EVM blockchain indexer written in Go, designed to ingest and store blockchain data into a PostgreSQL database.

## Prerequisites

Before interacting with the project, ensure you have the following installed:

- **Go** (v1.24 or later)
- **Docker & Docker Compose** (for running the database)
- **[golang-migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)** (CLI for database migrations)
- **[sqlc](https://docs.sqlc.dev/en/latest/overview/install.html)** (CLI for generating Go code from SQL)

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/KhanSufiyanMirza/evm-indexer-go.git
cd evm-indexer-go
```

### 2. Environment Setup

Ensure you have a `.env.local` file in the root directory with the necessary configuration. Example:

```env
RDB_MIGRATION_URL=db/migrations
RDB_HOST=localhost
RDB_PORT=5432
RDB_USER=evm_indexer
RDB_PASSWD=strongpassword
RDB_DB_NAME=evm_indexer_go
APP_NAME=evm-indexer-go
RPC_URL=https://rpc.flashbots.net
START_BLOCK=24347029
# Optional: run continuously (poll for new blocks instead of exiting after one pass)
# CONTINUOUS=true
# Optional: poll interval in continuous mode (default 12s, ~1 Ethereum block)
# BLOCK_POLL_INTERVAL=12s
# Optional: port for Prometheus metrics (default 9090)
# METRICS_PORT=9090
# Optional: ingestion block depth (default 12)
# INGESTION_BLOCK_DEPTH=12
# Optional: safe block depth (default 12)
# SAFE_BLOCK_DEPTH=12
```

### 3. Start Infrastructure

Use the Makefile to start the PostgreSQL database container:

```bash
make up
```

### 4. Run Migrations

Initialize the database schema:

```bash
make migrate-up
```

### 5. Run the Application

Start the indexer:

```bash
make run
```

With `CONTINUOUS=true` the indexer keeps running after the initial catch-up and polls for new blocks (every `BLOCK_POLL_INTERVAL`, default 12s), so the database stays near real-time.

## Development Commands

The project includes a `makefile` to simplify common development tasks. Run `make help` to see all available commands.

| Command | Description |
| :--- | :--- |
| `make up` | Start Docker services (Postgres) in background |
| `make down` | Stop and remove Docker services |
| `make logs` | View logs from Docker services |
| `make migrate-up` | Apply all up database migrations |
| `make migrate-down` | Rollback the last database migration |
| `make migrate-create` | Create a new migration file pair (interactive) |
| `make sqlc` | Generate Go code from SQL queries using `sqlc` |
| `make build` | Compile the application binary |
| `make test` | Run all unit tests |
| `make run` | Run the application locally |

## Project Structure

- `cmd/`: Application entrypoints
- `db/`: Database migrations and queries
- `internal/`: Private application code
- `docker-compose.yaml`: Infrastructure definition
- `makefile`: Task automation
## Observability & Metrics

The indexer features comprehensive, production-ready observability:
- **Prometheus Metrics:** Exposed at `http://localhost:9090/metrics` (configurable via `METRICS_PORT`). Tracks `blocks_processed_total`, `rpc_errors_total` (by type), `reorg_detected_total`, `lag_events_total`, and `block_processing_duration_seconds`.
- **Active Lag Detection:** Computes the lag between the chain tip and the last processed block. If lag exceeds `SAFE_BLOCK_DEPTH * 2`, it logs an `ALERT: High Lag Detected` event.
- **Structured Error Classification:** Distinguishes between `rpc_retry` (transient timeouts or rate limits) and `db_fatal` or `rpc_fatal` (critical failures) using structured logging.
- **Graceful Shutdown & Data Idempotency:** Safely handles SIGINT/SIGTERM, finalizing current blocks, and prevents duplicate data using PostgreSQL `ON CONFLICT` patterns.

## FAQs
### How logs are fetched?
- Block-based range
- Not “latest”
- One block at a time (for correctness)
### why log_index matters?
- multiple logs per txn and to uniquely identify txn
- to prevent duplicate logs and overwriting
- enables idempotency

### What this indexer does?
- Fetches blocks from the Ethereum node
- Parses logs from the blocks
- Stores the ERC20 Transfer logs in the database
### How resume works?
- Fetches the last processed block from the database
- Starts from the next block
- Processes all blocks
### How Idempotency is achieved?
- ON CONFLICT DO NOTHING (for save block and save log)
### What failures are handled?
- RPC timeout
- Process crash mid-block (restart from last processed block)
- Duplicate block processing
- Partial log insert
- DB Errors (except constraint violation errors)
- Network errors
- re-orgs
### Rollback strategy for reorg?
We use a soft-delete model.
`is_canonical` flag is used to identify if the block is canonical or not.

### What triggers an alert?
- **High Lag:** Exceeding `SAFE_BLOCK_DEPTH * 2` (indicates the indexer is falling behind).
- **Deep Re-orgs:** Re-organization depth greater than 3 blocks.
- **Process Down / DB Fatal Errors:** Critical issues like missing schemas or corrupted states.
- **Continuous RPC Failures:** When the configured node goes entirely offline or rejects requests consistently.
