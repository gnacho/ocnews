// Package netguard: mitiga SSRF en las conexiones salientes. Provee un
// http.Transport cuyo DialContext resuelve el host y rechaza destinos a IPs
// privadas, loopback, link-local o metadata cloud antes de conectar. Se usa
// en TODOS los clientes que fetchean URLs controladas por el usuario
// (feeds, favicons, imágenes, extracción de artículo, discover).
package netguard

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// isBlockedIP decide si una IP no debe alcanzarse (SSRF).
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	// IPv4-mapped IPv6 (::ffff:a.b.c.d) → analizar la IPv4 subyacente
	if v4 := ip.To4(); v4 != nil {
		return v4.IsLoopback() || v4.IsPrivate() || v4.IsLinkLocalUnicast() ||
			v4.IsLinkLocalMulticast() || v4.IsUnspecified() || v4.IsMulticast()
	}
	// Metadata de cloud (169.254.169.254) y rangos reservados habituales
	// (ya cubiertos por IsLinkLocalUnicast para 169.254/16).
	return false
}

// SafeTransport devuelve un http.Transport con protección SSRF y un
// timeout de dial razonable.
func SafeTransport() *http.Transport {
	return &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 10 * time.Second}
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("dirección inválida %q: %w", addr, err)
			}
			if ips, err := net.LookupIP(host); err != nil {
				return nil, fmt.Errorf("resolver %q: %w", host, err)
			} else {
				for _, ip := range ips {
					if isBlockedIP(ip) {
						return nil, fmt.Errorf("destino bloqueado (SSRF): %s -> %s", host, ip)
					}
				}
			}
			return d.DialContext(ctx, network, addr)
		},
		MaxIdleConns:        20,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
}

// Client devuelve un http.Client listo para uso con el transporte seguro.
func Client(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: SafeTransport()}
}

// ClientAllowLocal devuelve un client cuyo transporte permite loopback/local.
// SOLO para tests (httptest escucha en 127.0.0.1); nunca en producción.
func ClientAllowLocal(timeout time.Duration) *http.Client {
	t := SafeTransport()
	t.DialContext = (&net.Dialer{Timeout: 10 * time.Second}).DialContext
	return &http.Client{Timeout: timeout, Transport: t}
}
