// Package auth: validación de credenciales.
//   - Basic user:app-token (clientes News: Android, curl, PWA futura)
//   - Bearer <access-token OIDC> (sesión de la web de OpenCloud: las
//     extensiones usan clientService.httpAuthenticated)
// Ambos se validan contra la Graph API del servidor OpenCloud y se
// resuelven al MISMO shadow user vía oc_id (uuid del IDM).
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/gnacho/ocnews/backend/internal/store"
)

// Credential: exactamente una de las dos formas.
type Credential struct {
	Username, Password string // Basic
	Bearer             string // Bearer access token
}

// Validator decide si una credencial es válida.
type Validator interface {
	Validate(ctx context.Context, cred Credential) (*store.User, bool)
}

// LocalValidator: bcrypt contra la tabla users (solo Basic).
type LocalValidator struct {
	Store *store.Store
}

func (l *LocalValidator) Validate(_ context.Context, cred Credential) (*store.User, bool) {
	if cred.Bearer != "" || cred.Username == "" {
		return nil, false
	}
	u, err := l.Store.GetUserByUsername(cred.Username)
	if err != nil {
		return nil, false
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(cred.Password)) != nil {
		return nil, false
	}
	l.Store.TouchLogin(u.ID)
	return u, true
}

// OpenCloudValidator: Basic o Bearer contra Graph /me; shadow users unificados
// por oc_id; caché positiva con TTL para no golpear el IdP en cada petición.
type OpenCloudValidator struct {
	Store  *store.Store
	URL    string // raíz del servidor OpenCloud
	Client *http.Client
	Log    *slog.Logger

	mu    sync.Mutex
	cache map[string]cacheEntry // sha256(cred) → {userID, expira}

	upsertMu sync.Mutex // serializa upsertShadow (read-then-create)
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
	ID                       string `json:"id"`
	DisplayName              string `json:"displayName"`
	OnPremisesSamAccountName string `json:"onPremisesSamAccountName"`
	UserPrincipalName        string `json:"userPrincipalName"`
	Mail                     string `json:"mail"`
}

func (o *OpenCloudValidator) Validate(ctx context.Context, cred Credential) (*store.User, bool) {
	if cred.Bearer == "" && cred.Username == "" {
		return nil, false
	}
	key := cacheKey(cred)

	o.mu.Lock()
	if e, ok := o.cache[key]; ok && time.Now().Before(e.expiry) {
		o.mu.Unlock()
		if u, err := o.Store.GetUserByID(e.userID); err == nil {
			return u, true
		}
	} else {
		o.mu.Unlock()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.URL+"/graph/v1.0/me", nil)
	if err != nil {
		return nil, false
	}
	if cred.Bearer != "" {
		req.Header.Set("Authorization", "Bearer "+cred.Bearer)
	} else {
		req.SetBasicAuth(cred.Username, cred.Password)
	}
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

	u, err := o.upsertShadow(cred, me)
	if err != nil {
		o.Log.Error("shadow user", "err", err)
		return nil, false
	}
	o.Store.TouchLogin(u.ID)

	o.mu.Lock()
	o.cache[key] = cacheEntry{userID: u.ID, expiry: time.Now().Add(cacheTTL)}
	o.mu.Unlock()
	return u, true
}

// upsertShadow resuelve el shadow user: primero por oc_id (canónico); si no
// existe, por el username del Basic (para sombras previas sin oc_id, se
// vincula); si tampoco, se crea. Con Bearer el username preferido es el
// account name del IDM, no el uuid.
func (o *OpenCloudValidator) upsertShadow(cred Credential, me graphMe) (*store.User, error) {
	o.upsertMu.Lock()
	defer o.upsertMu.Unlock()

	if me.ID != "" {
		if u, err := o.Store.GetUserByOCID(me.ID); err == nil {
			if me.DisplayName != "" && u.DisplayName != me.DisplayName {
				_ = o.Store.UpdateProfile(u.ID, me.DisplayName, u.Language)
			}
			return o.Store.GetUserByOCID(me.ID)
		}
	}
	username := cred.Username
	if username == "" {
		username = me.OnPremisesSamAccountName
		if username == "" {
			username = me.UserPrincipalName
		}
		if username == "" {
			username = me.Mail
		}
		if username == "" {
			username = me.ID
		}
	}
	if u, err := o.Store.GetUserByUsername(username); err == nil && me.ID != "" {
		if u.OCID == "" {
			_ = o.Store.SetUserOCID(u.ID, me.ID)
		}
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
		role = "admin"
	}
	if _, err := o.Store.CreateUserWithOCID(username, me.ID, "shadow:"+string(n), me.DisplayName, role); err != nil {
		if errors.Is(err, store.ErrConflict) {
			if me.ID != "" {
				if u, e := o.Store.GetUserByOCID(me.ID); e == nil {
					return u, nil
				}
			}
			if u, e := o.Store.GetUserByUsername(username); e == nil {
				return u, nil
			}
		}
		return nil, fmt.Errorf("crear shadow: %w", err)
	}
	o.Log.Info("usuario opencloud vinculado", "username", username, "display", me.DisplayName)
	if me.ID != "" {
		return o.Store.GetUserByOCID(me.ID)
	}
	return o.Store.GetUserByUsername(username)
}

func cacheKey(cred Credential) string {
	var material string
	if cred.Bearer != "" {
		material = "bearer\x00" + cred.Bearer
	} else {
		material = "basic\x00" + cred.Username + "\x00" + cred.Password
	}
	h := sha256.Sum256([]byte(material))
	return hex.EncodeToString(h[:])
}

func randomShadowHash() string {
	b := make([]byte, 8)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
