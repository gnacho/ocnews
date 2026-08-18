// Package cred: cifrado simétrico (AES-256-GCM) para credenciales de feeds
// en reposo. La clave vive en <dataDir>/feedsecret (32B, 0600, generada la
// primera vez), siguiendo el patrón de imgproxy (imgsecret). Quien tenga
// acceso de lectura al data dir puede descifrar: la defensa es el
// aislamiento de ficheros, no el secreto compartido.
package cred

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Cipher struct {
	key []byte
}

// Load lee (o genera) la clave en <dataDir>/feedsecret.
func Load(dataDir string) (*Cipher, error) {
	path := filepath.Join(dataDir, "feedsecret")
	key, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		key = make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("generar feedsecret: %w", err)
		}
		if err := os.WriteFile(path, key, 0o600); err != nil {
			return nil, fmt.Errorf("persistir feedsecret: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("leer feedsecret: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("feedsecret inválido: %d bytes (esperados 32)", len(key))
	}
	return &Cipher{key: key}, nil
}

// Encrypt cifra y devuelve base64(nonce | ciphertext).
func (c *Cipher) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	gcm, err := c.gcm()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

// ErrDecrypt: el dato no descifra con esta clave (clave rotada o corrupción).
var ErrDecrypt = errors.New("no se puede descifrar la credencial")

// Decrypt invierte Encrypt; "" permanece "".
func (c *Cipher) Decrypt(b64 string) (string, error) {
	if b64 == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", ErrDecrypt
	}
	gcm, err := c.gcm()
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", ErrDecrypt
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", ErrDecrypt
	}
	return string(plain), nil
}

func (c *Cipher) gcm() (cipher.AEAD, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
