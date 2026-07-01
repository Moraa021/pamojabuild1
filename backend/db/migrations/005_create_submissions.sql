CREATE TABLE task_submissions (
    id BIGSERIAL PRIMARY KEY,
    task_slug VARCHAR(255) REFERENCES tasks(slug),
    volunteer_id BIGINT REFERENCES users(id),
    description TEXT,
    evidence_urls JSONB DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'submitted',
    submitted_at TIMESTAMP DEFAULT NOW(),
    reviewed_at TIMESTAMP
);
