## Can a sucessful txn emit no logs?
Yes, case like simple ETH transfer.
if txn have emit event call then only txn emit logs.
Contract function that doesn’t emit events it will emit no logs at all.
## Can a failed txn emit logs?
No, if transaction is reverted then all logs will be rolledback,and in txn receipt status will be 0 (false).

## Why Infra teams trust logs more than txn input?
Infra systems trust logs because they reflect executed state, not user intent.
- Logs = confirmed fact
## Why are block-range queries safer than “latest”?
Because:
	•	they are deterministic
	•	they can be retried
	•	they survive re-orgs
	•	they allow replay
## Why can large ranges fail?
Because:
	•	RPC limits
	•	payload size
	•	node performance
	•	provider throttling
## What happens if RPC returns partial logs?
Worst-case scenario:
	•	you mark block as processed
	•	missing logs never get indexed
	•	balances / payouts become wrong
	•	bug appears weeks later
## What does `exactly once` mean here?
- No duplicates
- No partials inserts
- idempotent (The system may attempt to process an event many times,
but the final state reflects the event only once.)
## Why block-based range?
- Block-based range is deterministic ()
- Block-based range can be retried
- Block-based range survives re-orgs
- Block-based range allows replay
## Where does idempotency truly live?
- Idempotency lives in the database constraints + deterministic keys
- For logs, it lives in the (txn_hash + log_index)
- For Blocks, it lives in the (block_number + block_hash)
## What breaks if logs are delayed?
Reality: logs are NOT real-time
Logs can be delayed because:
	•	RPC node is behind
	•	Network hiccups
	•	Your indexer lags
	•	Re-org causes rollback
	•	Finality depth not reached

Examples: 
- balance update
User deposits tokens.
Your system:
	•	Immediately credits user balance
	•	Allows withdrawal
	•	Log later disappears due to re-org

(You paid money that never existed)

- order fulfillment
	•	Payment log seen
	•	Item shipped
	•	Log delayed or rolled back

(Inventory & money mismatch)
Logs tell you:
	•	“This probably happened”
	•	Not “This definitely happened”

## Why business logic should wait for finality?
Key idea is
Ethereum can change its mind until finality.
A block can:
	•	Exist
	•	Emit logs
	•	Be replaced
	•	Disappear forever
Before finality:
	•	Logs are probabilistic
After finality:
	•	Logs are facts

## Why do we need finality depth?

Finality depth is the number of blocks after which a block is considered immutable.

In Ethereum, a block is considered finalized after 2 epochs (64 blocks) have been added after it.

This means that a block is considered immutable after 2 epochs (64 blocks) have been added after it.

## What is the finality depth of Ethereum?

The finality depth of Ethereum is after 2 epochs (2 epochs = 64 slots ≈ ~12.8 minutes).
maths:
	•	1 epoch = 32 slots
	•	1 slot ≈ 12 seconds
	•	At most 1 block per slot

## Why do we need finality depth?

Finality depth is needed to ensure that blocks are immutable and cannot be changed.