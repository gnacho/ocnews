# ocnews

Lector RSS/Atom para OpenCloud. Habla la API de News de Nextcloud (v1.3),
así que la app oficial de Nextcloud News para Android funciona contra él, y
se instala como extensión web dentro de la interfaz de OpenCloud.

> **Esto es software beta.** Está en desarrollo activo, las APIs y el
> comportamiento pueden cambiar, y puedes encontrar errores. Pruébalo antes
> en un servidor de pruebas y guarda copia del directorio de datos.

## Por qué existe

Llevo muchos años siendo un gran fan de Nextcloud. Lo uso, me gusta su
ecosistema y las apps que utilizo a diario son apps de Nextcloud. Pero con
los años el stack que hay debajo se me hizo demasiado: PHP, un servidor
web, una base de datos, trabajos de cron, actualizaciones de apps que se
llevan media instalación de Composer por delante. Cada máquina donde lo
monto acaba pareciendo un pequeño centro de datos.

OpenCloud toma el camino contrario: todo lo que hace, incluido lo que
Nextcloud hace en PHP, corre dentro de un único binario de Go. Ligero de
recursos, rápido de instalar, trivial de actualizar, fácil de entender
cuando algo falla. Es más joven, tiene menos apps, una comunidad menor y
menos madurez. Todavía no sé si ganará. Pero la base es un buen punto de
salida, así que lo sigo de cerca, lo pruebo en uso real y aporto lo que me
falta.

Este lector es la primera pieza. El objetivo es trasladar a OpenCloud la
experiencia de Nextcloud que me importa, manteniendo toda la compatibilidad
posible con los clientes de Nextcloud existentes. Por eso el backend
implementa la API REST de News v1.3 en lugar de inventar la suya: si ya
usas la app de Nextcloud News en Android, puedes apuntarla a ocnews y
seguir con tu flujo de siempre.

## Qué funciona hoy

