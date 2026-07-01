CREATE TABLE trustee_keys (
    task_slug VARCHAR(255) REFERENCES tasks(slug),
    trustee_index INT NOT NULL,
    user_id BIGINT REFERENCES users(id),
    xpub VARCHAR(255) NOT NULL,
    web_crypto_pubkey_hex VARCHAR(512) NOT NULL,
    PRIMARY KEY (task_slug, trustee_index)
);
