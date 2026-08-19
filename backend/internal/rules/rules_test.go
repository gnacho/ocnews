package rules

import "testing"

func TestParseAndBlock(t *testing.T) {
	rs, err := Parse(
		"EntryTitle=(?i)miniflux\nEntryURL=utm_\nEntryAuthor=(?i)spam",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !rs.Ignore(Fields{Title: "New Miniflux release"}) {
		t.Error("título con miniflux no ignorado")
	}
	if !rs.Ignore(Fields{URL: "https://x.com/a?utm_source=rss"}) {
		t.Error("URL con utm_ no ignorada")
	}
	if !rs.Ignore(Fields{Author: "Spam Bot"}) {
		t.Error("autor spam no ignorado")
	}
	if rs.Ignore(Fields{Title: "Noticias normales"}) {
		t.Error("artículo normal ignorado sin motivo")
	}
}

func TestKeepRules(t *testing.T) {
	rs, err := Parse("", "EntryTitle=(?i)odoo\nEntryTitle=(?i)python")
	if err != nil {
		t.Fatal(err)
	}
	if !rs.Ignore(Fields{Title: "Noticias varias"}) {
		t.Error("keep: artículo fuera de las reglas no ignorado")
	}
	if rs.Ignore(Fields{Title: "Odoo 17 lanzado"}) {
		t.Error("keep: artículo que casa con odoo ignorado")
	}
	if rs.Ignore(Fields{Title: "python 3.13"}) {
		t.Error("keep: artículo que casa con python ignorado")
	}
}

func TestBlockPlusKeep(t *testing.T) {
	// block primero: matchea block aunque no case keep
	rs, err := Parse("EntryTitle=(?i)ads", "EntryTitle=(?i)odoo")
	if err != nil {
		t.Fatal(err)
	}
	if !rs.Ignore(Fields{Title: "Ads for Odoo"}) {
		t.Error("block debería ignorar aunque keep case")
	}
	if rs.Ignore(Fields{Title: "Odoo news"}) {
		t.Error("keep: odoo no ignorado")
	}
	if !rs.Ignore(Fields{Title: "Random"}) {
		t.Error("keep: random debería ignorarse")
	}
}

func TestDatePatterns(t *testing.T) {
	rs, err := Parse("EntryDate=future", "")
	if err != nil {
		t.Fatal(err)
	}
	future := int64(4102444800) // 2100-01-01
	if !rs.Ignore(Fields{PubDate: future}) {
		t.Error("fecha futura no ignorada")
	}
	if rs.Ignore(Fields{PubDate: 1700000000}) {
		t.Error("fecha pasada ignorada")
	}

	rs2, err := Parse("EntryDate=before:2024-01-01", "")
	if err != nil {
		t.Fatal(err)
	}
	if !rs2.Ignore(Fields{PubDate: 1600000000}) { // 2020
		t.Error("artículo de 2020 no ignorado por before:2024")
	}
	if rs2.Ignore(Fields{PubDate: 1735689600}) { // 2025-01-01
		t.Error("artículo de 2025 ignorado por before:2024")
	}

	rs3, err := Parse("EntryDate=max-age:7d", "")
	if err != nil {
		t.Fatal(err)
	}
	if !rs3.Ignore(Fields{PubDate: 1700000000}) { // 2023, muy viejo
		t.Error("artículo viejo no ignorado por max-age:7d")
	}
}

func TestInvalidRules(t *testing.T) {
	if _, err := Parse("EntryTitle=(", ""); err == nil {
		t.Error("regex inválida debería fallar")
	}
	if _, err := Parse("EntryFoo=bar", ""); err == nil {
		t.Error("campo desconocido debería fallar")
	}
	if _, err := Parse("sinIgual", ""); err == nil {
		t.Error("línea sin '=' debería fallar")
	}
	if _, err := Parse("EntryDate=max-age:xyz", ""); err == nil {
		t.Error("max-age inválido debería fallar")
	}
	if _, err := Parse("EntryDate=before:31-12-2023", ""); err == nil {
		t.Error("fecha con formato erróneo debería fallar")
	}
}

func TestCommentsAndBlanks(t *testing.T) {
	rs, err := Parse("# comentario\n\nEntryTitle=(?i)x\n", "")
	if err != nil {
		t.Fatal(err)
	}
	if rs.Ignore(Fields{Title: "abc"}) {
		t.Error("sin match no debe ignorar")
	}
	if !rs.Ignore(Fields{Title: "X marks"}) {
		t.Error("match x no ignorado")
	}
}
