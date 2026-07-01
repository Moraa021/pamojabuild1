CREATE TABLE task_applications (
    id BIGSERIAL PRIMARY KEY,
    task_slug VARCHAR(255) REFERENCES tasks(slug),
    volunteer_id BIGINT REFERENCES users(id),
    message TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    applied_at TIMESTAMP DEFAULT NOW(),
    reviewed_at TIMESTAMP
);
