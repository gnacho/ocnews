// Package privacy: higiene de privacidad aplicada al ingesto de feeds —
// elimina parámetros de tracking de las URLs y pixel trackers del HTML.
package privacy

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// trackingPrefixes: prefijos de parámetros de query que se descartan
// (utm_*, fbclid, gclid…). Se comparan en minúsculas.
var trackingPrefixes = []string{
	"utm_",     // Google Analytics / campañas
	"fbclid",   // Facebook
	"gclid",    // Google Ads
	"gclsrc",   // Google Ads
	"msclkid",  // Microsoft Clarity
	"dclid",    // DoubleClick
	"twclid",   // Twitter/X
	"igshid",   // Instagram
	"mc_cid",   // MailChimp
	"mc_eid",   // MailChimp
	"_hsenc",   // HubSpot
	"_hsmi",    // HubSpot
	"oly_enc_id", // Outbrain
	"oly_anon_id", // Outbrain
	"vero_id",  // Vero
	"pk_source", // Matomo
	"pk_medium",
	"pk_campaign",
	"pk_keyword",
	"pk_content",
	"mtm_source", // Matomo Tag Manager
	"mtm_medium",
	"mtm_campaign",
	"mtm_keyword",
	"mtm_content",
	"yclid",   // Yandex
	"zanpid",  // Zanox
	"s_cid",   // Twitter
	"sfmc_id", // Salesforce
	"wickedid", // Bing
	"mkt_tok", // Marketo
	"cmpid",   // varios
}

// StripParams elimina los parámetros de tracking de una URL conservando el
// resto de la query. Devuelve la URL original si no se puede parsear.
func StripParams(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return raw
	}
	q := u.Query()
	if len(q) == 0 {
		return raw
	}
	changed := false
	for _, p := range trackingPrefixes {
		for k := range q {
			if strings.HasPrefix(strings.ToLower(k), p) {
				q.Del(k)
				changed = true
			}
		}
	}
	if !changed {
		return raw
	}
	u.RawQuery = q.Encode()
	return u.String()
}

var (
	imgTagRe   = regexp.MustCompile(`(?i)<img\b[^>]*>`)
	attrValRe  = regexp.MustCompile(`(?i)([a-z][a-z0-9-]*)\s*=\s*("([^"]*)"|'([^']*)')`)
	pixelSrcRe = regexp.MustCompile(`(?i)^(data:image/(gif|png|webp|jpeg);base64,[a-z0-9+/=]{0,96}|about:blank)$`)
)

// attr devuelve el valor del atributo name de un tag (entrecomillado).
func attr(tag, name string) string {
	for _, m := range attrValRe.FindAllStringSubmatch(tag, -1) {
		if strings.EqualFold(m[1], name) {
			if m[3] != "" {
				return m[3]
			}
			return m[4]
		}
	}
	return ""
}

// IsTrackingPixel decide si un tag <img> es un pixel de tracking: sin src
// real, src transparente o dimensiones 1x1.
func IsTrackingPixel(tag string) bool {
	lower := strings.ToLower(tag)
	src := attr(tag, "src")
	if src == "" {
		// sin src ni srcset no es una imagen real
		return attr(tag, "srcset") == ""
	}
	if pixelSrcRe.MatchString(src) {
		return true
	}
	small := 0
	if w := attrInt(lower, "width"); w >= 0 && w <= 2 {
		small++
	}
	if h := attrInt(lower, "height"); h >= 0 && h <= 2 {
		small++
	}
	if small == 2 {
		return true
	}
	return strings.Contains(lower, "width:1px") && strings.Contains(lower, "height:1px")
}

// attrInt parsea un atributo numérico; -1 si no está o no es numérico.
func attrInt(lowerTag, name string) int {
	v := attr(lowerTag, name)
	if v == "" {
		return -1
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return -1
	}
	return n
}

// RemovePixels elimina los <img> que son trackers y limpia parámetros de
// tracking de los src de las imágenes restantes. Best-effort con regex.
func RemovePixels(html string) string {
	if !strings.Contains(strings.ToLower(html), "<img") {
		return html
	}
	return imgTagRe.ReplaceAllStringFunc(html, func(tag string) string {
		if IsTrackingPixel(tag) {
			return ""
		}
		src := attr(tag, "src")
		if src == "" {
			return tag
		}
		clean := StripParams(src)
		if clean == src {
			return tag
		}
		return strings.Replace(tag, src, clean, 1)
	})
}
