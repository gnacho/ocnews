import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue3-gettext', async (importOriginal) => ({
  ...(await importOriginal<object>()),
  useGettext: () => ({
    $gettext: (message: string) => message,
    $ngettext: (singular: string, plural: string, count: number) =>
      (count === 1 ? singular : plural).replace('%d', String(count))
  })
}))

import ReaderPane from './ReaderPane.vue'
import type { Feed, Item } from '../api'

function makeItem(overrides: Partial<Item> = {}): Item {
  return {
    id: 7,
    guid: 'g',
    guidHash: 'h',
    url: 'https://example.test/nota',
    title: 'Una nota',
    author: 'Ana',
    pubDate: Math.floor(Date.now() / 1000) - 120,
    body: '<p>cuerpo</p>',
    enclosureMime: null,
    enclosureLink: null,
    mediaThumbnail: null,
    feedId: 3,
    unread: true,
    starred: false,
    lastModified: 0,
    ...overrides
  }
}

const feeds: Feed[] = [
  {
    id: 3,
    url: 'https://example.test/rss',
    title: 'Ejemplo',
    faviconLink: 'https://example.test/favicon.ico',
    added: 0,
    folderId: null,
    unreadCount: 1,
    ordering: 0,
    link: '',
    pinned: false,
    updateErrorCount: 0,
    lastUpdateError: null,
    isPodcast: false
  }
]

const reader = {
  class: 'news-theme-dark',
  maxWidth: '72ch',
  fontFamily: 'Georgia, serif',
  fontSizePx: '17px',
  bg: 'var(--news-bg)',
  fg: 'var(--news-fg)'
}

const stubs = { 'oc-button': { template: '<button><slot /></button>' } }

function mountPane(props: Record<string, unknown> = {}) {
  return mount(ReaderPane, {
    props: { item: makeItem(), feeds, body: '<p>texto del artículo</p>', reader, shared: false, ...props },
    global: { stubs, mocks: { $gettext: (m: string) => m } }
  })
}

describe('ReaderPane', () => {
  it('links the title to the original article', () => {
    const wrapper = mountPane()
    const heading = wrapper.find('h2')
    expect(heading.text()).toBe('Una nota')
    expect(heading.find('a').attributes('href')).toBe('https://example.test/nota')
    expect(heading.find('a').attributes('target')).toBe('_blank')
    expect(heading.find('a').attributes('rel')).toBe('noopener noreferrer')
  })

  it('shows the feed, the author and the relative time', () => {
    const text = mountPane().text()
    expect(text).toContain('Ejemplo')
    expect(text).toContain('Ana')
    expect(text).toContain('2 minutes ago')
  })

  it('drops the author when there is none', () => {
    const text = mountPane({ item: makeItem({ author: null }) }).text()
    expect(text).toContain('Ejemplo')
    expect(text).not.toContain('· Ana')
  })

  it('renders the body it receives as the article html', () => {
    const wrapper = mountPane()
    const body = wrapper.find('[data-testid="reader-body"]')
    expect(body.html()).toContain('texto del artículo')
    expect(body.classes()).toContain('news-theme-dark')
    expect(body.attributes('style')).toContain('max-width: 72ch')
    expect(body.attributes('style')).toContain('font-size: 17px')
  })

  it('plays audio enclosures', () => {
    const wrapper = mountPane({ item: makeItem({ enclosureMime: 'audio/mpeg', enclosureLink: 'https://a/x.mp3' }) })
    const audio = wrapper.find('audio')
    expect(audio.exists()).toBe(true)
    expect(audio.attributes('src')).toBe('https://a/x.mp3')
    expect(wrapper.find('video').exists()).toBe(false)
  })

  it('plays video enclosures', () => {
    const wrapper = mountPane({ item: makeItem({ enclosureMime: 'video/mp4', enclosureLink: 'https://a/x.mp4' }) })
    expect(wrapper.find('video').exists()).toBe(true)
    expect(wrapper.find('audio').exists()).toBe(false)
  })

  it('shows no player without an enclosure', () => {
    const wrapper = mountPane()
    expect(wrapper.find('audio').exists()).toBe(false)
    expect(wrapper.find('video').exists()).toBe(false)
  })

  it('offers the full article only when the feed does not carry it', () => {
    const without = mountPane()
    expect(without.findAll('button').some((b) => b.attributes('title') === 'Full article')).toBe(true)

    const withFull = mountPane({ item: makeItem({ feedFullContent: true }) })
    expect(withFull.findAll('button').some((b) => b.attributes('title') === 'Full article')).toBe(false)
  })

  it('marks the full article button when the body is the full one', () => {
    const wrapper = mountPane({ fullBodyActive: true })
    const button = wrapper.findAll('button').find((b) => b.attributes('title') === 'Show summary')!
    expect(button.exists()).toBe(true)
  })

  it('tells the reader when the full article is loading or failed', () => {
    expect(mountPane({ fullLoading: true }).text()).toContain('loading full article…')
    expect(mountPane({ fullFailed: true }).text()).toContain('full article not available')
  })

  it('labels the read and star actions from the article state', () => {
    const unread = mountPane()
    expect(unread.findAll('button').some((b) => b.attributes('title') === 'Mark as read')).toBe(true)

    const read = mountPane({ item: makeItem({ unread: false, starred: true }) })
    expect(read.findAll('button').some((b) => b.attributes('title') === 'Mark as unread')).toBe(true)
    expect(read.findAll('button').some((b) => b.attributes('title') === 'Unstar')).toBe(true)
  })

  it('labels the share action from the shared state', () => {
    expect(mountPane().findAll('button').some((b) => b.attributes('title') === 'Share article')).toBe(true)
    expect(mountPane({ shared: true }).findAll('button').some((b) => b.attributes('title') === 'Stop sharing')).toBe(true)
  })

  it('emits every header action with its payload', async () => {
    const item = makeItem()
    const wrapper = mountPane({ item })
    const byTitle = (title: string) => wrapper.findAll('button').find((b) => b.attributes('title') === title)!

    await byTitle('Full article').trigger('click')
    expect(wrapper.emitted('toggleFull')).toHaveLength(1)

    await byTitle('Mark as read').trigger('click')
    expect(wrapper.emitted('toggleUnread')?.[0]).toEqual([item])

    await byTitle('Star').trigger('click')
    expect(wrapper.emitted('toggleStar')?.[0]).toEqual([item])

    await byTitle('Open in new tab').trigger('click')
    expect(wrapper.emitted('openOriginal')).toHaveLength(1)

    await byTitle('Share article').trigger('click')
    expect(wrapper.emitted('toggleShare')).toHaveLength(1)

    await wrapper.findAll('button')[0].trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })
})
