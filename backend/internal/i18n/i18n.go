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
	"internal_error": {
		ES: "error interno",
		EN: "internal error",
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
