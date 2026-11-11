// Pure formatting helpers for the note list. Kept free of Vue and of the
// i18n plugin: the caller passes its $gettext/$ngettext so the functions can
// be unit tested.

export const MAX_SNIPPET = 220

export function plainTextSnippet(raw: string, max = MAX_SNIPPET): string {
  const text = raw
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/gi, ' ')
    .replace(/&amp;/gi, '&')
    .replace(/&quot;/gi, '"')
    .replace(/&#39;/gi, "'")
    .replace(/\s+/g, ' ')
    .trim()
  if (text.length <= max) return text
  return text.slice(0, max).trimEnd() + '…'
}

export function itemSnippet(item: { body?: string | null; title?: string | null }, max = MAX_SNIPPET): string {
  return plainTextSnippet(item.body || item.title || '', max)
}

export function clusterTooltip(
  clusterSize: number | undefined | null,
  t: (message: string) => string
): string {
  if (!clusterSize || clusterSize <= 1) return ''
  return t('This story also appears in ') + String(clusterSize - 1) + ' ' + t('other feed(s)')
}

export function feedTitle(feeds: { id: number; title: string }[], feedId: number): string {
  return feeds.find((f) => f.id === feedId)?.title ?? ''
}

export function feedIcon(feeds: { id: number; faviconLink?: string | null }[], feedId: number): string {
  return feeds.find((f) => f.id === feedId)?.faviconLink ?? ''
}

export interface RelativeTranslators {
  t: (message: string) => string
  n: (singular: string, plural: string, count: number) => string
  /** Fallback for anything older than a day, usually a formatted date. */
  date: (epoch: number) => string
}

export function relativeLabel(epoch: number, nowMs: number, tr: RelativeTranslators): string {
  if (!epoch) return ''
  const secs = Math.round(nowMs / 1000 - epoch)
  if (secs < 60) return tr.t('just now')
  if (secs < 3600) {
    const minutes = Math.floor(secs / 60)
    return tr.n('%d minute ago', '%d minutes ago', minutes).replace('%d', String(minutes))
  }
  if (secs < 86400) {
    const hours = Math.floor(secs / 3600)
    return tr.n('%d hour ago', '%d hours ago', hours).replace('%d', String(hours))
  }
  return tr.date(epoch)
}
