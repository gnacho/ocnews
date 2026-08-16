// Package feed obtiene y normaliza feeds RSS/Atom a los modelos del store.
// Fetcher es una interfaz para inyectar fakes en tests y en la API.
package feed

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/gnacho/ocnews/backend/internal/sanitize"
	"github.com/gnacho/ocnews/backend/internal/store"
)

const maxBodyBytes = 10 << 20 // 10 MB de techo por feed

type Fetcher interface {
	// Fetch descarga y parsea el feed; error si no se puede leer.
	Fetch(ctx context.Context, url string) (*store.Feed, []store.NewItem, error)
	// Discover autodetecta feeds RSS/Atom en una página web.
	Discover(ctx context.Context, url string) ([]DiscoveredFeed, error)
}

type HTTPFetcher struct {
	Client *http.Client
}

func NewHTTPFetcher(timeout time.Duration) *HTTPFetcher {
	return &HTTPFetcher{Client: &http.Client{Timeout: timeout}}
}

func (h *HTTPFetcher) Fetch(ctx context.Context, url string) (*store.Feed, []store.NewItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("url inválida: %w", err)
	}
	req.Header.Set("User-Agent", "ocnews/0.5 (+https://github.com/gnacho/ocnews)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")
	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("descargar feed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("descargar feed: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("leer feed: %w", err)
	}
	return Parse(body)
}

// minFullText: umbral de texto plano para considerar que un item llega
// completo desde el feed (los resúmenes típicos quedan en 100-600 chars).
const minFullText = 900

// HasFullContent dice si ALGÚN item de la tanda ya trae el artículo entero
// (heurística: suficiente texto plano). Se evalúa ANTES de sanitizar.
func HasFullContent(items []store.NewItem) bool {
	for _, it := range items {
		if len(plainTextLen(it.Body)) >= minFullText {
			return true
		}
	}
	return false
}

