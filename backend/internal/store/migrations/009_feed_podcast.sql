-- Detección de podcasts: un feed con items que traen enclosures de audio/vídeo
-- se marca is_podcast=1 para listarlo en la vista Podcasts de la extensión.

ALTER TABLE feeds ADD COLUMN is_podcast INTEGER NOT NULL DEFAULT 0;
