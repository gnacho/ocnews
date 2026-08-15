package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gnacho/ocnews/backend/internal/auth"
	"github.com/gnacho/ocnews/backend/internal/feed"
	"github.com/gnacho/ocnews/backend/internal/extract"
	"github.com/gnacho/ocnews/backend/internal/favicon"
	"github.com/gnacho/ocnews/backend/internal/imgproxy"
	"github.com/gnacho/ocnews/backend/internal/refresher"
	"github.com/gnacho/ocnews/backend/internal/store"
)

// fakeFetcher sirve feeds enlatados por URL; refreshes cuenta las llamadas
// para simular que el feed publica items nuevos.
type fakeFetcher struct {
	fetches  int
	failURLs map[string]bool
}

func (f *fakeFetcher) Fetch(_ context.Context, url string) (*store.Feed, []store.NewItem, error) {
	f.fetches++
	if f.failURLs != nil && f.failURLs[url] {
		return nil, nil, fmt.Errorf("feed roto (fake)")
	}
	fd := &store.Feed{Title: "Feed " + url, Link: "https://site.example"}
	base := len(url)
	if f.fetches > 1 {
		base++ // tras un refresh aparece un item extra
	}
	items := make([]store.NewItem, 0, base)
	for i := 0; i < base; i++ {
		guid := fmt.Sprintf("%s#%d", url, i)
		items = append(items, store.NewItem{
			GUID: guid, GUIDHash: fmt.Sprintf("h%d", i),
			URL: "https://site.example/" + fmt.Sprint(i),
			Title: fmt.Sprintf("Item %d de %s", i, url),
			PubDate: int64(1000 + i),
		})
	}
	return fd, items, nil
}

type testEnv struct {
	ts         *httptest.Server
	client     *http.Client
	user       string
	pass       string
	faviconDir string
	srv        *Server
}

func newTestEnv(t *testing.T, fetcher feed.Fetcher) *testEnv {
	t.Helper()
	st, err := store.Open(t.TempDir() + string(os.PathSeparator) + "test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	hash, err := auth.HashPassword("pass1234")
	if err != nil {
		t.Fatal(err)
	}
	nachoID, err := st.CreateUser("nacho", hash, "Nacho", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetUserLanguage(nachoID, "es"); err != nil {
		t.Fatal(err)
	}
	hash2, _ := auth.HashPassword("otra1234")
	if _, err := st.CreateUser("otro", hash2, "Otro", "user"); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	refresh := refresher.New(st, fetcher, log, time.Minute, time.Hour)
	favDir := t.TempDir()
	fc, err := favicon.NewCache(favDir, log)
	if err != nil {
		t.Fatal(err)
	}
	imgs, err := imgproxy.New(t.TempDir(), log)
	if err != nil {
		t.Fatal(err)
	}
	validator := &auth.LocalValidator{Store: st}
	ex := extract.New(5 * time.Second)
	srv := NewServer(st, validator, fetcher, refresh, fc, imgs, ex, 90*24*time.Hour, log)
	h := srv.Handler()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return &testEnv{ts: ts, client: ts.Client(), user: "nacho", pass: "pass1234", faviconDir: favDir, srv: srv}
}

// do ejecuta una petición con Basic auth del usuario dado.
func (e *testEnv) do(t *testing.T, method, path, user, pass string, body any) (int, []byte) {
	t.Helper()
	if strings.HasPrefix(path, "/api/") {
		return e.doRaw(t, method, path, user, pass, body)
	}
	return e.doRaw(t, method, Base+path, user, pass, body)
}

func (e *testEnv) doRaw(t *testing.T, method, path, user, pass string, body any) (int, []byte) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.ts.URL+path, rd)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth(user, pass)
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func decode(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decodificar %s: %v", data, err)
	}
}

func TestHealthzWithoutAuth(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	resp, err := e.client.Get(e.ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("healthz: %d", resp.StatusCode)
	}
}

