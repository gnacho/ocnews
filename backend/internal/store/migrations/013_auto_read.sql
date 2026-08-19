-- Auto-marcar como leído al llegar: reglas por usuario (feed_id 0 = todos)
-- que marcan unread=0 los items nuevos cuyo título casa con el patrón regex.

CREATE TABLE IF NOT EXISTS auto_read (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  feed_id       INTEGER NOT NULL DEFAULT 0,
  title_pattern TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auto_read_user ON auto_read(user_id);
