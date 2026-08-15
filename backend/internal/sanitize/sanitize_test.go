package sanitize

import (
	"strings"
	"testing"
)

func TestBodyStripsScripts(t *testing.T) {
	cases := []struct{ name, in, mustNotContain string }{
		{"script tag", `<p>hola</p><script>alert(1)</script>`, "<script"},
		{"inline handler", `<p onclick="evil()">hola</p>`, "onclick"},
		{"img onerror", `<img src="https://x.example/a.png" onerror="evil()">`, "onerror"},
		{"javascript href", `<a href="javascript:evil()">pincha</a>`, "javascript:"},
		{"iframe", `<iframe src="https://evil.example"></iframe>`, "<iframe"},
		{"style block", `<style>body{}</style><p>x</p>`, "<style"},
	}
	for _, c := range cases {
		out := Body(c.in)
		if strings.Contains(out, c.mustNotContain) {
			t.Errorf("%s: salida contiene %q: %s", c.name, c.mustNotContain, out)
		}
	}
}

func TestBodyKeepsFormatting(t *testing.T) {
	in := `<h2>Titulo</h2><p>Párrafo con <b>negrita</b>, <i>cursiva</i> y <a href="https://ejemplo.example/x" title="t">enlace</a>.</p><ul><li>uno</li></ul><figure><img src="https://ejemplo.example/i.png" alt="alt text" width="600"><figcaption>pie</figcaption></figure>`
	out := Body(in)
	for _, want := range []string{"<h2>", "<b>negrita</b>", `<a href="https://ejemplo.example/x"`, "<ul>", "<li>uno</li>", `<img src="https://ejemplo.example/i.png"`, "<figcaption>pie</figcaption>"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en: %s", want, out)
		}
	}
}

func TestBodyEmpty(t *testing.T) {
	if Body("") != "" {
		t.Error("vacío debe quedar vacío")
	}
}

func TestBodySanitizesBrokenHTML(t *testing.T) {
	// HTML roto se normaliza; el contenido legible sobrevive
	out := Body(`<p>sin cerrar <b>negrita`)
	if !strings.Contains(out, "sin cerrar") || !strings.Contains(out, "negrita") {
		t.Errorf("contenido perdido: %s", out)
	}
}
