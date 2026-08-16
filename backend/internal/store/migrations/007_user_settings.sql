-- Ajustes por usuario persistidos (extensión propia; la spec v1.3 no los define):
-- tabla clave-valor por usuario para preferencias de la app (tema lector,
-- intervalo de refresco, idioma UI, etc.). language vive en users.*; esto
-- cubre el resto sin migrar columnas por cada ajuste.

CREATE TABLE IF NOT EXISTS user_settings (
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  key     TEXT NOT NULL,
  value   TEXT NOT NULL,
  PRIMARY KEY (user_id, key)
);
