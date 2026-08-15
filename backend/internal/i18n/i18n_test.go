package i18n

import "testing"

func TestNegotiate(t *testing.T) {
	cases := []struct {
		user, accept string
		want         Lang
	}{
		{"es", "", ES},
		{"en", "es-ES,es;q=0.9", EN},   // preferencia explícita manda
		{"auto", "es-ES,es;q=0.9,en;q=0.8", ES},
		{"auto", "en-GB,en;q=0.9", EN},
		{"auto", "fr-FR,de;q=0.9", EN}, // sin match → default
		{"auto", "", EN},
		{"", "es", ES},
		{"auto", "es-ES", ES},
	}
	for _, c := range cases {
		if got := Negotiate(c.user, c.accept); got != c.want {
			t.Errorf("Negotiate(%q,%q)=%q, esperaba %q", c.user, c.accept, got, c.want)
		}
	}
}

func TestT(t *testing.T) {
	if T(ES, "folder_not_found") != "carpeta no encontrada" {
		t.Error("mensaje ES mal")
	}
	if T(EN, "folder_not_found") != "folder not found" {
		t.Error("mensaje EN mal")
	}
	if T(Lang("fr"), "folder_not_found") != "folder not found" {
		t.Error("idioma desconocido debe caer en EN")
	}
	if T(ES, "clave_inexistente") != "error interno" {
		t.Error("clave desconocida debe caer en internal_error")
	}
}
