package notify

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotify(t *testing.T) {
	var got struct {
		Topic   string `json:"topic"`
		Title   string `json:"title"`
		Message string `json:"message"`
		Tags    string `json:"tags"`
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decodificar body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	n := NewAllowLocal(ts.URL, "", log)
	if err := n.Notify(context.Background(), "mytopic", "Título", "Mensaje"); err != nil {
		t.Fatal(err)
	}
	if got.Topic != "mytopic" || got.Title != "Título" || got.Message != "Mensaje" {
		t.Errorf("payload erróneo: %+v", got)
	}
	if got.Tags == "" {
		t.Error("sin tags en el payload")
	}
}

func TestNotifyHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	n := NewAllowLocal(ts.URL, "", log)
	if err := n.Notify(context.Background(), "t", "x", "y"); err == nil {
		t.Error("HTTP 500 debería devolver error")
	}
}

func TestNotifyEmptyTopic(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	n := NewAllowLocal("http://127.0.0.1:9", "", log)
	if err := n.Notify(context.Background(), "", "x", "y"); err == nil {
		t.Error("topic vacío debería fallar sin tocar la red")
	}
}
