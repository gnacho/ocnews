package imgproxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// mediaServer simula un origen de mp3 con soporte Range (como un CDN real).
func mediaServer(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			return
		}
		if rng := r.Header.Get("Range"); rng != "" {
			// solo "bytes=s-e"
			rng = strings.TrimPrefix(rng, "bytes=")
			parts := strings.SplitN(rng, "-", 2)
			start, _ := strconv.Atoi(parts[0])
			end := len(content) - 1
			if parts[1] != "" {
				end, _ = strconv.Atoi(parts[1])
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Header().Set("Content-Range", "bytes "+strconv.Itoa(start)+"-"+strconv.Itoa(end)+"/"+strconv.Itoa(len(content)))
			w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
			w.WriteHeader(http.StatusPartialContent)
			io.WriteString(w, content[start:end+1])
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Accept-Ranges", "bytes")
		io.WriteString(w, content)
	}))
}

func TestMediaStreamingWithRange(t *testing.T) {
	p := newProxy(t)
	content := strings.Repeat("mp3data-", 100) // 800 bytes
	ts := mediaServer(t, content)
	defer ts.Close()

	u := ts.URL + "/podcast.mp3"
	sig := p.Sign(u)

	// petición completa (el <audio> la hace al empezar)
	w := httptest.NewRecorder()
	p.Serve(w, httptest.NewRequest("GET", "/img?u="+url.QueryEscape(u)+"&t="+sig, nil))
	if w.Code != 200 || w.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("full: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
	if w.Body.String() != content {
		t.Fatal("contenido media mal")
	}
	if w.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatal("falta Accept-Ranges (el seek del reproductor lo necesita)")
	}

	// seek: Range bytes=100-199 → 206 + trozo exacto
	req := httptest.NewRequest("GET", "/img?u="+url.QueryEscape(u)+"&t="+sig, nil)
	req.Header.Set("Range", "bytes=100-199")
	w = httptest.NewRecorder()
	p.Serve(w, req)
	if w.Code != http.StatusPartialContent {
		t.Fatalf("range: %d", w.Code)
	}
	if w.Body.String() != content[100:200] {
		t.Fatalf("trozo mal: %q", w.Body.String()[:8])
	}
	if cr := w.Header().Get("Content-Range"); !strings.HasPrefix(cr, "bytes 100-199/") {
		t.Fatalf("content-range: %s", cr)
	}
}

func TestMediaNotCached(t *testing.T) {
	p := newProxy(t)
	ts := mediaServer(t, "short")
	defer ts.Close()
	u := ts.URL + "/x.mp3"
	p.Serve(httptest.NewRecorder(), httptest.NewRequest("GET", "/img?u="+url.QueryEscape(u)+"&t="+p.Sign(u), nil))
	if _, err := osStat(filepathJoin(p.dir, cachePath(u))); !isNotExist(err) {
		t.Fatal("la media no debe cachearse en disco")
	}
}

func osStat(p string) (os.FileInfo, error) { return os.Stat(p) }
func isNotExist(err error) bool { return os.IsNotExist(err) }
func filepathJoin(a, b string) string { return filepath.Join(a, b) }
