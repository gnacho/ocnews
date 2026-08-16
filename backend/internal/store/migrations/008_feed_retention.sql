-- Retención por feed (extensión propia): override del global OCNEWS_RETENTION_DAYS.
-- 0 = usar la retención global; NULL no se usa (preferimos 0 por claridad).

ALTER TABLE feeds ADD COLUMN retention_days INTEGER NOT NULL DEFAULT 0;
