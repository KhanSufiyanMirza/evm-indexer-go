CREATE TABLE erc20_transfers (
    tx_hash TEXT NOT NULL,
    log_index INTEGER NOT NULL,
    block_number BIGINT NOT NULL,
    from_address TEXT NOT NULL,
    to_address TEXT NOT NULL,
    value NUMERIC NOT NULL,
    PRIMARY KEY (tx_hash, log_index)
);