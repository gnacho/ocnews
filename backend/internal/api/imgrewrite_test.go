package api

import (
	"strings"
	"testing"
)

func TestRewriteImages(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	in := `<p>hola</p><img src="https://cdn.example/pic.webp?resize=406%2C232&amp;ssl=1" alt="x"><img src="/local.png">`
	out := e.srv.rewriteImages(in)
	if !strings.Contains(out, `/img?u=https%3A%2F%2Fcdn.example%2Fpic.webp`) {
		t.Fatalf("src externo no reescrito con QueryEscape: %s", out)
	}
	if !strings.Contains(out, "&amp;t=") {
		t.Fatalf("falta el parámetro firmado: %s", out)
	}
	if !strings.Contains(out, `src="/local.png"`) {
		t.Fatalf("src relativo no debe tocarse: %s", out)
	}
	// round-trip: la u firmada debe coincidir con lo que q.Get() recuperará
	sig := e.srv.imgs.Sign("https://cdn.example/pic.webp?resize=406%2C232&ssl=1")
	if !strings.Contains(out, "&amp;t="+sig) {
		t.Fatalf("la firma debe calcularse sobre la URL original exacta: %s", out)
	}
}
