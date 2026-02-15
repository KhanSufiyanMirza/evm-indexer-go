-- Initialize the database schema
Create Table IF NOT EXISTS blocks (
    id SERIAL PRIMARY KEY,
    hash VARCHAR(66) NOT NULL,
    number BIGINT NOT NULL,
    parent_hash VARCHAR(66) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_canonical BOOLEAN DEFAULT TRUE,
    reorg_detected_at TIMESTAMP NULL,
    constraint unique_hash_number unique (hash, number)
);

