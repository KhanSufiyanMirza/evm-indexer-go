## what makes a block unique?
it's Block Hash + block Numer
## what data is required to detect reorg?
block number and block hash 
## what must never be duplicated?
block hash
## why block_number alone is not enough ?
block number can be duplicated if reorg happens,to detect reorg block number alone is not enough.
## why overwriting is dengerous ?
if we overwrite block then possibly we are making our off chain data inconsisten without 
considering reorg situation.
why we got the same block_number again?
is it having same hash?
## where did resume logic live?
based on last processed block in db we will resume,
not based on app memory or last fetched block from RPC.
## what state did you trust?
db state we will trust not app or RPC.

# Week 2 - Day 5 Reflection

## ❓ What does “exactly once” mean here?

In the context of blockchain infrastructure and distributed systems, "exactly once" is a semantic guarantee about the **result**, not the execution. It means that even if a block is processed multiple times (due to retries, crashes, or race conditions), the final state of our database remains correct and consistent, effectively reflecting a single successful processing. It shifts the focus from "preventing duplicates" to "making duplicates harmless" (idempotency).

## ❓ What happens if the same block is processed twice?

If the same block is processed twice in our system:
1.  **Block Fetching**: The block data is fetched again from the RPC provider.
2.  **Database Insertion**: The system attempts to insert the block into the `blocks` table.
3.  **Conflict Resolution**: The database detects a conflict on the unique constraint `(number, hash)`.
4.  **Graceful Handling**: The SQL query uses `ON CONFLICT DO NOTHING`, and the application logic interprets the resulting `pgx.ErrNoRows` (or affected rows = 0) as a success case, ensuring no error is returned and no panic occurs.
5.  **Data Integrity**: The existing data remains untouched and uncorrupted.
6.  **Logging**: The system logs a message indicating the block already exists, providing visibility without alarm.

