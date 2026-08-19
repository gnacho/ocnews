-- Extracción completa con selector CSS por feed (#39): cuando el sitio se
-- resiste a readability, el selector del feed se usa como objetivo antes de
-- rendirse.

ALTER TABLE feeds ADD COLUMN scraper_selector TEXT NOT NULL DEFAULT '';
