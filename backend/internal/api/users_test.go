package api

import (
	"testing"
)

// TestListWidthSetting: el ancho de la columna de artículos persiste en
// user_settings vía /api/me/settings (validación 240-700).
func TestListWidthSetting(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	// inválido → 422
	if code, _ := e.do(t, "PUT", "/api/me/settings", e.user, e.pass,
		map[string]string{"readerListWidth": "10"}); code != 422 {
		t.Fatalf("ancho 10 esperaba 422, tengo %d", code)
	}
	// válido → persiste
	code, body := e.do(t, "PUT", "/api/me/settings", e.user, e.pass,
		map[string]string{"readerListWidth": "520"})
	if code != 200 {
		t.Fatalf("guardar ancho: %d %s", code, body)
	}
	var got struct {
		ReaderListWidth string `json:"readerListWidth"`
	}
	decode(t, body, &got)
	if got.ReaderListWidth != "520" {
		t.Fatalf("ancho no persistido: %s", body)
	}
	// leer de nuevo
	code, body = e.do(t, "GET", "/api/me/settings", e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("leer settings: %d", code)
	}
	decode(t, body, &got)
	if got.ReaderListWidth != "520" {
		t.Fatalf("ancho no leído: %s", body)
	}
	// default (380) → se limpia
	code, body = e.do(t, "PUT", "/api/me/settings", e.user, e.pass,
		map[string]string{"readerListWidth": "380"})
	if code != 200 {
		t.Fatalf("reset ancho: %d", code)
	}
	decode(t, body, &got)
	if got.ReaderListWidth != "" {
		t.Fatalf("ancho default debería limpiarse: %s", body)
	}
}
