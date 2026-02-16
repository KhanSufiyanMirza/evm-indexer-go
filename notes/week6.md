## What Is Probabilistic Finality?
Ethereum doesn't say:
- "This block is 100% permanent."
- Instead:
	- The deeper a block is,
	- The more expensive it becomes to reverse.

- After ~12 blocks:
	- Extremely unlikely to revert
	- Exchanges treat this as safe

That’s economic finality, not mathematical certainty.
## Why 1 confirmation is unsafe?
- A single block can be reverted
- This is called a reorg
- It happens when two miners find a block at the same time
- The network chooses the longest chain
- The shorter chain is discarded
- The transactions in the shorter chain are reverted
## Why Exchanges Wait 12+ Confirmations?
Because they handle:
	•	Real money
	•	Arbitrage bots
	•	Double-spend risk
	•	Chain instability

If Binance credited deposits at 1 confirmation:
Attackers would exploit re-orgs.

Infrastructure engineers design for:

Worst-case, not happy-case.