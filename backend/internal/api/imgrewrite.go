package api

import (
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gnacho/ocnews/backend/internal/privacy"
	"github.com/gnacho/ocnews/backend/internal/store"
)

var imgSrcRe = regexp.MustCompile(`src="(https?://[^"]+)"`)

// linkHygiene lee el nivel de higiene de enlaces del usuario de la petición.
// Vacío o error = tracking, que es el default y no tiene falsos positivos.
func (s *Server) linkHygiene(r *http.Request) privacy.Level {
	u := user(r)
	if u == nil {
		return privacy.LevelTracking
	}
	v, err := s.store.GetUserSetting(u.ID, "link_hygiene")
	if err != nil {
		return privacy.LevelTracking
	}
	lvl, ok := privacy.ValidLevel(v)
	if !ok {
		return privacy.LevelTracking
	}
	return lvl
}

// ownerLinkHygiene resuelve el nivel de higiene del dueño de un feed: la
// vista pública de un artículo compartido se limpia con el ajuste de quien lo
// comparte, no con el del visitante.
func (s *Server) ownerLinkHygiene(feedID int64) privacy.Level {
	f, err := s.store.GetFeedByID(feedID)
	if err != nil || f == nil {
		return privacy.LevelTracking
	}
	v, err := s.store.GetUserSetting(f.UserID, "link_hygiene")
	if err != nil {
		return privacy.LevelTracking
	}
	lvl, ok := privacy.ValidLevel(v)
	if !ok {
		return privacy.LevelTracking
	}
	return lvl
}

// rewriteImages transforma los src http(s) del body hacia el proxy firmado
// (la CSP del host impide imágenes externas; el navegador sí puede cargar
// del propio dominio). El HTML ya viene sanitizado: los atributos src son
// URLs limpias. El & del query va escapado como &amp; para HTML válido.
// u viaja con url.QueryEscape (escapa % → %25): q.Get() del proxy recupera
// la URL EXACTA que se firmó (con sus %2C etc. intactos).
func (s *Server) rewriteImages(body string) string {
	if body == "" || s.imgs == nil {
		return body
	}
	return imgSrcRe.ReplaceAllStringFunc(body, func(m string) string {
		sub := imgSrcRe.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		// el HTML sanitizado guarda entities en atributos (&amp;): la URL
		// real que se firma y se fetchea es la desescapada
		u := html.UnescapeString(sub[1])
		return `src="` + Base + `/img?u=` + url.QueryEscape(u) +
			`&amp;t=` + s.imgs.Sign(u) + `"`
	})
}

// rewriteBody prepara el cuerpo de un artículo para el cliente: primero la
// higiene de enlaces (los href ya no llevan tracking ni intermediarios) y
// después el proxy de imágenes. En ese orden: limpiar hrefs no toca los src.
func (s *Server) rewriteBody(body string, level privacy.Level) string {
	return s.rewriteImages(privacy.CleanLinks(body, level))
}

// rewriteAll aplica el proxy a todos los items de una respuesta: bodies
// (imágenes y media embebida) y enclosures de audio/vídeo/imagen.
func (s *Server) rewriteAll(items []store.Item, level privacy.Level) {
	for i := range items {
		items[i].Body = s.rewriteBody(items[i].Body, level)
		if items[i].EnclosureLink != nil && items[i].EnclosureMime != nil {
			m := *items[i].EnclosureMime
			if s.mediaMime(m) && strings.HasPrefix(*items[i].EnclosureLink, "http") {
				u := html.UnescapeString(*items[i].EnclosureLink)
				proxied := Base + "/img?u=" + url.QueryEscape(u) + "&t=" + s.imgs.Sign(u)
				items[i].EnclosureLink = &proxied
			}
		}
	}
}

func (s *Server) mediaMime(m string) bool {
	return strings.HasPrefix(m, "audio/") || strings.HasPrefix(m, "video/") ||
		strings.HasPrefix(m, "image/")
}

// rewriteFavicon apunta al endpoint público del favicon servido desde el cache
// del backend (/favicon/{urlHash}) en vez de firmar la URL del origen. Así el
// cache es la única fuente (no se re-fetchea por instancia) y el <img> del
// navegador no necesita auth (la ruta es pública, solo lee de disco).
func (s *Server) rewriteFavicon(urlHash string) string {
	if urlHash == "" || s.favicons == nil {
		return ""
	}
	return Base + "/favicon/" + urlHash
}
