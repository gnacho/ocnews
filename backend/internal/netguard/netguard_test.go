package netguard

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestSafeTransportBlocksPrivate: el transporte seguro rechaza loopback/local.
func TestSafeTransportBlocksPrivate(t *testing.T) {
	// servidor en loopback: con el transporte seguro NO debe conectar
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := Client(3 * time.Second)
	_, err := c.Get(srv.URL)
	if err == nil {
		t.Fatalf("el transporte seguro conectó a loopback %s (SSRF no bloqueado)", srv.URL)
	}
	// el client allow-local SÍ debe conectar
	lc := ClientAllowLocal(3 * time.Second)
	resp, err := lc.Get(srv.URL)
	if err != nil {
		t.Fatalf("allow-local no conectó a loopback: %v", err)
	}
	resp.Body.Close()
}

// TestIsBlockedIP cubre los rangos críticos.
func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip   string
		block bool
	}{
		{"127.0.0.1", true},     // loopback
		{"10.0.0.5", true},      // RFC1918
		{"192.168.1.1", true},   // RFC1918
		{"172.16.0.1", true},    // RFC1918
		{"169.254.169.254", true}, // metadata
		{"::1", true},           // loopback v6
		{"8.8.8.8", false},      // público
		{"1.1.1.1", false},      // público
	}
	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if got := isBlockedIP(ip); got != c.block {
			t.Errorf("%s: esperaba block=%v, tengo %v", c.ip, c.block, got)
		}
	}
}
