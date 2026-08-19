// Package websub: cliente de WebSub (PubSubHubbub) para recibir items de los
// feeds publicados por hubs (WordPress, Blogger, Medium…) en tiempo real.
// Suscripción: POST form-urlencoded al hub con hub.mode/hub.topic/hub.callback.
package websub

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gnacho/ocnews/backend/internal/netguard"
)

type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: netguard.Client(15 * time.Second)}
}

// NewAllowLocal es como New pero permite loopback. SOLO tests.
func NewAllowLocal() *Client {
	return &Client{http: netguard.ClientAllowLocal(15 * time.Second)}
}

// Subscribe pide al hub la suscripción (o renovación) de un topic.
// secret permite firmar los deliveries (X-Hub-Signature).
func (c *Client) Subscribe(ctx context.Context, hub, topic, callback, secret string) error {
	u, err := url.Parse(hub)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("hub inválido: %q", hub)
	}
	form := url.Values{}
	form.Set("hub.mode", "subscribe")
	form.Set("hub.topic", topic)
	form.Set("hub.callback", callback)
	if secret != "" {
		form.Set("hub.secret", secret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hub, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("suscribir al hub %s: %w", hub, err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("hub %s respondió HTTP %d", hub, resp.StatusCode)
	}
	return nil
}
