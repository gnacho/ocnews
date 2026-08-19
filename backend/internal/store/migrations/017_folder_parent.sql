-- Subcarpetas (#41): parent_id permite anidar carpetas (un nivel). Al borrar
-- una carpeta padre, sus subcarpetas se mueven a la raíz (la API lo hace
-- antes del borrado; CASCADE es la red de seguridad).

ALTER TABLE folders ADD COLUMN parent_id INTEGER REFERENCES folders(id) ON DELETE SET NULL;
