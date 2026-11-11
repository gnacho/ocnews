import { describe, expect, it } from 'vitest'

import {
  MAX_SNIPPET,
  clusterTooltip,
  feedIcon,
  feedTitle,
  itemSnippet,
  plainTextSnippet,
  relativeLabel
} from './format'

const t = (message: string) => message
const n = (singular: string, plural: string, count: number) =>
  (count === 1 ? singular : plural).replace('%d', String(count))
const date = (epoch: number) => `date:${epoch}`
const tr = { t, n, date }

describe('plainTextSnippet', () => {
  it('strips tags and collapses whitespace', () => {
    expect(plainTextSnippet('<p>hola   <b>mundo</b>\n\n otra</p>')).toBe('hola mundo otra')
  })

  it('decodes the entities the feed bodies use', () => {
    expect(plainTextSnippet('a&nbsp;b &amp; c &quot;d&quot; &#39;e&#39;')).toBe('a b & c "d" \'e\'')
  })

  it('keeps a short text untouched', () => {
    expect(plainTextSnippet('texto corto')).toBe('texto corto')
  })

  it('cuts at the limit and adds an ellipsis', () => {
    const long = 'x'.repeat(MAX_SNIPPET + 40)
    const out = plainTextSnippet(long)
    expect(out.endsWith('…')).toBe(true)
    expect(out.length).toBe(MAX_SNIPPET + 1)
  })

  it('does not add the ellipsis at exactly the limit', () => {
    const exact = 'y'.repeat(MAX_SNIPPET)
    expect(plainTextSnippet(exact)).toBe(exact)
  })

  it('honours a custom limit', () => {
    expect(plainTextSnippet('abcdefgh', 4)).toBe('abcd…')
  })
})

describe('itemSnippet', () => {
  it('prefers the body', () => {
    expect(itemSnippet({ body: '<p>cuerpo</p>', title: 'titulo' })).toBe('cuerpo')
  })

  it('falls back to the title when there is no body', () => {
    expect(itemSnippet({ body: null, title: 'titulo' })).toBe('titulo')
    expect(itemSnippet({ body: '', title: 'titulo' })).toBe('titulo')
  })

  it('returns an empty string when there is nothing', () => {
    expect(itemSnippet({})).toBe('')
  })
})

describe('clusterTooltip', () => {
  it('says nothing when the story is alone', () => {
    expect(clusterTooltip(undefined, t)).toBe('')
    expect(clusterTooltip(0, t)).toBe('')
    expect(clusterTooltip(1, t)).toBe('')
  })

  it('counts the other feeds', () => {
    expect(clusterTooltip(3, t)).toBe('This story also appears in 2 other feed(s)')
  })
})

describe('feed lookups', () => {
  const feeds = [
    { id: 1, title: 'Uno', faviconLink: 'https://a/ico.png' },
    { id: 2, title: 'Dos' }
  ]

  it('finds the title and the icon by id', () => {
    expect(feedTitle(feeds, 1)).toBe('Uno')
    expect(feedIcon(feeds, 1)).toBe('https://a/ico.png')
  })

  it('returns an empty string for an unknown feed or a missing icon', () => {
    expect(feedTitle(feeds, 99)).toBe('')
    expect(feedIcon(feeds, 2)).toBe('')
  })
})

describe('relativeLabel', () => {
  const now = 1_700_000_000_000 // ms
  const nowSecs = now / 1000

  it('returns nothing without a date', () => {
    expect(relativeLabel(0, now, tr)).toBe('')
  })

  it('says "just now" under a minute', () => {
    expect(relativeLabel(nowSecs - 30, now, tr)).toBe('just now')
  })

  it('uses minutes under an hour, singular and plural', () => {
    expect(relativeLabel(nowSecs - 60, now, tr)).toBe('1 minute ago')
    expect(relativeLabel(nowSecs - 600, now, tr)).toBe('10 minutes ago')
  })

  it('uses hours under a day', () => {
    expect(relativeLabel(nowSecs - 3600, now, tr)).toBe('1 hour ago')
    expect(relativeLabel(nowSecs - 7200, now, tr)).toBe('2 hours ago')
  })

  it('falls back to the formatted date past a day', () => {
    expect(relativeLabel(nowSecs - 90000, now, tr)).toBe(`date:${nowSecs - 90000}`)
  })
})
