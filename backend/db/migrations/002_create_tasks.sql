CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug VARCHAR(255) UNIQUE NOT NULL,
    creator_id INTEGER REFERENCES users(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    region VARCHAR(100),
    location_detail VARCHAR(255),
    status VARCHAR(50) DEFAULT 'open',
    financial_state VARCHAR(50) DEFAULT 'ACTIVE',
    goal_sats INTEGER DEFAULT 0,
    max_volunteers INTEGER DEFAULT 1,
    volunteer_mode VARCHAR(50) DEFAULT 'open',
    image_path VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
