CREATE TABLE volunteer_profiles (
    user_id INTEGER PRIMARY KEY REFERENCES users(id),
    bio TEXT,
    skills TEXT DEFAULT '[]',
    lightning_address VARCHAR(255),
    onchain_address VARCHAR(255),
    reputation_score INTEGER DEFAULT 0,
    tier VARCHAR(50) DEFAULT 'New',
    completed_tasks INTEGER DEFAULT 0,
    total_earned_sats INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
