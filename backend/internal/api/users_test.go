package api

import (
	"fmt"
	"strings"
	"testing"
)

func TestUsersEndpoints(t *testing.T) {
	e := newTestEnv(t, &fakeFetcher{})

	// me + update de perfil
	code, body := e.do(t, "GET", "/api/me", e.user, e.pass, nil)
	if code != 200 || !strings.Contains(string(body), `"role":"admin"`) {
		t.Fatalf("me: %d %s", code, body)
	}
	code, body = e.do(t, "PUT", "/api/me", e.user, e.pass,
		map[string]string{"language": "en", "displayName": "Nacho G"})
	if code != 200 || !strings.Contains(string(body), `"language":"en"`) {
		t.Fatalf("update me: %d %s", code, body)
	}
	// idioma inválido → 422 con code estable
	code, body = e.do(t, "PUT", "/api/me", e.user, e.pass, map[string]string{"language": "fr"})
	if code != 422 || !strings.Contains(string(body), `"code":"invalid_language"`) {
		t.Fatalf("language inválido: %d %s", code, body)
	}
	// restaurar es para no afectar a TestErrorI18n (BD por test, pero por si acaso)
	e.do(t, "PUT", "/api/me", e.user, e.pass, map[string]string{"language": "es"})

	// cambio de contraseña: actual mal → 422; bien → 204 y login con la nueva
	code, _ = e.do(t, "PUT", "/api/me/password", e.user, e.pass,
		map[string]string{"currentPassword": "malapass", "newPassword": "nueva1234"})
	if code != 422 {
		t.Fatalf("password actual mal: %d", code)
	}
	if code, _ = e.do(t, "PUT", "/api/me/password", e.user, e.pass,
		map[string]string{"currentPassword": "pass1234", "newPassword": "nueva1234"}); code != 204 {
		t.Fatalf("cambio password: %d", code)
	}
	if code, _ = e.do(t, "GET", "/api/me", e.user, "pass1234", nil); code != 401 {
		t.Fatalf("password vieja debe fallar: %d", code)
	}
	if code, _ = e.do(t, "GET", "/api/me", e.user, "nueva1234", nil); code != 200 {
		t.Fatalf("password nueva debe valer: %d", code)
	}
	// corta → 422
	if code, _ = e.do(t, "PUT", "/api/me/password", e.user, "nueva1234",
		map[string]string{"currentPassword": "nueva1234", "newPassword": "corta"}); code != 422 {
		t.Fatalf("password corta: %d", code)
	}

	// admin crea usuario; duplicado → 409; username inválido → 422
	code, body = e.do(t, "POST", "/api/users", e.user, "nueva1234",
		map[string]string{"username": "maria", "password": "maria1234", "displayName": "María", "role": "user"})
	if code != 201 || !strings.Contains(string(body), `"username":"maria"`) {
		t.Fatalf("crear usuario: %d %s", code, body)
	}
	if code, _ = e.do(t, "POST", "/api/users", e.user, "nueva1234",
		map[string]string{"username": "maria", "password": "maria1234"}); code != 409 {
		t.Fatalf("duplicado: %d", code)
	}
	if code, _ = e.do(t, "POST", "/api/users", e.user, "nueva1234",
		map[string]string{"username": "M", "password": "maria1234"}); code != 422 {
		t.Fatalf("username inválido: %d", code)
	}

	// no-admin → 401
	if code, _ = e.do(t, "GET", "/api/users", "otro", "otra1234", nil); code != 401 {
		t.Fatalf("users por no-admin: %d", code)
	}
	// maria no puede gestionar usuarios pero puede usar su perfil
	if code, _ = e.do(t, "GET", "/api/me", "maria", "maria1234", nil); code != 200 {
		t.Fatalf("me de maria: %d", code)
	}

	// list
	code, body = e.do(t, "GET", "/api/users", e.user, "nueva1234", nil)
	if code != 200 || !strings.Contains(string(body), "maria") || !strings.Contains(string(body), "nacho") {
		t.Fatalf("list users: %d %s", code, body)
	}

	// update rol de maria a admin y de vuelta; último admin protegido
	var mariaID int64 = 3 // nacho=1, otro=2, maria=3 en BD nueva por test
	code, _ = e.do(t, "PUT", fmt.Sprintf("/api/users/%d", mariaID), e.user, "nueva1234",
		map[string]string{"role": "admin"})
	if code != 200 {
		t.Fatalf("promover maria: %d", code)
	}
	code, body = e.do(t, "PUT", fmt.Sprintf("/api/users/%d", mariaID), e.user, "nueva1234",
		map[string]string{"role": "user"})
	if code != 200 {
		t.Fatalf("degradar maria (hay 2 admins): %d %s", code, body)
	}
	// ahora solo queda nacho como admin → degradarse a sí mismo → 409
	code, _ = e.do(t, "PUT", "/api/users/1", e.user, "nueva1234", map[string]string{"role": "user"})
	if code != 409 {
		t.Fatalf("degradar último admin: %d", code)
	}
	// borrarse a sí mismo → 409
	if code, _ = e.do(t, "DELETE", "/api/users/1", e.user, "nueva1234", nil); code != 409 {
		t.Fatalf("self delete: %d", code)
	}
	// borrar a maria → 204 y su contenido desaparece
	if code, _ = e.do(t, "DELETE", fmt.Sprintf("/api/users/%d", mariaID), e.user, "nueva1234", nil); code != 204 {
		t.Fatalf("delete maria: %d", code)
	}
	if code, _ = e.do(t, "GET", "/api/me", "maria", "maria1234", nil); code != 401 {
		t.Fatalf("maria eliminada no debe autenticar: %d", code)
	}
}

func TestRefreshEndpoint(t *testing.T) {
	ff := &fakeFetcher{}
	e := newTestEnv(t, ff)
	e.do(t, "POST", "/feeds", e.user, e.pass, map[string]string{"url": "https://r.example/f"})
	before := ff.fetches
	code, body := e.do(t, "POST", "/refresh", e.user, e.pass, nil)
	if code != 200 || !strings.Contains(string(body), `"refreshed":1`) {
		t.Fatalf("refresh: %d %s", code, body)
	}
	if ff.fetches != before+1 {
		t.Fatalf("esperaba 1 fetch nuevo, tengo %d", ff.fetches-before)
	}
}
