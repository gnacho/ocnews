package api

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const hubRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<title>HubFeed</title><link>https://h.example</link>
<item><guid>hub-1</guid><title>Noticia por WebSub</title><link>https://h.example/1</link>
<description>Descripción breve</description></item>
</channel></rss>`

func TestWebSubCallback(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	feedURL := "https://hub.example/feed.xml"
	_, body := e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": feedURL})
	var feedResp struct {
		Feeds []struct{ ID int64 }
	}
	decode(t, body, &feedResp)
	if len(feedResp.Feeds) != 1 {
		t.Fatalf("feed no creado: %s", body)
	}
	feedID := feedResp.Feeds[0].ID

	// registrar la suscripción como lo haría el scheduler
	if err := e.srv.store.UpsertWebSub(feedID, "https://hub.example/hub", feedURL); err != nil {
		t.Fatal(err)
	}
	if err := e.srv.store.SaveWebSubSecret(feedID, "supersecreto", "/cb/"+fmt.Sprint(feedID)); err != nil {
		t.Fatal(err)
	}

	// 1) verificación GET → challenge
	cb := fmt.Sprintf("/index.php/apps/news/api/v1-3/websub/callback/%d?hub.mode=subscribe&hub.topic=%s&hub.challenge=abcdef&hub.lease_seconds=86400", feedID, feedURL)
	resp, err := e.client.Get(e.ts.URL + cb)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("verificación: %d", resp.StatusCode)
	}
	buf := make([]byte, 16)
	n, _ := resp.Body.Read(buf)
	if string(buf[:n]) != "abcdef" {
		t.Errorf("challenge no devuelto: %q", buf[:n])
	}

	// 2) delivery firmado → 204 e item ingerido
	mac := hmac.New(sha1.New, []byte("supersecreto"))
	mac.Write([]byte(hubRSS))
	sig := "sha1=" + hex.EncodeToString(mac.Sum(nil))
	code, _ := e.doRaw(t, "POST", fmt.Sprintf("/index.php/apps/news/api/v1-3/websub/callback/%d", feedID),
		"", "", nil)
	_ = code
	// doRaw no manda body personalizado; hacemos el POST a mano con cabecera
	req := newHubPost(t, e, feedID, hubRSS, sig)
	resp2, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 204 {
		t.Fatalf("delivery: %d", resp2.StatusCode)
	}

	// el item llegó al feed
	_, itemsBody := e.do(t, "GET", "/items?type=0&id="+fmt.Sprint(feedID), e.user, e.pass, nil)
	var items struct {
		Items []struct{ Title string }
	}
	decode(t, itemsBody, &items)
	found := false
	for _, it := range items.Items {
		if strings.Contains(it.Title, "WebSub") {
			found = true
		}
	}
	if !found {
		t.Errorf("el item de WebSub no llegó al feed: %s", itemsBody)
	}

	// 3) firma incorrecta → 401
	badReq := newHubPost(t, e, feedID, hubRSS, "sha1=00deadbeef00")
	resp3, err := e.client.Do(badReq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != 401 {
		t.Fatalf("firma incorrecta esperaba 401, tengo %d", resp3.StatusCode)
	}
}

func TestVerifyHubSignature(t *testing.T) {	body := []byte("hola")
	mac := hmac.New(sha1.New, []byte("k"))
	mac.Write(body)
	if !verifyHubSignature("sha1="+hex.EncodeToString(mac.Sum(nil)), "k", body) {
		t.Error("firma sha1 válida rechazada")
	}
	if verifyHubSignature("sha1="+hex.EncodeToString(mac.Sum(nil)), "otra", body) {
		t.Error("firma con clave distinta aceptada")
	}
	if verifyHubSignature("sha1=zz", "k", body) {
		t.Error("firma no hex aceptada")
	}
	if verifyHubSignature("md5=abc", "k", body) {
		t.Error("algoritmo no soportado aceptado")
	}
}

func newHubPost(t *testing.T, e *testEnv, feedID int64, bodyXML, sig string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost,
		e.ts.URL+fmt.Sprintf("/index.php/apps/news/api/v1-3/websub/callback/%d", feedID),
		strings.NewReader(bodyXML))
	if err != nil {
		t.Fatal(err)
	}
	if sig != "" {
		req.Header.Set("X-Hub-Signature", sig)
	}
	return req
}
