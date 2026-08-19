package api

import (
	"fmt"
	"testing"
)

func TestFeedScraperAPI(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	_, body := e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://sc.example/f"})
	var feedResp struct {
		Feeds []struct{ ID int64 }
	}
	decode(t, body, &feedResp)
	if len(feedResp.Feeds) != 1 {
		t.Fatalf("feed no creado: %s", body)
	}
	feedID := feedResp.Feeds[0].ID

	// selector inválido → 422
	if code, _ := e.do(t, "POST", fmt.Sprintf("/feeds/%d/scraper", feedID), e.user, e.pass,
		map[string]string{"scraperSelector": "div["}); code != 422 {
		t.Fatalf("selector inválido esperaba 422, tengo %d", code)
	}

	// selector válido → 200 y persistido
	code, body := e.do(t, "POST", fmt.Sprintf("/feeds/%d/scraper", feedID), e.user, e.pass,
		map[string]string{"scraperSelector": "div#articleBody, article"})
	if code != 200 {
		t.Fatalf("guardar selector: %d %s", code, body)
	}
	var got struct {
		ScraperSelector string `json:"scraperSelector"`
	}
	decode(t, body, &got)
	if got.ScraperSelector != "div#articleBody, article" {
		t.Errorf("selector no persistido: %s", body)
	}

	// vacío = quitar
	code, body = e.do(t, "POST", fmt.Sprintf("/feeds/%d/scraper", feedID), e.user, e.pass,
		map[string]string{"scraperSelector": ""})
	if code != 200 {
		t.Fatalf("quitar selector: %d", code)
	}
	decode(t, body, &got)
	if got.ScraperSelector != "" {
		t.Errorf("selector no quitado: %s", body)
	}
}
