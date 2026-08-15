package feed

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gnacho/ocnews/backend/internal/store"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("leer fixture: %v", err)
	}
	return b
}

func TestParseRSS(t *testing.T) {
	f, items, err := Parse(loadFixture(t, "rss.xml"))
	if err != nil {
		t.Fatalf("parse rss: %v", err)
	}
	if f.Title != "Ejemplo Blog" || f.Link != "https://ejemplo.example" {
		t.Errorf("feed meta mal: %+v", f)
	}
	if len(items) != 2 {
		t.Fatalf("esperaba 2 items, tengo %d", len(items))
	}

	first := items[0]
	if first.GUID != "urn:ejemplo:1" || first.GUIDHash == "" {
		t.Errorf("guid mal: %+v", first)
	}
	if first.Title != "Primera entrada" {
		t.Errorf("title: %q", first.Title)
	}
	if first.Author != "ana@example.com (Ana Prueba)" && first.Author == "" {
		t.Errorf("author vacío inesperado: %q", first.Author)
	}
	if first.URL != "https://ejemplo.example/1" {
		t.Errorf("url: %q", first.URL)
	}
	// content:encoded manda sobre description
	wantBody := "<p>Cuerpo <b>completo</b> de la primera</p>"
	if first.Body != wantBody {
		t.Errorf("body: %q (esperaba content:encoded)", first.Body)
	}
	if first.EnclosureLink == nil || *first.EnclosureLink != "https://ejemplo.example/audio/1.mp3" {
		t.Errorf("enclosure: %+v", first.EnclosureLink)
	}
	if first.EnclosureMime == nil || *first.EnclosureMime != "audio/mpeg" {
		t.Errorf("enclosure mime: %+v", first.EnclosureMime)
	}
	if first.MediaThumbnail == nil || *first.MediaThumbnail != "https://ejemplo.example/img/1.jpg" {
		t.Errorf("media:thumbnail: %+v", first.MediaThumbnail)
	}
	// Mon, 02 Jan 2006 15:04:05 -0700 = 1136214245
	if first.PubDate != time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("MST", -7*3600)).Unix() {
		t.Errorf("pubDate: %d", first.PubDate)
	}

	second := items[1]
	if second.EnclosureLink != nil || second.MediaThumbnail != nil {
		t.Errorf("item sin enclosure no debe tenerlos: %+v", second)
	}
	if second.Body != "Resumen de la segunda" {
		t.Errorf("body fallback description: %q", second.Body)
	}
	if first.Fingerprint == second.Fingerprint || first.Fingerprint == "" {
		t.Errorf("fingerprint debe diferir y ser no vacío")
	}
}

func TestParseAtom(t *testing.T) {
	f, items, err := Parse(loadFixture(t, "atom.xml"))
	if err != nil {
		t.Fatalf("parse atom: %v", err)
	}
	if f.Title != "Atom Ejemplo" || f.Link != "https://atom.example/" {
		t.Errorf("feed meta mal: %+v", f)
	}
	if len(items) != 1 {
		t.Fatalf("esperaba 1 item, tengo %d", len(items))
	}
	it := items[0]
	if it.GUID != "urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a" {
		t.Errorf("atom id como guid: %q", it.GUID)
	}
	if it.Author != "Luis Atom" {
		t.Errorf("author: %q", it.Author)
	}
	if it.Body != "<p>Contenido <em>atom</em></p>" {
		t.Errorf("body: %q", it.Body)
	}
	if it.PubDate != time.Date(2003, 12, 13, 18, 30, 2, 0, time.UTC).Unix() {
		t.Errorf("pubDate: %d", it.PubDate)
	}
}

func TestParseInvalid(t *testing.T) {
	if _, _, err := Parse([]byte("esto no es xml")); err == nil {
		t.Fatal("esperaba error con basura")
	}
}

// dedupe: mismo guid → mismo hash (unicidad por feed_id+guid_hash en BD).
func TestGUIDHashEstable(t *testing.T) {
	_, items, err := Parse(loadFixture(t, "rss.xml"))
	if err != nil {
		t.Fatal(err)
	}
	_, again, _ := Parse(loadFixture(t, "rss.xml"))
	if items[0].GUIDHash != again[0].GUIDHash {
		t.Error("guid_hash debe ser determinista")
	}
}

func TestHasFullContent(t *testing.T) {
	summary := "<p>" + strings.Repeat("a", 300) + "</p>"
	full := "<p>" + strings.Repeat("a", 1200) + "</p>"
	if HasFullContent([]store.NewItem{{Body: summary}}) {
		t.Error("resumen corto no debe marcar full")
	}
	if !HasFullContent([]store.NewItem{{Body: summary}, {Body: full}}) {
		t.Error("un item largo basta para marcar full")
	}
	if HasFullContent(nil) {
		t.Error("sin items no marca full")
	}
}
