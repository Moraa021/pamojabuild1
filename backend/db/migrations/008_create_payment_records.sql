CREATE TABLE IF NOT EXISTS volunteer_payments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_slug VARCHAR(255) REFERENCES tasks(slug),
    volunteer_id INTEGER REFERENCES users(id),
    amount_sats INTEGER NOT NULL,
    payment_method VARCHAR(50),
    status VARCHAR(50) DEFAULT 'pending',
    transaction_hash VARCHAR(255),
    paid_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS lightning_invoices (
    payment_request TEXT NOT NULL,
    payment_hash VARCHAR(255) PRIMARY KEY,
    amount_sats INTEGER NOT NULL,
    task_slug VARCHAR(255) REFERENCES tasks(slug),
    settled INTEGER DEFAULT 0,
    settled_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS payout_signatures (
    task_slug VARCHAR(255) REFERENCES tasks(slug),
    trustee_public_key_hex VARCHAR(512) NOT NULL,
    l1_signature_fragment TEXT,
    l2_web_crypto_signature TEXT,
    PRIMARY KEY (task_slug, trustee_public_key_hex)
);
