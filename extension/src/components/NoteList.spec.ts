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

import NoteList from './NoteList.vue'
import type { Feed, Item } from '../api'

function makeItem(overrides: Partial<Item> = {}): Item {
  return {
    id: 1,
    guid: 'g1',
    guidHash: 'h1',
    url: 'https://example.test/1',
    title: 'Primera nota',
    author: null,
    pubDate: Math.floor(Date.now() / 1000) - 600,
    body: '<p>cuerpo de la <b>nota</b></p>',
    enclosureMime: null,
    enclosureLink: null,
    mediaThumbnail: null,
    feedId: 10,
    unread: true,
    starred: false,
    lastModified: 0,
    ...overrides
  }
}

const feeds: Feed[] = [
  {
    id: 10,
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

const stubs = { 'oc-button': { template: '<button><slot /></button>' } }

function mountList(props: Record<string, unknown> = {}) {
  return mount(NoteList, {
    props: {
      items: [makeItem()],
      feeds,
      loading: false,
      error: '',
      title: 'Todas',
      unreadCount: 1,
      searchActive: false,
      selectedId: null,
      moreAvailable: false,
      width: 380,
      showAll: false,
      oldestFirst: false,
      searchQuery: '',
      ...props
    },
    global: { stubs, mocks: { $gettext: (m: string) => m } }
  })
}

describe('NoteList', () => {
  it('shows the title, the unread badge and the mark-all button', () => {
    const wrapper = mountList()
    expect(wrapper.find('h1').text()).toContain('Todas')
    expect(wrapper.find('h1').text()).toContain('· 1')
    expect(wrapper.find('h1 button').exists()).toBe(true)
  })

  it('hides the unread badge and the mark-all button at zero', () => {
    const wrapper = mountList({ unreadCount: 0 })
    expect(wrapper.find('h1').text()).not.toContain('·')
    expect(wrapper.find('h1 button').exists()).toBe(false)
  })

  it('renders one row per article with its snippet, feed and relative time', () => {
    const wrapper = mountList()
    const rows = wrapper.findAll('[data-testid="note-row"]')
    expect(rows).toHaveLength(1)
    const text = rows[0].text()
    expect(text).toContain('Primera nota')
    expect(text).toContain('cuerpo de la nota')
    expect(text).toContain('Ejemplo')
    expect(text).toContain('10 minutes ago')
  })

  it('marks the selected row and dims the read ones', () => {
    const wrapper = mountList({ selectedId: 1, items: [makeItem({ unread: false })] })
    const row = wrapper.find('[data-testid="note-row"]')
    expect(row.attributes('style')).toContain('--news-hover-bg')
    expect(row.attributes('style')).toContain('opacity: 0.65')
  })

  it('emits select, toggleUnread and toggleStar with the article', async () => {
    const item = makeItem()
    const wrapper = mountList({ items: [item] })
    await wrapper.find('[data-testid="note-row"]').trigger('click')
    expect(wrapper.emitted('select')?.[0]).toEqual([item])

    const buttons = wrapper.findAll('[data-testid="note-row"] button')
    await buttons[0].trigger('click')
    expect(wrapper.emitted('toggleUnread')?.[0]).toEqual([item])
    await buttons[1].trigger('click')
    expect(wrapper.emitted('toggleStar')?.[0]).toEqual([item])
  })

  it('shows the cluster badge only for duplicated stories', () => {
    const many = mountList({ items: [makeItem({ clusterSize: 2 })] })
    expect(many.find('[title="This story also appears in 1 other feed(s)"]').exists()).toBe(true)

    const single = mountList({ items: [makeItem({ clusterSize: 1 })] })
    expect(single.find('[title^="This story also appears"]').exists()).toBe(false)
  })

  it('renders the loading, error and empty states instead of the rows', () => {
    const loading = mountList({ loading: true, items: [] })
    expect(loading.text()).toContain('Loading…')
    expect(loading.find('[data-testid="note-row"]').exists()).toBe(false)

    const failed = mountList({ error: 'Sin red', items: [] })
    expect(failed.text()).toContain('Sin red')
    expect(failed.text()).toContain('No articles')

    expect(mountList({ items: [] }).text()).toContain('No articles')
  })

  it('emits the search actions and keeps the query in sync', async () => {
    const wrapper = mountList()
    const input = wrapper.find('input[type="search"]')
    await input.setValue('cafe')
    expect(wrapper.emitted('update:searchQuery')?.at(-1)).toEqual(['cafe'])

    await input.trigger('keydown.enter')
    expect(wrapper.emitted('search')).toHaveLength(1)

    await wrapper.find('input[type="search"]').setValue('algo')
    const clear = wrapper.findAll('button').find((b) => b.attributes('title') === 'Clear search')
    await clear!.trigger('click')
    expect(wrapper.emitted('clearSearch')).toHaveLength(1)
  })

  it('offers load more only when there is more to load', async () => {
    expect(mountList().findAll('button').some((b) => b.text() === 'Load more')).toBe(false)
    const wrapper = mountList({ moreAvailable: true })
    const more = wrapper.findAll('button').find((b) => b.text() === 'Load more')!
    await more.trigger('click')
    expect(wrapper.emitted('loadMore')).toHaveLength(1)
  })

  it('exposes focusSearch for the keyboard shortcuts', () => {
    const wrapper = mount(NoteList, {
      props: {
        items: [makeItem()],
        feeds,
        loading: false,
        error: '',
        title: 'Todas',
        unreadCount: 1,
        searchActive: false,
        selectedId: null,
        moreAvailable: false,
        width: 380,
        showAll: false,
        oldestFirst: false,
        searchQuery: ''
      },
      global: { stubs, mocks: { $gettext: (m: string) => m } },
      attachTo: document.body
    })
    const input = wrapper.find('input[type="search"]').element as HTMLInputElement
    ;(document.activeElement as HTMLElement | null)?.blur()
    ;(wrapper.vm as unknown as { focusSearch: () => void }).focusSearch()
    expect(document.activeElement).toBe(input)
    wrapper.unmount()
  })
})
