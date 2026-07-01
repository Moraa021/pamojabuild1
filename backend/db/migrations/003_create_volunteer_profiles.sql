CREATE TABLE volunteer_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id),
    bio TEXT,
    skills JSONB DEFAULT '[]',
    lightning_address VARCHAR(255),
    onchain_address VARCHAR(255),
    reputation_score INT DEFAULT 0,
    tier VARCHAR(50) DEFAULT 'New',
    completed_tasks INT DEFAULT 0,
    total_earned_sats BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
