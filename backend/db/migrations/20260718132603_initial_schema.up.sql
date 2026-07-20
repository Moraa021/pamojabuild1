CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'volunteer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tasks (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(255) NOT NULL UNIQUE,
    creator_id BIGINT REFERENCES users(id) ON DELETE RESTRICT,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category VARCHAR(100) NOT NULL DEFAULT '',
    region VARCHAR(100) NOT NULL DEFAULT '',
    location_detail VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'open',
    financial_state VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    goal_sats BIGINT NOT NULL DEFAULT 0 CHECK (goal_sats >= 0),
    max_volunteers BIGINT NOT NULL DEFAULT 1 CHECK (max_volunteers >= 0),
    volunteer_mode VARCHAR(50) NOT NULL DEFAULT 'open',
    image_path VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tasks_filters
    ON tasks (status, category, region, created_at DESC);

CREATE TABLE volunteer_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bio TEXT NOT NULL DEFAULT '',
    skills JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(skills) = 'array'),
    lightning_address VARCHAR(255) NOT NULL DEFAULT '',
    onchain_address VARCHAR(255) NOT NULL DEFAULT '',
    reputation_score INTEGER NOT NULL DEFAULT 0,
    tier VARCHAR(50) NOT NULL DEFAULT 'New',
    completed_tasks INTEGER NOT NULL DEFAULT 0 CHECK (completed_tasks >= 0),
    total_earned_sats BIGINT NOT NULL DEFAULT 0 CHECK (total_earned_sats >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_applications (
    id BIGSERIAL PRIMARY KEY,
    task_slug VARCHAR(255) NOT NULL REFERENCES tasks(slug) ON DELETE CASCADE,
    volunteer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    message TEXT NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMPTZ,
    CONSTRAINT uq_task_application_volunteer UNIQUE (task_slug, volunteer_id)
);

CREATE INDEX idx_task_applications_volunteer
    ON task_applications (volunteer_id, applied_at DESC);

CREATE TABLE task_submissions (
    id BIGSERIAL PRIMARY KEY,
    task_slug VARCHAR(255) NOT NULL REFERENCES tasks(slug) ON DELETE CASCADE,
    volunteer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    description TEXT NOT NULL,
    evidence_urls JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(evidence_urls) = 'array'),
    status VARCHAR(50) NOT NULL DEFAULT 'submitted',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMPTZ
);

CREATE INDEX idx_task_submissions_volunteer
    ON task_submissions (volunteer_id, submitted_at DESC);

CREATE TABLE trustee_keys (
    task_slug VARCHAR(255) NOT NULL REFERENCES tasks(slug) ON DELETE CASCADE,
    trustee_index INTEGER NOT NULL CHECK (trustee_index BETWEEN 0 AND 4),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    xpub VARCHAR(255) NOT NULL,
    web_crypto_pubkey_hex VARCHAR(512) NOT NULL,
    PRIMARY KEY (task_slug, trustee_index),
    CONSTRAINT uq_task_trustee_user UNIQUE (task_slug, user_id)
);

CREATE TABLE ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    task_slug VARCHAR(255) NOT NULL REFERENCES tasks(slug) ON DELETE RESTRICT,
    entry_type VARCHAR(50) NOT NULL,
    amount_sats BIGINT NOT NULL,
    reference_id VARCHAR(255) NOT NULL DEFAULT '',
    previous_hash BYTEA,
    row_hmac BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ledger_entries_task_order
    ON ledger_entries (task_slug, id);

CREATE TABLE volunteer_payments (
    id BIGSERIAL PRIMARY KEY,
    task_slug VARCHAR(255) NOT NULL REFERENCES tasks(slug) ON DELETE RESTRICT,
    volunteer_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    amount_sats BIGINT NOT NULL CHECK (amount_sats >= 0),
    payment_method VARCHAR(50) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    transaction_hash VARCHAR(255) NOT NULL DEFAULT '',
    paid_at TIMESTAMPTZ
);

CREATE INDEX idx_volunteer_payments_history
    ON volunteer_payments (volunteer_id, paid_at DESC NULLS LAST);

CREATE TABLE lightning_invoices (
    payment_request TEXT NOT NULL,
    payment_hash VARCHAR(64) PRIMARY KEY,
    amount_sats BIGINT NOT NULL CHECK (amount_sats > 0),
    task_slug VARCHAR(255) NOT NULL REFERENCES tasks(slug) ON DELETE RESTRICT,
    status VARCHAR(32) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'settled', 'expired')),
    settled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ,
    settled_at TIMESTAMPTZ,
    add_index BIGINT NOT NULL DEFAULT 0 CHECK (add_index >= 0),
    settle_index BIGINT NOT NULL DEFAULT 0 CHECK (settle_index >= 0)
);

CREATE INDEX idx_lightning_invoices_status
    ON lightning_invoices (status);

CREATE INDEX idx_lightning_invoices_settle_index
    ON lightning_invoices (settle_index);

CREATE TABLE lightning_sync_state (
    key VARCHAR(128) PRIMARY KEY,
    value_integer BIGINT NOT NULL CHECK (value_integer >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE payout_signatures (
    task_slug VARCHAR(255) NOT NULL REFERENCES tasks(slug) ON DELETE CASCADE,
    trustee_public_key_hex VARCHAR(512) NOT NULL,
    l1_signature_fragment TEXT NOT NULL DEFAULT '',
    l2_web_crypto_signature TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (task_slug, trustee_public_key_hex)
);
