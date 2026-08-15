-- F2: identidad OpenCloud canónica (uuid del IDM) para unificar
-- usuarios shadow creados vía Basic (app token) y vía Bearer (sesión web).

ALTER TABLE users ADD COLUMN oc_id TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX idx_users_oc_id ON users (oc_id) WHERE oc_id != '';
