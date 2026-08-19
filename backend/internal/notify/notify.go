// Package notify: notificaciones push vía ntfy (https://ntfy.sh o servidor
// propio). El topic puede ser global (env) o por usuario (user_settings).
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gnacho/ocnews/backend/internal/netguard"
)

type Notifier struct {
	baseURL string
	topic   string // topic global (vacío = solo per-usuario)
	client  *http.Client
	log     *slog.Logger
}

// New crea un Notifier. topic puede quedar vacío (notificaciones solo con
// topic por usuario).
func New(baseURL, topic string, log *slog.Logger) *Notifier {
	if baseURL == "" {
		baseURL = "https://ntfy.sh"
	}
	return &Notifier{
		baseURL: baseURL,
		topic:   topic,
		client:  netguard.Client(10 * time.Second),
		log:     log,
	}
}

// NewAllowLocal es como New pero permite loopback. SOLO tests (httptest).
func NewAllowLocal(baseURL, topic string, log *slog.Logger) *Notifier {
	if baseURL == "" {
		baseURL = "https://ntfy.sh"
	}
	return &Notifier{
		baseURL: baseURL,
		topic:   topic,
		client:  netguard.ClientAllowLocal(10 * time.Second),
		log:     log,
	}
}

// Topic devuelve el topic global configurado ("" si no hay).
func (n *Notifier) Topic() string { return n.topic }

// Notify publica un mensaje en el topic dado (ntfy POST JSON).
func (n *Notifier) Notify(ctx context.Context, topic, title, message string) error {
	if topic == "" {
		return fmt.Errorf("topic vacío")
	}
	body, err := json.Marshal(map[string]string{
		"topic":   topic,
		"title":   title,
		"message": message,
		"tags":    "inbox_tray",
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		n.baseURL+"/"+url.PathEscape(topic), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("enviar a ntfy: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("ntfy: HTTP %d", resp.StatusCode)
	}
	return nil
}
