package auth

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gnacho/ocnews/backend/internal/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// fakeGraph: servidor OpenCloud de mentira que solo valida un par fijo.
type fakeGraph struct {
	mu       sync.Mutex
	meHits   int
	password string // la password válida para "nacho"
}

func (f *fakeGraph) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.meHits++
		f.mu.Unlock()
		if r.URL.Path != "/graph/v1.0/me" {
			http.NotFound(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "nacho" || pass != f.password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"uuid-1234","displayName":"Nacho Real","onPremisesSamAccountName":"nacho"}`)
	})
}

func TestOpenCloudValidatorShadowUser(t *testing.T) {
	st := newStore(t)
	fg := &fakeGraph{password: "app-token-xyz"}
	ts := httptest.NewServer(fg.handler(t))
	t.Cleanup(ts.Close)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	v := NewOpenCloudValidator(st, ts.URL, log)

	// credenciales malas → false
	if u, ok := v.Validate(context.Background(), "nacho", "malapass"); ok || u != nil {
		t.Fatal("pass mala debe fallar")
	}

	// credenciales buenas → shadow creado con displayName del Graph
	u, ok := v.Validate(context.Background(), "nacho", "app-token-xyz")
	if !ok || u == nil {
		t.Fatal("login válido rechazado")
	}
	if u.Username != "nacho" || u.DisplayName != "Nacho Real" {
		t.Fatalf("shadow mal: %+v", u)
	}
	if u.Role != "admin" {
		t.Fatalf("primer usuario debe ser admin: %s", u.Role)
	}

	// el login LOCAL contra el shadow NUNCA funciona (hash inutilizable)
	lv := &LocalValidator{Store: st}
	if _, ok := lv.Validate(context.Background(), "nacho", "app-token-xyz"); ok {
		t.Fatal("el shadow no debe autenticar localmente")
	}

	// segunda validación misma pass → cache (sin hit nuevo al Graph)
	before := fg.meHits
	u2, ok := v.Validate(context.Background(), "nacho", "app-token-xyz")
	if !ok || u2.ID != u.ID {
		t.Fatal("revalidación falló")
	}
	if fg.meHits != before {
		t.Fatalf("cache no aplicada: %d hits nuevos", fg.meHits-before)
	}

	// password rotada → el token NUEVO valida (crea entrada de cache nueva);
	// el viejo sigue vivo hasta agotar el TTL (tradeoff documentado de la
	// caché positiva: revocación efectiva en <= cacheTTL)
	fg.mu.Lock()
	fg.password = "nueva"
	fg.mu.Unlock()
	if _, ok := v.Validate(context.Background(), "nacho", "nueva"); !ok {
		t.Fatal("token nuevo debe validar")
	}
	if _, ok := v.Validate(context.Background(), "nacho", "app-token-xyz"); !ok {
		t.Log("nota: token viejo aún en cache (TTL); esperado")
	}
}
