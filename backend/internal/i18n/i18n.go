// Package i18n: catálogo ES/EN de los mensajes de la API.
// Negociación: users.language (si es es|en) → Accept-Language → en.
package i18n

import (
	"strings"
)

// Lang es el idioma resuelto para una petición.
type Lang string

const (
	ES Lang = "es"
	EN Lang = "en"
)

// Catálogo de mensajes por clave estable (el code del envelope de error).
var messages = map[string]map[Lang]string{
	"unauthorized": {
		ES: "no autorizado",
		EN: "unauthorized",
	},
	"admin_required": {
		ES: "requiere permisos de administrador",
		EN: "administrator role required",
	},
	"folder_not_found": {
		ES: "carpeta no encontrada",
		EN: "folder not found",
	},
	"feed_not_found": {
		ES: "feed no encontrado",
		EN: "feed not found",
	},
	"item_not_found": {
		ES: "elemento no encontrado",
		EN: "item not found",
	},
	"user_not_found": {
		ES: "usuario no encontrado",
		EN: "user not found",
	},
	"folder_exists": {
		ES: "ya existe una carpeta con ese nombre",
		EN: "a folder with this name already exists",
	},
	"feed_exists": {
		ES: "ya sigues este feed",
		EN: "you already follow this feed",
	},
	"invalid_folder_name": {
		ES: "nombre de carpeta no válido",
		EN: "invalid folder name",
	},
	"invalid_title": {
		ES: "título no válido",
		EN: "invalid title",
	},
	"invalid_url": {
		ES: "url no válida",
		EN: "invalid url",
	},
	"invalid_body": {
		ES: "cuerpo de la petición no válido",
		EN: "invalid request body",
	},
	"invalid_type": {
		ES: "tipo no válido",
		EN: "invalid type",
	},
	"feed_unreadable": {
		ES: "no se ha podido leer el feed",
		EN: "could not read the feed",
	},
	"invalid_username": {
		ES: "nombre de usuario no válido",
		EN: "invalid username",
	},
	"invalid_password": {
		ES: "contraseña no válida (mínimo 8 caracteres)",
		EN: "invalid password (minimum 8 characters)",
	},
	"invalid_language": {
		ES: "idioma no válido",
		EN: "invalid language",
	},
	"invalid_role": {
		ES: "rol no válido",
		EN: "invalid role",
	},
	"user_exists": {
		ES: "ya existe una usuaria o usuario con ese nombre",
		EN: "a user with this name already exists",
	},
	"cannot_delete_self": {
		ES: "no puedes eliminar tu propia cuenta",
		EN: "you cannot delete your own account",
	},
	"cannot_delete_last_admin": {
		ES: "no se puede quitar el último administrador",
		EN: "cannot remove the last administrator",
	},
	"internal_error": {
		ES: "error interno",
		EN: "internal error",
	},
	"full_unavailable": {
		ES: "no se ha podido extraer el artículo completo",
		EN: "full article extraction unavailable",
	},
	"no_feeds_found": {
		ES: "no se han encontrado feeds en esa página",
		EN: "no feeds found on that page",
	},
	"invalid_query": {
		ES: "búsqueda no válida",
		EN: "invalid search query",
	},
	"invalid_theme": {
		ES: "tema no válido",
		EN: "invalid theme",
	},
	"invalid_reader_width": {
		ES: "ancho del lector no válido",
		EN: "invalid reader width",
	},
	"invalid_feed_interval": {
		ES: "intervalo de refresco no válido (5-1440 minutos)",
		EN: "invalid refresh interval (5-1440 minutes)",
	},
	"invalid_retention": {
		ES: "retención no válida (0-3650 días)",
		EN: "invalid retention (0-3650 days)",
	},
	"invalid_reader_font": {
		ES: "fuente del lector no válida",
		EN: "invalid reader font",
	},
	"invalid_reader_font_size": {
		ES: "tamaño de fuente no válido (13-20 px)",
		EN: "invalid font size (13-20 px)",
	},
	"feed_auth_required": {
		ES: "este feed requiere autenticación (usuario y contraseña)",
		EN: "this feed requires authentication (username and password)",
	},
	"invalid_credentials": {
		ES: "credenciales no válidas",
		EN: "invalid credentials",
	},
	"invalid_rule": {
		ES: "regla inválida (formato Campo=regex; campos: EntryTitle, EntryURL, EntryAuthor, EntryContent, EntryDate)",
		EN: "invalid rule (format Field=regex; fields: EntryTitle, EntryURL, EntryAuthor, EntryContent, EntryDate)",
	},
	"invalid_search": {
		ES: "nombre y consulta de búsqueda son obligatorios",
		EN: "search name and query are required",
	},
	"search_not_found": {
		ES: "búsqueda guardada no encontrada",
		EN: "saved search not found",
	},
	"rule_not_found": {
		ES: "regla no encontrada",
		EN: "rule not found",
	},
}

// T devuelve el mensaje traducido; claves desconocidas caen en internal_error.
func T(lang Lang, key string) string {
	if m, ok := messages[key]; ok {
		if s, ok := m[lang]; ok && s != "" {
			return s
		}
		return m[EN]
	}
	return messages["internal_error"][lang]
}

// Negotiate resuelve el idioma: preferencia explícita del usuario y,
// si es auto, la cabecera Accept-Language; por defecto inglés.
func Negotiate(userLang, acceptLanguage string) Lang {
	switch strings.ToLower(strings.TrimSpace(userLang)) {
	case "es":
		return ES
	case "en":
		return EN
	}
	for _, part := range strings.Split(acceptLanguage, ",") {
		tag := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		switch {
		case strings.HasPrefix(tag, "es"):
			return ES
		case strings.HasPrefix(tag, "en"):
			return EN
		}
	}
	return EN
}