func TestAuthRequired(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	code, _ := e.do(t, "GET", "/version", "", "", nil)
	if code != 401 {
		t.Fatalf("sin auth esperaba 401, tengo %d", code)
	}
	code, _ = e.do(t, "GET", "/version", "nacho", "malapass", nil)
	if code != 401 {
		t.Fatalf("pass mala esperaba 401, tengo %d", code)
	}
}

func TestMiscEndpoints(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	code, body := e.do(t, "GET", "/version", e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("version: %d", code)
	}
	var v struct{ Version string }
	decode(t, body, &v)
	if v.Version != reportedVersion {
		t.Errorf("version: %q", v.Version)
	}

	code, body = e.do(t, "GET", "/status", e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("status: %d", code)
	}
	if !strings.Contains(string(body), "improperlyConfiguredCron") {
		t.Errorf("status sin warnings: %s", body)
	}

	code, body = e.do(t, "GET", "/user", e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("user: %d", code)
	}
	var u struct{ UserID string `json:"userId"` }
	decode(t, body, &u)
	if u.UserID != "nacho" {
		t.Errorf("user: %s", body)
	}
}

func TestFolderCRUD(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	code, body := e.do(t, "POST", "/folders", e.user, e.pass, map[string]string{"name": "Tech"})
	if code != 200 {
		t.Fatalf("crear carpeta: %d %s", code, body)
	}
	var fr struct {
		Folders []store.Folder
	}
	decode(t, body, &fr)
	if len(fr.Folders) != 1 || fr.Folders[0].Name != "Tech" {
		t.Fatalf("respuesta carpeta: %s", body)
	}
	folderID := fr.Folders[0].ID

	if code, _ := e.do(t, "POST", "/folders", e.user, e.pass, map[string]string{"name": "Tech"}); code != 409 {
		t.Fatalf("duplicada esperaba 409, tengo %d", code)
	}
	if code, _ := e.do(t, "POST", "/folders", e.user, e.pass, map[string]string{"name": ""}); code != 422 {
		t.Fatalf("vacía esperaba 422, tengo %d", code)
	}
	// aislamiento: otro usuario no ve la carpeta
	code, body = e.do(t, "GET", "/folders", "otro", "otra1234", nil)
	if code != 200 || strings.Contains(string(body), "Tech") {
		t.Fatalf("aislamiento carpetas roto: %d %s", code, body)
	}

	if code, _ = e.do(t, "PUT", fmt.Sprintf("/folders/%d", folderID), e.user, e.pass, map[string]string{"name": "Tech2"}); code != 204 {
		t.Fatalf("rename: %d", code)
	}
	code, body = e.do(t, "GET", "/folders", e.user, e.pass, nil)
	if !strings.Contains(string(body), "Tech2") {
		t.Fatalf("rename no aplicado: %s", body)
	}
	if code, _ = e.do(t, "DELETE", "/folders/999", e.user, e.pass, nil); code != 404 {
		t.Fatalf("delete inexistente esperaba 404, tengo %d", code)
	}
	if code, _ = e.do(t, "DELETE", fmt.Sprintf("/folders/%d", folderID), e.user, e.pass, nil); code != 204 {
		t.Fatalf("delete: %d", code)
	}
}

