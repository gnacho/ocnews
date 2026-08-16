-- Filtrado por keywords por feed (News 28.4.0): feed_filter guarda las
-- keywords (title/body/url) que ocultan los artículos que casan; items
-- gana la columna filtered=1 para los que el filtro descarta.

CREATE TABLE IF NOT EXISTS feed_filter (
  feed_id        INTEGER PRIMARY KEY,
  title_keywords TEXT NOT NULL DEFAULT '',
  body_keywords  TEXT NOT NULL DEFAULT '',
  url_keywords   TEXT NOT NULL DEFAULT ''
);

ALTER TABLE items ADD COLUMN filtered INTEGER NOT NULL DEFAULT 0;
