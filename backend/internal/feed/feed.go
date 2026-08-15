// Package feed obtiene y normaliza feeds RSS/Atom a los modelos del store.
// Fetcher es una interfaz para inyectar fakes en tests y en la API.
package feed

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/gnacho/ocnews/backend/internal/store"
)

const maxBodyBytes = 10 << 20 // 10 MB de techo por feed

type Fetcher interface {
	// Fetch descarga y parsea el feed; error si no se puede leer.
	Fetch(ctx context.Context, url string) (*store.Feed, []store.NewItem, error)
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
