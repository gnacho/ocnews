// Package config carga y valida la configuración del backend ocnews.
// Todo llega por variables de entorno; validación fail-fast al arranque.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr         string        // OCNEWS_ADDR, default ":8094"
	DataDir      string        // OCNEWS_DATA_DIR, default "./data"
	FetchTimeout time.Duration // OCNEWS_FETCH_TIMEOUT, default 20s
	LogLevel     string        // OCNEWS_LOG_LEVEL, default "info"
	AuthUser     string        // AUTH_USER: bootstrap del primer usuario admin
	AuthPass     string        // AUTH_PASS: bootstrap del primer usuario admin

	FeedInterval time.Duration // OCNEWS_FEED_INTERVAL, default 15m (base del scheduler)
	MaxGap       time.Duration // OCNEWS_MAX_GAP, default 6h (tope adaptativo)
	Retention    time.Duration // OCNEWS_RETENTION_DAYS, default 90d; 0 = desactivada

	AuthMode       string // OCNEWS_AUTH_MODE: local (default) | opencloud
	OpenCloudURL   string // OCNEWS_OPENCOLOUD_URL: raíz del servidor OpenCloud (modo opencloud)

	NtfyURL   string // OCNEWS_NTFY_URL, default https://ntfy.sh (base de notificaciones)
	NtfyTopic string // OCNEWS_NTFY_TOPIC: topic global de ntfy (vacío = desactivado salvo per-usuario)
}

func Load() (*Config, error) {
	c := &Config{
		Addr:           env("OCNEWS_ADDR", ":8094"),
		DataDir:        env("OCNEWS_DATA_DIR", "./data"),
		FetchTimeout:   20 * time.Second,
		LogLevel:       env("OCNEWS_LOG_LEVEL", "info"),
		AuthUser:       os.Getenv("AUTH_USER"),
		AuthPass:       os.Getenv("AUTH_PASS"),
		FeedInterval:   15 * time.Minute,
		MaxGap:         6 * time.Hour,
		Retention:      90 * 24 * time.Hour,
		AuthMode:       env("OCNEWS_AUTH_MODE", "local"),
		OpenCloudURL:   os.Getenv("OCNEWS_OPENCOLOUD_URL"),
		NtfyURL:        env("OCNEWS_NTFY_URL", "https://ntfy.sh"),
		NtfyTopic:      os.Getenv("OCNEWS_NTFY_TOPIC"),
	}
	for _, e := range []struct {
		key string
		dst *time.Duration
	}{
		{"OCNEWS_FETCH_TIMEOUT", &c.FetchTimeout},
		{"OCNEWS_FEED_INTERVAL", &c.FeedInterval},
		{"OCNEWS_MAX_GAP", &c.MaxGap},
	} {
		if v := os.Getenv(e.key); v != "" {
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("%s inválido (%q): %w", e.key, v, err)
			}
			*e.dst = d
		}
	}
	if v := os.Getenv("OCNEWS_RETENTION_DAYS"); v != "" {
		days, err := strconv.Atoi(v)
		if err != nil || days < 0 {
			return nil, fmt.Errorf("OCNEWS_RETENTION_DAYS inválido: %q", v)
		}
		if days == 0 {
			c.Retention = 0 // retención desactivada explícitamente
		} else {
			c.Retention = time.Duration(days) * 24 * time.Hour
		}
	}
	if c.FetchTimeout <= 0 {
		return nil, fmt.Errorf("OCNEWS_FETCH_TIMEOUT debe ser > 0 (got %s)", c.FetchTimeout)
	}
	if c.FeedInterval <= 0 {
		return nil, fmt.Errorf("OCNEWS_FEED_INTERVAL debe ser > 0 (got %s)", c.FeedInterval)
	}
	if c.MaxGap < c.FeedInterval {
		return nil, fmt.Errorf("OCNEWS_MAX_GAP (%s) debe ser >= OCNEWS_FEED_INTERVAL (%s)", c.MaxGap, c.FeedInterval)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return nil, fmt.Errorf("OCNEWS_LOG_LEVEL inválido: %q (debug|info|warn|error)", c.LogLevel)
	}
	switch c.AuthMode {
	case "local", "opencloud":
	default:
		return nil, fmt.Errorf("OCNEWS_AUTH_MODE inválido: %q (local|opencloud)", c.AuthMode)
	}
	if c.AuthMode == "opencloud" && c.OpenCloudURL == "" {
		return nil, fmt.Errorf("OCNEWS_AUTH_MODE=opencloud exige OCNEWS_OPENCOLOUD_URL")
	}
	if c.Addr == "" {
		return nil, fmt.Errorf("OCNEWS_ADDR no puede estar vacío")
	}
	return c, nil
}

// FaviconsDir devuelve la ruta de la caché de favicons.
func (c *Config) FaviconsDir() string {
	return c.DataDir + string(os.PathSeparator) + "favicons"
}

// DBPath devuelve la ruta del fichero SQLite dentro del data dir.
func (c *Config) DBPath() string {
	return c.DataDir + string(os.PathSeparator) + "ocnews.db"
}

// ParseOffsetID parsea un parámetro de query numérico (usa def si ausente).
func ParseOffsetID(s string, def int64) (int64, error) {
	if s == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parámetro numérico inválido: %q", s)
	}
	return n, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
