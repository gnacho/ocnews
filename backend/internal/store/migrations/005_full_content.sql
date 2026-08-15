-- Detección de feeds que ya sirven contenido completo: si algún item
-- nuevo supera el umbral de texto, el feed se marca y el cliente no
-- ofrece la extracción completa (redundante).

ALTER TABLE feeds ADD COLUMN full_content INTEGER NOT NULL DEFAULT 0;
