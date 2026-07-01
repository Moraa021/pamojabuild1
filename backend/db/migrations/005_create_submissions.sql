CREATE TABLE task_submissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_slug VARCHAR(255) REFERENCES tasks(slug),
    volunteer_id INTEGER REFERENCES users(id),
    description TEXT,
    evidence_urls TEXT DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'submitted',
    submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMP
);
