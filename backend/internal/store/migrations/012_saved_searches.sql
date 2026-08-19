-- Búsquedas guardadas del usuario: una query persistida que se muestra en el
-- sidebar como un feed virtual (estilo FreshRSS user queries / tt-rss
-- generated feeds). No es una suscripción real.

CREATE TABLE IF NOT EXISTS saved_searches (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  query      TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_saved_searches_user ON saved_searches(user_id);
