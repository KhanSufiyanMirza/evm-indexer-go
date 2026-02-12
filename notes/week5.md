## What exactly is reorg?
reorg is a situation where the canonical chain of a blockchain is switched to a different chain due to a fork.
or when network replaces previously accepted blocks with a longer alternative chain (weighted). 
- It can happen due to 
    - network latency
    - miner collusion
    - MEV extraction
## What does ethereum guarantee? What Does It Not?
Guarantees:
- Ethereum eventually agrees on a single canonical chain on the network.  
- immutability of the chain after finality (Finality after enough confirmations)
- Transactions are never lost (they stay in mempool)

Does Not Guarantee:
- instant finality
- latest block is stable
- reorgs are rare
- Immutability of recent history
- Your transaction will be included in a specific block
- Your transaction will be executed exactly once (important)
- Block #N today will have the same hash tomorrow
- Your transaction will be included in a specific block

## Why reorg are usually shallow but still dangerous?
- Because even 1-block instability can corrupt financial logic.
- Your off chain logic can be corrupted by a 1-block reorg.
### Note:
 - PoS Ethereum: 1-2 blocks re-orgs happen daily.
 - 3+ blocks re-orgs are rare (a few per month).
 - 5 blocks almost always due to client bugs or MEV attacks.

 ## What does it mean if this comparison (Block N.ParentHash != Block N-1.Hash) fails?
 It means there is a reorg at block N.