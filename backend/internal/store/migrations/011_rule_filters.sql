-- Filtros avanzados por regex block/keep (estilo Miniflux): feed_rules por
-- feed y global_rules por usuario. Formato multilínea "Campo=regex" con
-- campos EntryTitle/EntryURL/EntryAuthor/EntryContent/EntryDate.

CREATE TABLE IF NOT EXISTS feed_rules (
  feed_id INTEGER PRIMARY KEY REFERENCES feeds(id) ON DELETE CASCADE,
  block   TEXT NOT NULL DEFAULT '',
  keep    TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS global_rules (
  user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  block   TEXT NOT NULL DEFAULT '',
  keep    TEXT NOT NULL DEFAULT ''
);