// plainTextLen: longitud aproximada de texto (etiquetas fuera).
func plainTextLen(html string) string {
	var b strings.Builder
	inTag := false
	for _, r := range html {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SanitizeItems limpia el HTML de los cuerpos de una tanda de items.
// ÚNICA vía de entrada a la BD: la llaman la suscripción (API) y el
// refresher (scheduler) — nunca persistir bodies sin pasar por aquí.
func SanitizeItems(items []store.NewItem) {
	for i := range items {
		items[i].Body = sanitize.Body(items[i].Body)
	}
}

// Parse normaliza un documento RSS/Atom (separable de la red para tests).
func Parse(body []byte) (*store.Feed, []store.NewItem, error) {
	fp := gofeed.NewParser()
	parsed, err := fp.ParseString(string(body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsear feed: %w", err)
	}

	f := &store.Feed{
		URL:    parsed.FeedLink,
		Title:  strings.TrimSpace(parsed.Title),
		Link:   websiteLink(parsed),
		Added:  time.Now().Unix(),
	}
	if f.Title == "" {
		f.Title = "feed sin título"
	}

	items := make([]store.NewItem, 0, len(parsed.Items))
	for _, it := range parsed.Items {
		guid := it.GUID
		if guid == "" {
			guid = it.Link
		}
		if guid == "" {
			continue // sin identificador no es deduplicable
		}
		pubDate := time.Now().Unix()
		if it.PublishedParsed != nil {
			pubDate = it.PublishedParsed.Unix()
		} else if it.UpdatedParsed != nil {
			pubDate = it.UpdatedParsed.Unix()
		}
		body := it.Content
		if body == "" {
			body = it.Description
		}
		ni := store.NewItem{
			GUID:     guid,
			GUIDHash: hashGUID(guid),
			URL:      it.Link,
			Title:    strings.TrimSpace(it.Title),
			Author:   authorOf(it),
			PubDate:  pubDate,
			Body:     body,
		}
		if len(it.Enclosures) > 0 {
			mime, link := it.Enclosures[0].Type, it.Enclosures[0].URL
			ni.EnclosureMime, ni.EnclosureLink = &mime, &link
		}
		if thumb, ok := mediaExt(it, "thumbnail", "url"); ok {
			ni.MediaThumbnail = &thumb
		}
		if desc, ok := mediaExt(it, "description", ""); ok {
			ni.MediaDescription = &desc
		}
		ni.Fingerprint = fingerprint(ni)
		items = append(items, ni)
	}
	return f, items, nil
}

func websiteLink(f *gofeed.Feed) string {
	switch {
	case f.Link != "":
		return f.Link
	case len(f.Links) > 0:
		return f.Links[0]
	case f.FeedLink != "":
		return f.FeedLink
	}
	return ""
}

func authorOf(it *gofeed.Item) string {
	if it.Author != nil {
		return strings.TrimSpace(it.Author.Name)
	}
	for _, a := range it.Authors {
		if a != nil && a.Name != "" {
			return strings.TrimSpace(a.Name)
		}
	}
	return ""
}

// mediaExt extrae extensiones media:* (RSS con namespaces media).
func mediaExt(it *gofeed.Item, name, attr string) (string, bool) {
	exts, ok := it.Extensions["media"]
	if !ok {
		return "", false
	}
	list, ok := exts[name]
	if !ok || len(list) == 0 {
		return "", false
	}
	if attr == "" {
		if list[0].Value == "" {
			return "", false
		}
		return list[0].Value, true
	}
	v, ok := list[0].Attrs[attr]
	return v, ok && v != ""
}

func hashGUID(guid string) string {
	sum := md5.Sum([]byte(guid))
	return fmt.Sprintf("%x", sum)
}

// fingerprint: hash sobre título+body+url+enclosure (dedupe entre feeds).
func fingerprint(ni store.NewItem) string {
	enc := ""
	if ni.EnclosureLink != nil {
		enc = *ni.EnclosureLink
	}
	sum := md5.Sum([]byte(ni.Title + "\x00" + ni.Body + "\x00" + ni.URL + "\x00" + enc))
	return fmt.Sprintf("%x", sum)
}

// DiscoveredFeed: un feed candidato detectado en la página de un sitio.
type DiscoveredFeed struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Type  string `json:"type"` // rss | atom
}

var feedLinkRe = regexp.MustCompile(`(?i)<link[^>]*rel=["']\s*alternate\s*["'][^>]*>`)

// Discover descarga una página web y extrae los enlaces de feed RSS/Atom
// (autodetección de suscripciones, feature de la extensión). Devuelve los
// feeds encontrados; si la URL ES un feed válido, devuelve ese único feed.
func (h *HTTPFetcher) Discover(ctx context.Context, url string) ([]DiscoveredFeed, error) {
	if f, items, err := h.Fetch(ctx, url); err == nil && len(items) >= 0 {
		_ = f
		// es un feed ya → lo devolvemos tal cual
		return []DiscoveredFeed{{URL: url, Title: f.Title, Type: feedKind(f.URL)}}, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("url inválida: %w", err)
	}
	req.Header.Set("User-Agent", "ocnews/0.5 (+https://github.com/gnacho/ocnews)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("descargar página: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("descargar página: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("leer página: %w", err)
	}
	base, _ := urlpkg.Parse(url)
	out := []DiscoveredFeed{}
	seen := map[string]bool{}
	for _, m := range feedLinkRe.FindAll(body, -1) {
		link := linkAttr(string(m), "href")
		typ := strings.ToLower(linkAttr(string(m), "type"))
		if link == "" {
			continue
		}
		if !strings.Contains(typ, "rss") && !strings.Contains(typ, "atom") && !strings.HasSuffix(link, ".xml") {
			continue
		}
		abs, err := base.Parse(link)
		if err != nil {
			continue
		}
		kind := "rss"
		if strings.Contains(typ, "atom") {
			kind = "atom"
		}
		if seen[abs.String()] {
			continue
		}
		seen[abs.String()] = true
		out = append(out, DiscoveredFeed{URL: abs.String(), Title: titleFromFeedLink(string(m)), Type: kind})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no se encontraron feeds en la página")
	}
	return out, nil
}

func feedKind(u string) string {
	if strings.Contains(strings.ToLower(u), "atom") {
		return "atom"
	}
	return "rss"
}

func linkAttr(tag, attr string) string {
	re := regexp.MustCompile(`(?i)` + attr + `\s*=\s*["']([^"']*)["']`)
	m := re.FindStringSubmatch(tag)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func titleFromFeedLink(tag string) string {
	if t := linkAttr(tag, "title"); t != "" {
		return t
	}
	return ""
}
