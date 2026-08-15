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

// fakeGraph: OpenCloud de mentira; valida Basic (user fijo) y Bearer (token fijo),
// ambos resuelven al MISMO /me (mismo oc_id).
type fakeGraph struct {
	mu       sync.Mutex
	meHits   int
	password string
	bearer   string
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
		ok := false
		if user, pass, b := r.BasicAuth(); b {
			ok = user == "nacho" && pass == f.password
		} else if h := r.Header.Get("Authorization"); h != "" {
			ok = h == "Bearer "+f.bearer
		}
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"uuid-1234","displayName":"Nacho Real","onPremisesSamAccountName":"nacho"}`)
	})
}

func TestOpenCloudValidatorBasicAndBearerUnify(t *testing.T) {
	st := newStore(t)
	fg := &fakeGraph{password: "app-token-xyz", bearer: "oidc-access-tok"}
	ts := httptest.NewServer(fg.handler(t))
	t.Cleanup(ts.Close)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	v := NewOpenCloudValidator(st, ts.URL, log)

	// Basic malo → false
	if u, ok := v.Validate(context.Background(), Credential{Username: "nacho", Password: "malapass"}); ok || u != nil {
		t.Fatal("pass mala debe fallar")
	}
	// Basic bueno → shadow admin con displayName del Graph
	uBasic, ok := v.Validate(context.Background(), Credential{Username: "nacho", Password: "app-token-xyz"})
	if !ok || uBasic.Username != "nacho" || uBasic.DisplayName != "Nacho Real" || uBasic.OCID != "uuid-1234" {
		t.Fatalf("shadow Basic mal: %+v", uBasic)
	}
	if uBasic.Role != "admin" {
		t.Fatalf("primer usuario debe ser admin: %s", uBasic.Role)
	}

	// Bearer (sesión web) → MISMO usuario (oc_id), sin duplicar
	uBearer, ok := v.Validate(context.Background(), Credential{Bearer: "oidc-access-tok"})
	if !ok {
		t.Fatal("bearer válido rechazado")
	}
	if uBearer.ID != uBasic.ID {
		t.Fatalf("Basic y Bearer deben resolver al mismo shadow: %d vs %d", uBasic.ID, uBearer.ID)
	}

	// Bearer malo → false
	if _, ok := v.Validate(context.Background(), Credential{Bearer: "caducado"}); ok {
		t.Fatal("bearer malo debe fallar")
	}

	// cache: misma credencial no golpea el Graph de nuevo
	before := fg.meHits
	if _, ok := v.Validate(context.Background(), Credential{Bearer: "oidc-access-tok"}); !ok {
		t.Fatal("revalidación bearer falló")
	}
	if fg.meHits != before {
		t.Fatalf("cache no aplicada: %d hits nuevos", fg.meHits-before)
	}

	// LocalValidator NUNCA acepta Bearer ni el hash shadow
	lv := &LocalValidator{Store: st}
	if _, ok := lv.Validate(context.Background(), Credential{Bearer: "oidc-access-tok"}); ok {
		t.Fatal("local no debe aceptar bearer")
	}
	if _, ok := lv.Validate(context.Background(), Credential{Username: "nacho", Password: "app-token-xyz"}); ok {
		t.Fatal("el shadow no debe autenticar localmente")
	}
}
