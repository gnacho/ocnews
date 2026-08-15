package api

import (
	"html"
	"net/url"
	"regexp"

	"github.com/gnacho/ocnews/backend/internal/store"
)

var imgSrcRe = regexp.MustCompile(`src="(https?://[^"]+)"`)

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

// rewriteAll aplica el proxy a todos los items de una respuesta.
func (s *Server) rewriteAll(items []store.Item) {
	for i := range items {
		items[i].Body = s.rewriteImages(items[i].Body)
	}
}
