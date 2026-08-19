package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestShareAPI(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	_, body := e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://sh.example/f"})
	var feedResp struct {
		Feeds []struct{ ID int64 }
	}
	decode(t, body, &feedResp)
	if len(feedResp.Feeds) != 1 {
		t.Fatalf("feed no creado: %s", body)
	}

	// obtener un item del feed
	_, body = e.do(t, "GET", "/items?type=0&id="+fmt.Sprint(feedResp.Feeds[0].ID), e.user, e.pass, nil)
	var items struct {
		Items []struct {
			ID    int64
			Title string
		}
	}
	decode(t, body, &items)
	if len(items.Items) == 0 {
		t.Fatalf("sin items: %s", body)
	}
	itemID := items.Items[0].ID

	code, body := e.do(t, "POST", fmt.Sprintf("/items/%d/share", itemID), e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("crear share: %d %s", code, body)
	}
	var share struct {
		Share struct {
			Token string `json:"token"`
			URL   string `json:"url"`
		} `json:"share"`
	}
	decode(t, body, &share)
	if share.Share.Token == "" || share.Share.URL == "" {
		t.Fatalf("share incompleto: %s", body)
	}

	// la URL pública debe servir HTML sin auth
	resp, err := e.client.Get(e.ts.URL + Base + "/share/" + share.Share.Token)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("vista pública: %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type de la vista pública: %q", ct)
	}

	// borrar share → 204 y la vista pública pasa a 404
	if code, _ := e.do(t, "DELETE", fmt.Sprintf("/items/%d/share", itemID), e.user, e.pass, nil); code != 204 {
		t.Fatalf("borrar share: %d", code)
	}
	resp2, err := e.client.Get(e.ts.URL + Base + "/share/" + share.Share.Token)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("vista pública tras borrar esperaba 404, tengo %d", resp2.StatusCode)
	}
}