func TestFeedAndSyncFlow(t *testing.T) {
	ff := &fakeFetcher{}
	e := newTestEnv(t, ff)

	// carpeta + feed
	code, body := e.do(t, "POST", "/folders", e.user, e.pass, map[string]string{"name": "Lecturas"})
	if code != 200 {
		t.Fatalf("crear carpeta: %d", code)
	}
	var fr struct {
		Folders []store.Folder
	}
	decode(t, body, &fr)
	folderID := fr.Folders[0].ID

	code, body = e.do(t, "POST", "/feeds", e.user, e.pass,
		map[string]any{"url": "https://feed.example/rss", "folderId": folderID})
	if code != 200 {
		t.Fatalf("crear feed: %d %s", code, body)
	}
	var feedResp struct {
		Feeds []struct {
			ID          int64  `json:"id"`
			URL         string `json:"url"`
			Title       string `json:"title"`
			FolderID    *int64 `json:"folderId"`
			UnreadCount int64  `json:"unreadCount"`
		}
		NewestItemID int64 `json:"newestItemId"`
	}
	decode(t, body, &feedResp)
	if len(feedResp.Feeds) != 1 || feedResp.Feeds[0].URL != "https://feed.example/rss" {
		t.Fatalf("respuesta feed: %s", body)
	}
	// la URL de suscripción manda sobre la del XML
	if feedResp.Feeds[0].Title != "Feed https://feed.example/rss" {
		t.Errorf("title feed: %s", body)
	}
	if feedResp.Feeds[0].FolderID == nil || *feedResp.Feeds[0].FolderID != folderID {
		t.Errorf("folderId: %s", body)
	}
	if feedResp.Feeds[0].UnreadCount != int64(len("https://feed.example/rss")) {
		t.Errorf("unreadCount: %d (url len %d)", feedResp.Feeds[0].UnreadCount, len("https://feed.example/rss"))
	}
	newest := feedResp.NewestItemID
	if newest == 0 {
		t.Fatal("newestItemId debería existir con items")
	}
	feedID := feedResp.Feeds[0].ID

	// duplicado → 409; roto → 422
	if code, _ = e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://feed.example/rss"}); code != 409 {
		t.Fatalf("feed dup esperaba 409: %d", code)
	}
	ff.failURLs = map[string]bool{"https://roto.example/rss": true}
	if code, _ = e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://roto.example/rss"}); code != 422 {
		t.Fatalf("feed roto esperaba 422: %d", code)
	}

	// initial sync: unread all + starred
	code, body = e.do(t, "GET", "/items?type=3&getRead=false&batchSize=-1", e.user, e.pass, nil)
	if code != 200 {
		t.Fatalf("items: %d", code)
	}
	var itemsResp struct {
		Items []store.Item
	}
	decode(t, body, &itemsResp)
	n := len(itemsResp.Items)
	if n != len("https://feed.example/rss") {
		t.Fatalf("items iniciales: %d", n)
	}
	// orden por defecto: newest first
	if itemsResp.Items[0].ID != newest {
		t.Errorf("orden: primero %d, esperaba %d", itemsResp.Items[0].ID, newest)
	}
	firstID := itemsResp.Items[0].ID

	// paginación: batchSize 2 + offset por id (cursor: items con id <= cursor)
	// OJO: capturar ids ANTES del siguiente decode (json.Unmarshal reutiliza
	// el backing array del slice y pisaría page1).
	code, body = e.do(t, "GET", "/items?type=3&getRead=true&batchSize=2", e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	page1 := itemsResp.Items
	if len(page1) != 2 {
		t.Fatalf("batchSize: %d", len(page1))
	}
	cursor := page1[len(page1)-1].ID
	secondID := page1[1].ID // page1[0] == firstID (mismo item más nuevo)
	code, body = e.do(t, "GET", fmt.Sprintf("/items?type=3&getRead=true&batchSize=2&offset=%d", cursor), e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	if len(itemsResp.Items) != 2 {
		t.Fatalf("página 2 con batchSize=2 debe tener 2 items, tengo %d", len(itemsResp.Items))
	}
	for _, it := range itemsResp.Items {
		if it.ID > cursor {
			t.Fatalf("offset devuelve items más nuevos que el cursor: %d > %d", it.ID, cursor)
		}
	}

	// marcar leído múltiple: variante spec (POST itemIds)
	if code, _ = e.do(t, "POST", "/items/read/multiple", e.user, e.pass,
		map[string]any{"itemIds": []int64{firstID}}); code != 204 {
		t.Fatalf("read multiple POST: %d", code)
	}
	// variante How To Sync (PUT items)
	if code, _ = e.do(t, "PUT", "/items/read/multiple", e.user, e.pass,
		map[string]any{"items": []int64{secondID}}); code != 204 {
		t.Fatalf("read multiple PUT: %d", code)
	}

	code, body = e.do(t, "GET", "/items?type=3&getRead=false&batchSize=-1", e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	if len(itemsResp.Items) != n-2 {
		t.Fatalf("tras marcar 2 leídos quedan %d, esperaba %d", len(itemsResp.Items), n-2)
	}

	// GET /feeds refleja unreadCount
	code, body = e.do(t, "GET", "/feeds", e.user, e.pass, nil)
	decode(t, body, &feedResp)
	if feedResp.Feeds[0].UnreadCount != int64(n-2) {
		t.Errorf("unreadCount en feeds: %d", feedResp.Feeds[0].UnreadCount)
	}

	// star single + starred sync
	if code, _ = e.do(t, "POST", fmt.Sprintf("/items/%d/star", firstID), e.user, e.pass, nil); code != 204 {
		t.Fatalf("star: %d", code)
	}
	code, body = e.do(t, "GET", "/items?type=2&getRead=true&batchSize=-1", e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	if len(itemsResp.Items) != 1 || !itemsResp.Items[0].Starred {
		t.Fatalf("starred: %s", body)
	}

	// /items/updated refleja cambios de estado (lastModified bump)
	code, body = e.do(t, "GET", "/items/updated?lastModified=1&type=3", e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	if len(itemsResp.Items) != n {
		t.Fatalf("updated tras marcas: %d (esperaba %d)", len(itemsResp.Items), n)
	}

	// feed read con tope newestItemId
	if code, _ = e.do(t, "POST", fmt.Sprintf("/feeds/%d/read", feedID), e.user, e.pass,
		map[string]any{"newestItemId": newest}); code != 204 {
		t.Fatalf("feed read: %d", code)
	}
	code, body = e.do(t, "GET", "/items?type=3&getRead=false", e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	if len(itemsResp.Items) != 0 {
		t.Fatalf("tras feed read quedan %d unread", len(itemsResp.Items))
	}

	// updater API: admin refresca y aparece 1 item nuevo
	fetchesBefore := ff.fetches
	code, _ = e.do(t, "GET", "/feeds/update?userId=nacho&feedId="+fmt.Sprint(feedID), e.user, e.pass, nil)
	if code != 204 {
		t.Fatalf("feeds/update: %d", code)
	}
	if ff.fetches != fetchesBefore+1 {
		t.Fatal("el updater no ha fetcheado")
	}
	code, body = e.do(t, "GET", "/items?type=3&getRead=true&batchSize=1&oldestFirst=false", e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	code, body = e.do(t, "GET", "/items?type=3&getRead=false", e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	if len(itemsResp.Items) != 1 {
		t.Fatalf("tras refresh esperaba 1 unread nuevo, tengo %d", len(itemsResp.Items))
	}

	// updater con usuario no-admin → 401
	if code, _ = e.do(t, "GET", "/feeds/update?userId=nacho&feedId=1", "otro", "otra1234", nil); code != 401 {
		t.Fatalf("updater por no-admin: %d", code)
	}

	// move + rename + delete
	if code, _ = e.do(t, "POST", fmt.Sprintf("/feeds/%d/move", feedID), e.user, e.pass,
		map[string]any{"folderId": nil}); code != 204 {
		t.Fatalf("move: %d", code)
	}
	if code, _ = e.do(t, "POST", fmt.Sprintf("/feeds/%d/rename", feedID), e.user, e.pass,
		map[string]string{"feedTitle": "Mi feed"}); code != 204 {
		t.Fatalf("rename feed: %d", code)
	}
	if code, _ = e.do(t, "DELETE", fmt.Sprintf("/feeds/%d", feedID), e.user, e.pass, nil); code != 204 {
		t.Fatalf("delete feed: %d", code)
	}
	code, body = e.do(t, "GET", "/items?type=3&getRead=true", e.user, e.pass, nil)
	decode(t, body, &itemsResp)
	if len(itemsResp.Items) != 0 {
		t.Fatalf("borrar feed debe llevarse sus items: %d", len(itemsResp.Items))
	}
}

func TestMarkAllRead(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	_, body := e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://a.example/f"})
	var feedResp struct {
		NewestItemID int64 `json:"newestItemId"`
	}
	decode(t, body, &feedResp)

	if code, _ := e.do(t, "POST", "/items/read", e.user, e.pass,
		map[string]any{"newestItemId": feedResp.NewestItemID}); code != 204 {
		t.Fatal("mark all read falló")
	}
	_, body = e.do(t, "GET", "/items?type=3&getRead=false", e.user, e.pass, nil)
	var itemsResp struct {
		Items []store.Item
	}
	decode(t, body, &itemsResp)
	if len(itemsResp.Items) != 0 {
		t.Fatalf("quedan %d unread", len(itemsResp.Items))
	}
}

func TestUserIsolation(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://mine.example/f"})
	_, body := e.do(t, "GET", "/items?type=3&getRead=true", "otro", "otra1234", nil)
	var itemsResp struct {
		Items []store.Item
	}
	decode(t, body, &itemsResp)
	if len(itemsResp.Items) != 0 {
		t.Fatalf("aislamiento roto: otro ve %d items", len(itemsResp.Items))
	}
	// marcar por id ajeno no debe funcionar (404)
	if code, _ := e.do(t, "POST", "/items/1/read", "otro", "otra1234", nil); code != 404 {
		t.Fatalf("marcar item ajeno: %d (esperaba 404)", code)
	}
}

func TestCORSPresent(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})
	req, _ := http.NewRequest("OPTIONS", e.ts.URL+Base+"/feeds", nil)
	req.Header.Set("Origin", "https://pwa.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("CORS ausente en preflight")
	}
}

type errEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// i18n: usuario es → mensajes en español; auto → Accept-Language; nada → en.
func TestErrorI18n(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})

	// usuario con language=es (nacho se crea con es en newTestEnv)
	code, body := e.do(t, "DELETE", "/folders/999", e.user, e.pass, nil)
	if code != 404 {
		t.Fatalf("404 esperado: %d", code)
	}
	var env errEnvelope
	decode(t, body, &env)
	if env.Error.Code != "folder_not_found" {
		t.Errorf("code: %q", env.Error.Code)
	}
	if env.Error.Message != "carpeta no encontrada" {
		t.Errorf("mensaje es: %q", env.Error.Message)
	}

	// auto + Accept-Language en → inglés (DELETE: no existe GET de carpeta suelta)
	req, _ := http.NewRequest("DELETE", e.ts.URL+Base+"/folders/999", nil)
	req.SetBasicAuth("otro", "otra1234")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	decode(t, raw, &env)
	if env.Error.Message != "folder not found" {
		t.Errorf("mensaje en vía Accept-Language: %q", env.Error.Message)
	}

	// 401 sin auth negocia por Accept-Language (es)
	req2, _ := http.NewRequest("GET", e.ts.URL+Base+"/version", nil)
	req2.Header.Set("Accept-Language", "es-ES,es;q=0.9")
	resp2, err := e.client.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	raw2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	decode(t, raw2, &env)
	if env.Error.Code != "unauthorized" || env.Error.Message != "no autorizado" {
		t.Errorf("401 i18n: %s", raw2)
	}

	// /user expone language (extensión propia)
	_, body = e.do(t, "GET", "/user", e.user, e.pass, nil)
	if !strings.Contains(string(body), `"language":"es"`) {
		t.Errorf("/user language: %s", body)
	}
}
