package config

import (
	"strings"
	"testing"
)

// La URL de OpenCloud se lee con dos nombres: la ortografía correcta
// (OCNEWS_OPENCLOUD_URL) gana, y el alias histórico con errata
// (OCNEWS_OPENCOLOUD_URL) se mantiene como fallback (issue #58).
func TestLoadOpenCloudURLSpelling(t *testing.T) {
	t.Run("spelling correcta", func(t *testing.T) {
		t.Setenv("OCNEWS_AUTH_MODE", "opencloud")
		t.Setenv("OCNEWS_OPENCLOUD_URL", "https://cloud.example.com")
		t.Setenv("OCNEWS_OPENCOLOUD_URL", "")
		c, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if c.OpenCloudURL != "https://cloud.example.com" {
			t.Fatalf("OpenCloudURL = %q", c.OpenCloudURL)
		}
	})

	t.Run("alias historico con errata", func(t *testing.T) {
		t.Setenv("OCNEWS_AUTH_MODE", "opencloud")
		t.Setenv("OCNEWS_OPENCLOUD_URL", "")
		t.Setenv("OCNEWS_OPENCOLOUD_URL", "https://legacy.example.com")
		c, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if c.OpenCloudURL != "https://legacy.example.com" {
			t.Fatalf("OpenCloudURL = %q", c.OpenCloudURL)
		}
	})

	t.Run("la ortografia correcta gana si ambas estan", func(t *testing.T) {
		t.Setenv("OCNEWS_AUTH_MODE", "opencloud")
		t.Setenv("OCNEWS_OPENCLOUD_URL", "https://cloud.example.com")
		t.Setenv("OCNEWS_OPENCOLOUD_URL", "https://legacy.example.com")
		c, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if c.OpenCloudURL != "https://cloud.example.com" {
			t.Fatalf("OpenCloudURL = %q", c.OpenCloudURL)
		}
	})

	t.Run("sin URL en modo opencloud falla y nombra la variable", func(t *testing.T) {
		t.Setenv("OCNEWS_AUTH_MODE", "opencloud")
		t.Setenv("OCNEWS_OPENCLOUD_URL", "")
		t.Setenv("OCNEWS_OPENCOLOUD_URL", "")
		_, err := Load()
		if err == nil {
			t.Fatal("esperaba error sin OCNEWS_OPENCLOUD_URL")
		}
		if !strings.Contains(err.Error(), "OCNEWS_OPENCLOUD_URL") {
			t.Fatalf("el error debería nombrar OCNEWS_OPENCLOUD_URL: %v", err)
		}
	})
}
