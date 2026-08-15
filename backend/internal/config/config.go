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
}

func Load() (*Config, error) {
	c := &Config{
		Addr:         env("OCNEWS_ADDR", ":8094"),
		DataDir:      env("OCNEWS_DATA_DIR", "./data"),
		FetchTimeout: 20 * time.Second,
		LogLevel:     env("OCNEWS_LOG_LEVEL", "info"),
		AuthUser:     os.Getenv("AUTH_USER"),
		AuthPass:     os.Getenv("AUTH_PASS"),
	}
	if v := os.Getenv("OCNEWS_FETCH_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("OCNEWS_FETCH_TIMEOUT inválido (%q): %w", v, err)
		}
		c.FetchTimeout = d
	}
	if c.FetchTimeout <= 0 {
		return nil, fmt.Errorf("OCNEWS_FETCH_TIMEOUT debe ser > 0 (got %s)", c.FetchTimeout)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return nil, fmt.Errorf("OCNEWS_LOG_LEVEL inválido: %q (debug|info|warn|error)", c.LogLevel)
	}
	if c.Addr == "" {
		return nil, fmt.Errorf("OCNEWS_ADDR no puede estar vacío")
	}
	return c, nil
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
