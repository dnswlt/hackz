-- Recreate tables as follows:
--
-- psql -U loadtest_owner -d loadtest -f db/schema.sql
--

DROP TABLE IF EXISTS items;

CREATE TABLE items (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    payload TEXT NOT NULL,
    create_time TIMESTAMPTZ NOT NULL DEFAULT now()
);
