-- WebSub (PubSubHubbub) (#44): suscripción a hubs para recibir items en
-- tiempo real. Un feed con rel=hub puede empujar contenido al callback.

CREATE TABLE IF NOT EXISTS websub (
  feed_id     INTEGER PRIMARY KEY REFERENCES feeds(id) ON DELETE CASCADE,
  hub         TEXT NOT NULL,
  topic       TEXT NOT NULL,
  secret      TEXT NOT NULL DEFAULT '',
  callback    TEXT NOT NULL DEFAULT '',
  lease_until INTEGER NOT NULL DEFAULT 0,
  status      TEXT NOT NULL DEFAULT 'pending'
);
