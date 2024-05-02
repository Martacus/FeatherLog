CREATE TABLE IF NOT EXISTS "user"
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username   VARCHAR(255) UNIQUE,
    email      VARCHAR(255) UNIQUE,
    password   BYTEA NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);