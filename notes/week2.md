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
