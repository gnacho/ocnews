package cred

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRoundtrip(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	enc, err := c.Encrypt("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if enc == "s3cret" || enc == "" {
		t.Fatalf("el cifrado no puede ser el plano ni vacío: %q", enc)
	}
	dec, err := c.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != "s3cret" {
		t.Fatalf("roundtrip: %q", dec)
	}
	// dos cifrados del mismo texto difieren (nonce aleatorio)
	enc2, _ := c.Encrypt("s3cret")
	if enc == enc2 {
		t.Fatal("mismo texto, mismo cifrado: nonce reutilizado")
	}
	// vacío permanece vacío
	if e, _ := c.Encrypt(""); e != "" {
		t.Fatalf("vacío cifrado: %q", e)
	}
	if d, _ := c.Decrypt(""); d != "" {
		t.Fatalf("vacío descifrado: %q", d)
	}
}

func TestKeyPersists(t *testing.T) {
	dir := t.TempDir()
	c1, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	enc, _ := c1.Encrypt("abc")
	// segunda carga: misma clave → descifra lo anterior
	c2, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dec, err := c2.Decrypt(enc); err != nil || dec != "abc" {
		t.Fatalf("clave no persistida: %v %q", err, dec)
	}
	info, err := os.Stat(filepath.Join(dir, "feedsecret"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permisos feedsecret: %o", info.Mode().Perm())
	}
}

func TestDecryptWrongKey(t *testing.T) {
	c1, _ := Load(t.TempDir())
	c2, _ := Load(t.TempDir())
	enc, _ := c1.Encrypt("abc")
	if _, err := c2.Decrypt(enc); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("esperaba ErrDecrypt: %v", err)
	}
	if _, err := c1.Decrypt("no-es-base64!!!"); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("base64 inválido: %v", err)
	}
}
