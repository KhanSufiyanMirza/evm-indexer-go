## Where does truth live after restart?
After a restart, the truth of the chain is stored in the database.
## DB or memory?
DB
## What happens if last block was half-processed?
we will fetch the last successfully processed block from the database and process it again from the next block which eventually leads to idempotency and process every remaining block and half processed block.

## Which errors are retryable?
1. RPC errors
2. DB errors (except constraint violation errors)
3. Network errors

## Which error must stop the process?
1. Invalid RPC URL
2. Invalid DB conf or URL
3. Constraint violation errors
