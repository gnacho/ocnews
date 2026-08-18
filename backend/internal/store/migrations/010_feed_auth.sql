-- 010: credenciales HTTP Basic por feed (auth_pass cifrada AES-GCM, ver internal/cred).
ALTER TABLE feeds ADD COLUMN auth_user TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN auth_pass TEXT NOT NULL DEFAULT '';
