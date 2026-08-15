// Package sanitize: limpieza HTML de los cuerpos de items antes de
// persistirlos. Política conservadora: se conserva el formato habitual de
// los artículos (párrafos, cabeceras, listas, imágenes, enlaces, tablas)
// y se elimina todo lo ejecutable o peligroso (script, style, iframes,
// handlers on*, javascript:). Bluemonday reescribe el HTML a válido.
package sanitize

import (
	"github.com/microcosm-cc/bluemonday"
)

var policy *bluemonday.Policy

func init() {
	p := bluemonday.NewPolicy()

	// Elementos de texto y estructura habituales en artículos.
	p.AllowElements("p", "br", "hr", "blockquote", "pre", "code",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"ul", "ol", "li", "dl", "dt", "dd",
		"strong", "b", "em", "i", "u", "s", "sub", "sup", "mark", "small",
		"table", "thead", "tbody", "tr", "th", "td", "caption", "colgroup", "col",
		"figure", "figcaption", "picture", "source", "details", "summary", "abbr", "kbd")

	// Enlaces: solo esquemas seguros, target/rel impuestos por el cliente.
	p.AllowAttrs("href", "title").OnElements("a")
	p.AllowURLSchemes("http", "https", "mailto")
	p.AllowRelativeURLs(true)

	// Imágenes: src explícito, alt/title informativos.
	p.AllowAttrs("src", "alt", "title", "width", "height", "loading").OnElements("img")
	p.AllowAttrs("src", "srcset", "media").OnElements("source")

	// Multimedia embebida básica (podcasts/vídeo de terceros quedan como
	// enclosure en el contrato; aquí solo audio/vídeo directos).
	p.AllowAttrs("src", "controls", "preload").OnElements("audio", "video")
	p.AllowAttrs("src").OnElements("track")

	// Código: lenguajes como valor de clase restringido.
	p.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("code", "pre", "span")

	policy = p
}

// Body limpia el HTML de un cuerpo de artículo. Vacío entra, vacío sale.
func Body(html string) string {
	if html == "" {
		return ""
	}
	return policy.Sanitize(html)
}
