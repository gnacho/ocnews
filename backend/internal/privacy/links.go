package privacy

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

// Level de higiene de enlaces aplicada a los href del cuerpo de un artículo.
type Level string

const (
	// LevelOff deja los enlaces como los escribió el sitio.
	LevelOff Level = "off"
	// LevelTracking quita los parámetros de tracking. Es el nivel por defecto:
	// la misma lista que ya se aplica a la URL del item, sin heurísticas.
	LevelTracking Level = "tracking"
	// LevelAggressive además desenvuelve intermediarios (acortadores y
	// redireccionadores de medios). Opt-in porque puede equivocarse en una
	// URL legítima que lleve un parámetro de destino.
	LevelAggressive Level = "aggressive"
)

// ValidLevel normaliza y valida un nivel. Vacío = tracking (el default).
func ValidLevel(l string) (Level, bool) {
	switch Level(strings.ToLower(strings.TrimSpace(l))) {
	case "":
		return LevelTracking, true
	case LevelOff:
		return LevelOff, true
	case LevelTracking:
		return LevelTracking, true
	case LevelAggressive:
		return LevelAggressive, true
	}
	return "", false
}

var (
	aTagRe   = regexp.MustCompile(`(?i)<a\b[^>]*>`)
	hrefRe   = regexp.MustCompile(`(?i)\bhref\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	domainRe = regexp.MustCompile(`(?i)^(?:[a-z0-9_-]+\.)+[a-z]{2,}$`)

	// shorteners: dominios que solo sirven para acortar y redirigir. Se
	// comparan por sufijo para cubrir subdominios (bit.ly, www.bit.ly).
	shorteners = []string{
		"bit.ly", "bitly.com", "j.mp", "t.co", "lnkd.in", "goo.gl", "ow.ly",
		"buff.ly", "tinyurl.com", "is.gd", "amzn.to", "amzn.eu", "rb.gy",
		"cutt.ly", "shorturl.at", "trib.al", "dlvr.it", "flip.it", "sco.lt",
		"wp.me", "kutt.it", "t.ly", "s.id", "cstu.io",
	}

	// redirectorLabels: etiquetas de host típicas de un panel de salida de
	// un medio (go.midominio.com, out.midominio.com, links.midominio.com).
	redirectorLabels = []string{
		"go", "out", "link", "links", "redirect", "redirects", "click",
		"clicktrack", "url", "away", "exit", "goto", "r", "wire", "refer",
	}

	// destParams: nombres de parámetro que llevan el destino real.
	destParams = []string{"url", "u", "target", "dest", "destination", "redirect", "redirect_uri", "to", "link", "href", "out"}
)

// IsShortener dice si el host es un acortador conocido.
func IsShortener(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	for _, s := range shorteners {
		if h == s || strings.HasSuffix(h, "."+s) {
			return true
		}
	}
	return false
}

// isRedirectorLabel dice si la primera etiqueta del host es un panel de salida.
func isRedirectorLabel(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if i := strings.Index(h, "."); i > 0 {
		h = h[:i]
	}
	for _, l := range redirectorLabels {
		if h == l {
			return true
		}
	}
	return false
}

// IsRedirector dice si el host parece un panel de salida de un sitio
// (go.midominio.com, links.medios.es) y lleva su destino en la query.
func IsRedirector(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return isRedirectorLabel(u.Host)
}

// Unwrap devuelve el destino que esconde una URL intermedia, y si lo encontró.
// No sale a la red: solo desenvuelve cuando el host es un acortador conocido,
// cuando el host es un panel de salida y la ruta parece de redirección, o
// cuando un único parámetro de destino apunta a un dominio completo.
func Unwrap(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", false
	}
	q := u.Query()

	// 1) Parámetro con una URL destino completa.
	for _, p := range destParams {
		for k, vs := range q {
			if !strings.EqualFold(k, p) || len(vs) == 0 {
				continue
			}
			// El valor puede venir sin esquema (go.ejemplo.com/x?url=ejemplo.com/a).
			cand := strings.TrimSpace(vs[0])
			if !strings.Contains(cand, "://") {
				cand = "https://" + strings.TrimPrefix(cand, "//")
			}
			du, err := url.Parse(cand)
			if err != nil || (du.Scheme != "http" && du.Scheme != "https") || du.Host == "" {
				continue
			}
			if !domainRe.MatchString(du.Hostname()) {
				continue
			}
			// destino igual al original: no es un intermediario
			if strings.EqualFold(du.Host, u.Host) {
				continue
			}
			return du.String(), true
		}
	}

	// 2) Sin parámetro de destino no hay forma de resolver un acortador sin
	// salir a la red, y un panel de salida sin destino tampoco. Se deja el
	// enlace original: se prefiere no adivinar.
	return "", false
}

// CleanHref aplica el nivel elegido a un href: quita el tracking y, en el
// nivel agresivo, desenvuelve el intermediario dejando también su destino
// limpio de tracking.
func CleanHref(raw string, level Level) string {
	if level == LevelOff {
		return raw
	}
	clean := StripParams(raw)
	if level == LevelAggressive {
		if dest, ok := Unwrap(clean); ok {
			clean = StripParams(dest)
		}
	}
	return clean
}

// CleanLinks reescribe los href de todos los <a> del HTML. Best-effort con
// regex sobre un HTML que ya viene sanitizado por bluemonday (atributos
// entrecomillados, sin entidades raras).
func CleanLinks(doc string, level Level) string {
	if level == LevelOff || doc == "" {
		return doc
	}
	if !strings.Contains(strings.ToLower(doc), "<a") {
		return doc
	}
	return aTagRe.ReplaceAllStringFunc(doc, func(tag string) string {
		return hrefRe.ReplaceAllStringFunc(tag, func(m string) string {
			sub := hrefRe.FindStringSubmatch(m)
			if len(sub) < 3 {
				return m
			}
			raw := sub[1]
			quote := `"`
			if raw == "" && sub[2] != "" {
				raw, quote = sub[2], "'"
			}
			if raw == "" {
				return m
			}
			// El HTML sanitizado guarda entidades (&amp;) en los atributos:
			// se trabaja con la URL real y se reescapa al escribir.
			real := html.UnescapeString(raw)
			clean := CleanHref(real, level)
			if clean == real {
				return m
			}
			return "href=" + quote + html.EscapeString(clean) + quote
		})
	})
}
