## If this service runs in prod, what would I want to know at 3 AM?
minimum requirement signals:
- Liveness
	- Is the service running?

- Progress
	- What block height is currently processed?
	- How far behind chain tip are we?

- Performance
	- Blocks processed per second
	- RPC latency

- Errors
	- RPC failures
	- DB failures
	- Re-org count

## Day 5 — Alerting Mindset (Even If Local)

**What would wake me up at 3 AM?**
- **Process Down / DB Fatal Errors:** The indexer itself crashed and restarting won't fix it (e.g. schema mismatch).
- **Lag exceeding `SAFE_BLOCK_DEPTH * 2` constantly:** The indexer is completely stalled or processing far too slowly to catch up. Alert log: `ALERT: High Lag Detected`.
- **Deep Re-orgs (> 3 blocks):** Especially on fast-finality chains, a 3+ block reorg might indicate a chain fork, node split, or an attack. Alert log: `ALERT: Deep Reorg Detected`.
- **Continuous `rpc_fatal` errors:** This indicates a complete failure to connect to the node, possibly IP ban, node crash, or networking issue.

**What can wait till morning?**
- **Intermittent `rpc_retry` warnings:** As long as we recover through backoff logic and progress continues, this is expected noise.
- **Short Re-orgs (1 or 2 blocks):** Normal blockchain operation, the application handles this gracefully.
- **Small Lag spikes:** If the lag jumps slightly but catches up within a few polling intervals, it's just normal variation in block time or temporary RPC slowness.
