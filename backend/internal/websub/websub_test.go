package websub

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestSubscribe(t *testing.T) {
	var got url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("método: %s", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		got, _ = url.ParseQuery(string(b))
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()

	c := NewAllowLocal()
	if err := c.Subscribe(context.Background(), ts.URL, "https://site.example/feed.xml",
		"https://ocnews.example/cb/1", "secreto"); err != nil {
		t.Fatal(err)
	}
	if got.Get("hub.mode") != "subscribe" {
		t.Errorf("hub.mode: %q", got.Get("hub.mode"))
	}
	if got.Get("hub.topic") != "https://site.example/feed.xml" {
		t.Errorf("hub.topic: %q", got.Get("hub.topic"))
	}
	if got.Get("hub.callback") != "https://ocnews.example/cb/1" {
		t.Errorf("hub.callback: %q", got.Get("hub.callback"))
	}
	if got.Get("hub.secret") != "secreto" {
		t.Errorf("hub.secret: %q", got.Get("hub.secret"))
	}
}

func TestSubscribeError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()
	c := NewAllowLocal()
	if err := c.Subscribe(context.Background(), ts.URL, "t", "c", ""); err == nil {
		t.Error("HTTP 400 debería devolver error")
	}
}

func TestSubscribeInvalidHub(t *testing.T) {
	c := NewAllowLocal()
	if err := c.Subscribe(context.Background(), "ftp://hub.example", "t", "c", ""); err == nil {
		t.Error("hub sin http/https debería fallar")
	}
}
