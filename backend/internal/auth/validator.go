// Package auth: validación de credenciales HTTP Basic.
// Dos modos:
//   - Local: bcrypt contra la tabla users (apps standalone).
//   - OpenCloud: Basic user:app-token validado contra la Graph API del
//     servidor OpenCloud (GET /graph/v1.0/me). Los usuarios se crean como
//     shadow en la tabla local (hash inutilizable) al primer login; la
//     autenticación SIEMPRE es externa.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/gnacho/ocnews/backend/internal/store"
)

// Validator decide si unas credenciales Basic son válidas.
type Validator interface {
	Validate(ctx context.Context, username, password string) (*store.User, bool)
}

// LocalValidator: bcrypt contra la tabla users.
type LocalValidator struct {
	Store *store.Store
}

func (l *LocalValidator) Validate(_ context.Context, username, password string) (*store.User, bool) {
	u, err := l.Store.GetUserByUsername(username)
	if err != nil {
		return nil, false
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, false
	}
	l.Store.TouchLogin(u.ID)
	return u, true
}

// OpenCloudValidator: Basic user:app-token contra Graph /me del servidor
// OpenCloud. Caché positiva en memoria (TTL) para no golpear el IdP en cada
// petición; los usuarios se materializan como shadow al primer login.
type OpenCloudValidator struct {
	Store  *store.Store
	URL    string // raíz del servidor OpenCloud, p.ej. https://drive.example
	Client *http.Client
	Log    *slog.Logger

	mu    sync.Mutex
	cache map[string]cacheEntry // sha256(user:pass) → {userID, expira}
}

type cacheEntry struct {
	userID int64
	expiry time.Time
}

const cacheTTL = 5 * time.Minute

func NewOpenCloudValidator(st *store.Store, url string, log *slog.Logger) *OpenCloudValidator {
	return &OpenCloudValidator{
		Store:  st,
		URL:    strings.TrimRight(url, "/"),
		Client: &http.Client{Timeout: 10 * time.Second},
		Log:    log,
		cache:  map[string]cacheEntry{},
	}
}

// graphMe: campos de GET /graph/v1.0/me que usamos.
type graphMe struct {
	ID                         string `json:"id"`
	DisplayName                string `json:"displayName"`
	OnPremisesSamAccountName   string `json:"onPremisesSamAccountName"`
	UserPrincipalName          string `json:"userPrincipalName"`
}

func (o *OpenCloudValidator) Validate(ctx context.Context, username, password string) (*store.User, bool) {
	key := cacheKey(username, password)

	o.mu.Lock()
	if e, ok := o.cache[key]; ok && time.Now().Before(e.expiry) {
		o.mu.Unlock()
		u, err := o.Store.GetUserByID(e.userID)
		if err == nil {
			return u, true
		}
	} else {
		o.mu.Unlock()
	}

	// validar contra OpenCloud: reenviamos el MISMO Basic
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.URL+"/graph/v1.0/me", nil)
	if err != nil {
		return nil, false
	}
	req.SetBasicAuth(username, password)
	resp, err := o.Client.Do(req)
	if err != nil {
		o.Log.Warn("opencloud graph /me inalcanzable", "err", err)
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	var me graphMe
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return nil, false
	}

	u, err := o.upsertShadow(username, me)
	if err != nil {
		o.Log.Error("crear shadow user", "err", err)
		return nil, false
	}
	o.Store.TouchLogin(u.ID)

	o.mu.Lock()
	o.cache[key] = cacheEntry{userID: u.ID, expiry: time.Now().Add(cacheTTL)}
	o.mu.Unlock()
	return u, true
}

// upsertShadow materializa el usuario OpenCloud en la tabla local.
// password_hash = hash inutilizable: el login local NUNCA funciona.
func (o *OpenCloudValidator) upsertShadow(username string, me graphMe) (*store.User, error) {
	if u, err := o.Store.GetUserByUsername(username); err == nil {
		// refrescar display name si cambió
		if me.DisplayName != "" && u.DisplayName != me.DisplayName {
			_ = o.Store.UpdateProfile(u.ID, me.DisplayName, u.Language)
		}
		return o.Store.GetUserByUsername(username)
	}
	n, err := bcrypt.GenerateFromPassword([]byte(randomShadowHash()), bcrypt.MinCost)
	if err != nil {
		return nil, err
	}
	role := "user"
	if users, err := o.Store.CountUsers(); err == nil && users == 0 {
		role = "admin" // el primer usuario de la instancia es admin
	}
	id, err := o.Store.CreateUser(username, "shadow:"+string(n), me.DisplayName, role)
	if err != nil {
		return nil, fmt.Errorf("shadow user: %w", err)
	}
	o.Log.Info("usuario opencloud vinculado", "username", username, "display", me.DisplayName)
	return o.Store.GetUserByID(id)
}

func cacheKey(username, password string) string {
	h := sha256.Sum256([]byte(username + "\x00" + password))
	return hex.EncodeToString(h[:])
}

func randomShadowHash() string {
	b := make([]byte, 8)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
