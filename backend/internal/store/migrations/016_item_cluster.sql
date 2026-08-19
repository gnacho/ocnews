-- Clustering de la misma noticia en varios feeds (#42): cluster_key normaliza
-- título+cuerpo para agrupar duplicados entre suscripciones.

ALTER TABLE items ADD COLUMN cluster_key TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_items_cluster ON items(user_id, cluster_key);
