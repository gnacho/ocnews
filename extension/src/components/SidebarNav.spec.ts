import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { Rss, Star } from 'lucide-vue-next'

vi.mock('vue3-gettext', async (importOriginal) => ({
  ...(await importOriginal<object>()),
  useGettext: () => ({ $gettext: (message: string) => message })
}))

import SidebarNav from './SidebarNav.vue'
import type { NavEntry } from '../lib/nav'

const entries: NavEntry[] = [
  { key: 'all', kind: 'all', label: 'All articles', icon: Rss, sel: { kind: 'all' }, unread: 12 },
  { key: 'starred', kind: 'starred', label: 'Starred', icon: Star, sel: { kind: 'starred' }, unread: 0 },
  { key: 'folder-1', kind: 'folder', label: 'Tecnología', icon: Rss, sel: { kind: 'folder', id: 1 }, unread: 4, depth: 1 },
  {
    key: 'feed-2',
    kind: 'feed',
    label: 'Ejemplo',
    icon: Rss,
    sel: { kind: 'feed', id: 2 },
    unread: 3,
    depth: 2,
    favicon: 'https://example.test/favicon.ico',
    error: 'HTTP 500'
  },
  { key: 'search-9', kind: 'search', label: 'mi búsqueda', icon: Rss, sel: { kind: 'all' }, unread: 0, searchId: 9 }
]

const stubs = { 'oc-button': { template: '<button><slot /></button>' } }

function mountNav(props: Record<string, unknown> = {}) {
  return mount(SidebarNav, {
    props: { entries, activeKey: 'all', openMenu: '', refreshing: false, ...props },
    global: { stubs, mocks: { $gettext: (m: string) => m } }
  })
}

describe('SidebarNav', () => {
  it('renders one row per entry with its label, unread count and indentation', () => {
    const rows = mountNav().findAll('[data-testid="nav-entry"]')
    expect(rows).toHaveLength(entries.length)
    expect(rows[0].text()).toContain('All articles')
    expect(rows[0].text()).toContain('12')
    expect(rows[2].attributes('style')).toContain('padding-left: 28px')
    expect(rows[3].attributes('style')).toContain('padding-left: 48px')
  })

  it('marks the active entry and bolds the ones with unread articles', () => {
    const rows = mountNav().findAll('[data-testid="nav-entry"]')
    expect(rows[0].attributes('style')).toContain('--oc-role-secondary-container')
    expect(rows[0].attributes('style')).toContain('font-weight: 600')
    expect(rows[1].attributes('style')).toContain('font-weight: 400')
  })

  it('shows the favicon of a feed and the warning of a failing one', () => {
    const feedRow = mountNav().findAll('[data-testid="nav-entry"]')[3]
    expect(feedRow.find('img').attributes('src')).toBe('https://example.test/favicon.ico')
    expect(feedRow.find('[title="HTTP 500"]').exists()).toBe(true)
  })

  it('emits the entry when a row is clicked', async () => {
    const wrapper = mountNav()
    await wrapper.findAll('[data-testid="nav-entry"]')[2].trigger('click')
    expect(wrapper.emitted('select')?.[0]).toEqual([entries[2]])
  })

  it('opens the context menu only for feeds, folders and saved searches', async () => {
    const wrapper = mountNav()
    const rows = wrapper.findAll('[data-testid="nav-entry"]')
    const optionsOf = (index: number) => rows[index].findAll('button').length
    expect(optionsOf(0)).toBe(0)
    expect(optionsOf(2)).toBe(1)
    expect(optionsOf(4)).toBe(1)

    await rows[2].find('button').trigger('click')
    expect(wrapper.emitted('update:openMenu')?.at(-1)).toEqual(['folder-1'])
  })

  it('lists the feed actions in the context menu and emits the chosen one', async () => {
    const wrapper = mountNav({ openMenu: 'feed-2' })
    const menu = wrapper.find('[data-testid="context-menu"]')
    const labels = menu.findAll('button').map((b) => b.text())
    expect(labels).toEqual([
      'Rename',
      'Move to folder…',
      'Credentials…',
      'Filter articles…',
      'Rules…',
      'Scraper…',
      'Retention…',
      'Mark all read',
      'Delete'
    ])
    await menu.findAll('button')[1].trigger('click')
    expect(wrapper.emitted('move')?.[0]).toEqual([entries[3]])
  })

  it('lists the folder actions in the context menu', () => {
    const menu = mountNav({ openMenu: 'folder-1' }).find('[data-testid="context-menu"]')
    expect(menu.findAll('button').map((b) => b.text())).toEqual([
      'Rename',
      'New subfolder…',
      'Mark all read',
      'Delete'
    ])
  })

  it('marks the delete action as dangerous', () => {
    const menu = mountNav({ openMenu: 'folder-1' }).find('[data-testid="context-menu"]')
    const remove = menu.findAll('button').at(-1)!
    expect(remove.attributes('style')).toContain('--news-error')
  })

  it('emits the header actions: add feed, refresh and new folder', async () => {
    const wrapper = mountNav()
    await wrapper.find('.news-cta').trigger('click')
    const headerButtons = wrapper.find('.news-section').findAll('button')
    await headerButtons[0].trigger('click')
    await headerButtons[1].trigger('click')
    expect(wrapper.emitted('addFeed')).toHaveLength(1)
    expect(wrapper.emitted('refresh')).toHaveLength(1)
    expect(wrapper.emitted('createFolder')).toHaveLength(1)
  })

  it('disables the refresh button while refreshing', () => {
    const wrapper = mountNav({ refreshing: true })
    expect(wrapper.find('.news-section').findAll('button')[0].attributes('disabled')).toBeDefined()
  })

  it('emits settings and shortcuts from the footer', async () => {
    const wrapper = mountNav()
    const footer = wrapper.findAll('.news-quiet-action')
    await footer[0].trigger('click')
    await footer[1].trigger('click')
    expect(wrapper.emitted('settings')).toHaveLength(1)
    expect(wrapper.emitted('shortcuts')).toHaveLength(1)
  })

  it('emits the chosen OPML file and clears the input', async () => {
    const wrapper = mountNav()
    const input = wrapper.find('input[type="file"]')
    const file = new File(['<opml/>'], 'subs.opml', { type: 'text/xml' })
    Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
    await input.trigger('change')
    expect(wrapper.emitted('importOpml')?.[0]).toEqual([file])
  })

  it('exposes the OPML picker for the settings dialog', () => {
    const wrapper = mountNav()
    const input = wrapper.find('input[type="file"]').element as HTMLInputElement
    let clicked = false
    input.click = () => { clicked = true }
    ;(wrapper.vm as unknown as { openOpmlPicker: () => void }).openOpmlPicker()
    expect(clicked).toBe(true)
  })
})