**Backend** (parte de [ocapps](https://github.com/gnacho/ocapps), un único
binario estático de Go, SQLite, sin servicios externos):

- API de News v1.3 completa: carpetas, feeds, items, estado leído/no leído/
  destacado, endpoints de sync (`/items`, `/items/updated`, marcado por
  lotes)
- **Multiusuario**: cada cuenta de OpenCloud obtiene sus propios feeds,
  carpetas y reglas automáticamente. El login se valida contra la Graph API
  del servidor (app passwords para clientes, token de sesión para la web).
- Demonio de feeds: intervalos adaptativos (los feeds tranquilos se espacian,
  los activos se mantienen frescos), backoff exponencial ante errores,
  jitter para que los feeds no se sincronicen a la vez, retención nocturna
  de items leídos antiguos
- Saneamiento HTML de todo body al ingestar (política de lista blanca)
- Proxy firmado de medios: las imágenes y el audio/vídeo de los feeds se ven
  dentro de la app incluso con CSP estrictas, con streaming HTTP Range para
  poder saltar dentro del vídeo
- Extracción del artículo completo: los feeds que solo publican resumen
  pueden descargarse del sitio original y extraerse en el servidor
  (readability), con caché permanente; los feeds que ya traen el texto
  completo se detectan y se dejan tal cual
- Import/export OPML, caché de favicons, rutas updater de la spec
- Mensajes de error en español/inglés negociados por usuario
- WebSub, búsquedas guardadas, reglas de filtro, notificaciones push (ntfy)

**Extensión web** (Vue 3, instalable como cualquier app web de OpenCloud):

- Barra lateral con carpetas, feeds y contadores de no leídos; suscribir,
  renombrar, mover, borrar y marcar como leído por feed o carpeta
- Lista de artículos con filtro no leídos/todos, orden antiguos/recientes,
  paginación
- Panel de lectura con tipografía real, imágenes, reproductores de
  audio/vídeo integrados, destacados, conmutador leído/no leído y enlace al
  original
- Import/export OPML desde la interfaz

**Android**: el cliente oficial de Nextcloud News funciona. Crea un app
token en los ajustes de tu cuenta OpenCloud y configura la app con la URL
del servidor, tu usuario y ese token. Nombre para mostrar, sync, marcado,
enclosures de pódcast: todo probado contra él.

## Arquitectura

Las extensiones web de OpenCloud son solo frontend, así que el motor de
feeds vive en un servicio acompañante. Desde 2026, el backend forma parte
de `ocapps`, un servidor unificado que también cubre Notes y ocphotos:

```
┌─────────────────────────────────────────────────────────────┐
│ OpenCloud Web                                               │
│   web-app-news (extensión Vue 3, en el conmutador de apps)  │
│        │  REST mismo origen, auth de sesión                 │
└────────┼────────────────────────────────────────────────────┘
         │  /index.php/apps/news/api/v1-3/   (reverse proxy)
         ▼
    ocapps backend (Go, binario único)
    • API News v1.3           • demonio de feeds
    • saneamiento             • proxy firmado de medios
    • SQLite (multiusuario)   • auth Graph de OpenCloud
    • API Notes v1.4          • API de fotos
          ▲
          │  Basic auth (app password)
    App de Nextcloud News para Android (sin modificar)
```

## Instalación

### La forma fácil (OpenCloud App Store)

Descarga el zip de la última release desde la [página de releases](https://github.com/gnacho/ocnews/releases):

```bash
# Descarga news-X.Y.Z.zip y extrae a la carpeta de apps de OpenCloud
# (normalmente /var/lib/opencloud/web/assets/apps o /etc/opencloud/web/assets/apps)
```

Añade a `/etc/opencloud/apps.yaml`:

```yaml
news:
  config: {}
```

Reinicia OpenCloud. La app News aparece en el conmutador de aplicaciones.

### Compilar desde fuente

```bash
cd extension
pnpm install && pnpm build
```

Copia `dist/` a la carpeta de apps de OpenCloud como `news/`.

### Backend (ocapps)

El backend vive en el repo [ocapps](https://github.com/gnacho/ocapps).
Consulta su `deploy/README.md` para la guía completa de instalación
(systemd, variables de entorno, snippets de proxy).

Referencia rápida para el proxy:

```nginx
location /index.php/apps/news/ {
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_pass http://127.0.0.1:8096;
}
```

No enrutes `/api/` ni otros prefijos hacia el backend de news; el cliente
web de OpenCloud los usa.

## Paquete de Store / zip de release

Las releases se construyen automáticamente con GitHub Actions cuando se
empuja un tag `news-v*`. El workflow instala dependencias, compila la
extensión, crea el zip con el layout oficial (`news/manifest.json` +
`news/js/remoteEntry-*.mjs`) y lo adjunta a la release de GitHub.

## Capturas

| Lector | Lector (oscuro) |
|---|---|
| ![lector](assets/screenshot-reader-light.png) | ![lector oscuro](assets/screenshot-reader-dark.png) |

| Artículo con imágenes y texto completo | Artículo (oscuro) |
|---|---|
| ![artículo](assets/screenshot-article-light.png) | ![artículo oscuro](assets/screenshot-article-dark.png) |

Tomadas de una instancia de desarrollo local con la interfaz en inglés.

## Hoja de ruta

Ideas en orden aproximado, no promesas:

1. Pulir la extensión: traducciones es/en de la UI, diseño para móvil,
   navegación con teclado
2. Filtros de artículos por feed (reglas por título/cuerpo/palabras clave,
   como la API de filtros de News 28.4)
3. Búsqueda de texto completo en los items
4. Ingeniería de releases: CI, builds con goreleaser, script de instalación
5. Un cliente PWA independiente para navegadores móviles (aparcado; la app
   de Android cubre el caso de uso por ahora)
6. Seguir la evolución de OpenCloud y adaptarse (el SDK de extensiones
   todavía se mueve)

## Contribuir

Issues y pull requests son bienvenidos, en inglés. Ten en cuenta que es un
proyecto de una persona en tiempo libre: con colaboraciones o apoyo podría
crecer más rápido, pero no puedo prometer nada.

## Licencia

AGPL-3.0. Ver [LICENSE](LICENSE).
