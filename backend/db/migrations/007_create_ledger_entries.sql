CREATE TABLE IF NOT EXISTS ledger_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_slug VARCHAR(255) REFERENCES tasks(slug),
    entry_type VARCHAR(50) NOT NULL,
    amount_sats INTEGER NOT NULL,
    reference_id VARCHAR(255),
    previous_hash BLOB,
    row_hmac BLOB NOT NULL
);
