-- F1: favicon por hash de URL + racha sin items nuevos (scheduler adaptativo)

ALTER TABLE feeds ADD COLUMN url_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN no_new_streak INTEGER NOT NULL DEFAULT 0;
CREATE INDEX idx_feeds_url_hash ON feeds (url_hash);
CREATE INDEX idx_feeds_next_update ON feeds (next_update);
