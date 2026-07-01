CREATE TABLE ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    task_slug VARCHAR(255) REFERENCES tasks(slug),
    entry_type VARCHAR(50) NOT NULL,
    amount_sats BIGINT NOT NULL,
    reference_id VARCHAR(255),
    previous_hash BYTEA,
    row_hmac BYTEA NOT NULL
);
