# ocnews

RSS/Atom feed reader for OpenCloud, modeled on the Nextcloud News app
(https://apps.nextcloud.com/apps/news) including a webapp clone of its Android
client (https://github.com/nextcloud/news-android).

## Architecture

OpenCloud Web extensions are frontend-only (Vue 3 + Vite + module federation),
installed by copying `dist/` into `$OC_DATA_DIR/web/assets/apps`. They cannot
ship a backend, so the feed engine lives in a separate service.

```
┌─────────────────────────────────────────────────────────┐
│  OpenCloud Web (host)                                   │
│   ┌─────────────────────────┐                           │
│   │ web-app-news (extension)│  Vue 3, clone of the      │
│   │                        │  News web UI               │
│   └───────────┬─────────────┘                           │
│               │ REST (token auth)                       │
└───────────────┼─────────────────────────────────────────┘
                │
   ┌────────────▼────────────────┐      ┌───────────────────────┐
   │ news-backend (Go service)   │      │ news-webapp (PWA)     │
   │  • News REST API v1.3       │◄────►│  Material 3 clone of   │
   │  • feed fetcher daemon      │ REST │  the Android client    │
   │  • RSS/Atom + sanitization  │      │  (login, lists, reader)│
   │  • SQLite, multi-user       │      └───────────────────────┘
   └─────────────────────────────┘
```

## Layout

- `backend/`  — Go service: News REST API v1.3 + feed fetcher daemon + SQLite
- `extension/`— OpenCloud Web extension (Vue 3), clone of the News web UI
- `webapp/`   — Material 3 PWA, clone of the news-android client

## Roadmap

- **F0** Spike: Go backend with minimal News API v1.3 (folders/feeds/items +
  sync), one test feed, basic auth; validate the Android client sync pattern.
- **F1** Backend complete: periodic fetch, robust parser, sanitization,
  multi-user SQLite, full API v1.3 + web routes, favicon, OPML, /status.
- **F2** OpenCloud Web extension: installable clone of the News web UI.
- **F3** Material 3 PWA: webapp identical to the news-android client.
- **F4** Integration + deploy: OpenCloud token auth, CT deploy, demo.

## Running the backend (F0)

```bash
cd backend
go test ./...
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o ocnews ./cmd/ocnews
OCNEWS_ADDR=:8094 OCNEWS_DATA_DIR=./data AUTH_USER=admin AUTH_PASS=secret \
  ./ocnews
```

API base path (same as Nextcloud News): `/index.php/apps/news/api/v1-3/`.
ES/EN error messages negotiated from the user's `language` preference
(`auto` falls back to `Accept-Language`, then English).

## References

- OpenCloud extension system: https://docs.opencloud.eu/docs/dev/web/extension-system/
- OpenCloud web apps install: https://docs.opencloud.eu/docs/admin/configuration/web-applications
- Nextcloud News: https://github.com/nextcloud/news
- News REST API v1.3: https://nextcloud.github.io/news/api/api-v1-3/
- News Android client: https://github.com/nextcloud/news-android
