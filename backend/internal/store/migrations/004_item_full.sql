-- F1: cuerpo completo extraído del artículo original (readability);
-- los feeds suelen traer solo el resumen.

CREATE TABLE item_full (
  item_id    INTEGER PRIMARY KEY REFERENCES items(id) ON DELETE CASCADE,
  body       TEXT NOT NULL,
  fetched_at INTEGER NOT NULL
);
