CREATE TABLE IF NOT EXISTS task_applications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_slug VARCHAR(255) REFERENCES tasks(slug),
    volunteer_id INTEGER REFERENCES users(id),
    message TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMP
);
