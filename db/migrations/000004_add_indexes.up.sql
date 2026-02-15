CREATE INDEX IF NOT EXISTS idx_blocks_number_canonical ON blocks (number) WHERE is_canonical = TRUE;
CREATE INDEX IF NOT EXISTS idx_erc20_transfers_block_number ON erc20_transfers (block_number);
CREATE INDEX IF NOT EXISTS idx_erc20_transfers_token_address ON erc20_transfers (token_address);
