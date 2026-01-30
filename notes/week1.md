## 3 things Ethereum does not guarantee
- No guarantee of Immediate txn inclusion
- No guarantee against chain reorg
- No guarantee again MEV (Maximal extractable value)
    - validator can reorder your txn and insert their own txn before/after
## Real risk for infra system
- mempool Inconsistencies
- Archive node dependencies
    - Your infra break when node prune historical data.
- MEV and Frontrunning can happen with your users.
- Chain reorg breaking state assumption
## What is reorg?
it's temporary fork chain situation where chain which have logenst and weighted fork get accepted as cannonical chain.
happens due to network latency,Inconsistencies between client,and fork choices rules to accumuate new block and sometime it happens because client are trying to create MEV opportunity.

## why infra engineer are more than app developer?
app developer deals with
- smart contract
- backend system (apis)
- frontend

and consider or assumption they make:
- chain is stable
- events are final
- RPC's are reliable
infra engineer deals with
- chain reorg
- partial failure
- RPC Inconsistencies
- event duplication
## with which field we can detect chain reorg
with blockNumber and blockHash we can figure out/detect chain reorg.
consider you have map[blockNumber]blockHash.
when you got new block N then check weather N-1 block hash is equal to N parent block hash if not hten it's chain reorg.
## which are imp fields
BlockNumber,BlockHash,ParentHash(optional because you can any time track previous block number and its Hash),Timestamp,Transactions
## if RPC Lies, what happens?
- data Inconsistency
- MEV (Frontrunning via RPC manipulation)
- you can miss important events
- off chain database compromised
## if block disappres, what happens?
- we can process non finalize event which is not present on main chain
- loss of money
- Application breaks
- onchain system breaks
- smart contract interactions breaks
## where should live - chain or database?
The Fundamental Principle: Chain as Source of Truth
