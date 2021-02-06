CREATE TABLE IF NOT EXISTS pairs (
	key         TEXT PRIMARY KEY,
	value       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS pairs_value ON pairs(value);

-- add more commands here