# 001 — Clicking the News app icon in the app switcher does nothing

- **Date**: 2026-08-15
- **Component**: extension (web-app-news)
- **Status**: fixed (see commit)

## Symptom

Clicking the News icon in the OpenCloud web app switcher on
cloud.example.com produces no navigation.

## Root cause

`src/index.ts` registered TWO routes with the same `path: '/'`
(a redirect record plus the root component record). The redirect record
wins route matching and points to `/news/`, which resolves back to the
extension root redirect → redirect loop, navigation aborts.

Reference: official apps (web-app-notes) register a single `path: '/'`
component route and let the host mount it under `/{appId}` via the
menu item `path: urlJoin(appInfo.id)`.

## Fix

- Single root route (`/`) with the NewsApp component; no redirect record.
- Also switch the app icon to the lucide `rss` glyph as requested
  (the host menu resolves lucide names: sticky-note, map-2, flow-chart…).
