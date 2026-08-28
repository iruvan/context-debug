-- Runs automatically on first start of the postgres container
-- (docker-entrypoint-initdb.d). Matches the query in dependency.go.

CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO users (name, email) VALUES
    ('Ada Lovelace',   'ada@example.com'),
    ('Alan Turing',    'alan@example.com'),
    ('Grace Hopper',   'grace@example.com')
ON CONFLICT (email) DO NOTHING;
