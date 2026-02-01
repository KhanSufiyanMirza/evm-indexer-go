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