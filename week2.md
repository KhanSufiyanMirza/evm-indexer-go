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
