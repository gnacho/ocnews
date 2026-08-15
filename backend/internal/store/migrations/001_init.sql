-- ocnews schema inicial (F0)
-- Multiusuario: feeds/folders/items cuelgan de user_id.
-- items.guid_hash = md5(guid); unicidad por (feed_id, guid_hash).

CREATE TABLE users (
  id            INTEGER PRIMARY KEY,
  username      TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  display_name  TEXT NOT NULL DEFAULT '',
  role          TEXT NOT NULL DEFAULT 'user',
  language      TEXT NOT NULL DEFAULT 'auto' CHECK (language IN ('auto','es','en')),
  created_at    INTEGER NOT NULL,
  last_login_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE folders (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name    TEXT NOT NULL,
  UNIQUE (user_id, name)
);

CREATE TABLE feeds (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id            INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  folder_id          INTEGER REFERENCES folders(id) ON DELETE SET NULL,
  url                TEXT NOT NULL,
  link               TEXT NOT NULL DEFAULT '',
  title              TEXT NOT NULL DEFAULT '',
  favicon           TEXT NOT NULL DEFAULT '',
  added              INTEGER NOT NULL,
  next_update        INTEGER NOT NULL DEFAULT 0,
  ordering           INTEGER NOT NULL DEFAULT 0,
  pinned             INTEGER NOT NULL DEFAULT 0,
  update_error_count INTEGER NOT NULL DEFAULT 0,
  last_update_error  TEXT NOT NULL DEFAULT '',
  UNIQUE (user_id, url)
);

CREATE TABLE items (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id          INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  feed_id          INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
  guid             TEXT NOT NULL,
  guid_hash        TEXT NOT NULL,
  url              TEXT NOT NULL DEFAULT '',
  title            TEXT NOT NULL DEFAULT '',
  author           TEXT NOT NULL DEFAULT '',
  pub_date         INTEGER NOT NULL DEFAULT 0,
  body             TEXT NOT NULL DEFAULT '',
  enclosure_mime   TEXT,
  enclosure_link   TEXT,
  media_thumbnail  TEXT,
  media_description TEXT,
  fingerprint      TEXT NOT NULL DEFAULT '',
  unread           INTEGER NOT NULL DEFAULT 1,
  starred          INTEGER NOT NULL DEFAULT 0,
  last_modified    INTEGER NOT NULL,
  UNIQUE (feed_id, guid_hash)
);

CREATE INDEX idx_items_user_modified ON items (user_id, last_modified);
CREATE INDEX idx_items_user_id       ON items (user_id, id);
CREATE INDEX idx_items_feed          ON items (feed_id, id);
CREATE INDEX idx_feeds_user          ON feeds (user_id);
CREATE INDEX idx_folders_user        ON folders (user_id);
