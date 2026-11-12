import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue3-gettext', async (importOriginal) => ({
  ...(await importOriginal<object>()),
  useGettext: () => ({ $gettext: (message: string) => message })
}))

import AutoReadRulesDialog from './AutoReadRulesDialog.vue'
import ScraperDialog from './ScraperDialog.vue'
import FeedRulesDialog from './FeedRulesDialog.vue'
import ShortcutsDialog from './ShortcutsDialog.vue'
import { SHORTCUTS } from '../lib/shortcuts'
import type { Feed } from '../api'

const stubs = { 'oc-button': { template: '<button><slot /></button>' } }
const global = { stubs, mocks: { $gettext: (m: string) => m } }

const feeds: Feed[] = [
  {
    id: 4,
    url: 'https://example.test/rss',
    title: 'Ejemplo',
    faviconLink: '',
    added: 0,
    folderId: null,
    unreadCount: 0,
    ordering: 0,
    link: '',
    pinned: false,
    updateErrorCount: 0,
    lastUpdateError: null,
    isPodcast: false
  }
]

const rules = [
  { id: 1, feedId: 0, titlePattern: '(?i)urgente' },
  { id: 2, feedId: 4, titlePattern: '^(RE|FW):' },
  { id: 3, feedId: 99, titlePattern: 'borrar' }
]

describe('AutoReadRulesDialog', () => {
  function mountDialog(props: Record<string, unknown> = {}) {
    return mount(AutoReadRulesDialog, {
      props: { rules, feeds, pattern: '', feedId: 0, 'onUpdate:pattern': () => {}, 'onUpdate:feedId': () => {}, ...props },
      global
    })
  }

  it('lists the rules with the feed they apply to', () => {
    const wrapper = mountDialog()
    const rows = wrapper.findAll('[data-testid="auto-read-list"] code')
    expect(rows.map((r) => r.text())).toEqual(['(?i)urgente', '^(RE|FW):', 'borrar'])
    const text = wrapper.find('[data-testid="auto-read-list"]').text()
    expect(text).toContain('All feeds')
    expect(text).toContain('Ejemplo')
    expect(text).toContain('#99')
  })

  it('hides the list when there are no rules', () => {
    expect(mountDialog({ rules: [] }).find('[data-testid="auto-read-list"]').exists()).toBe(false)
  })

  it('offers every feed plus the all-feeds option', () => {
    const options = mountDialog().find('[data-testid="auto-read-feed"]').findAll('option').map((o) => o.text())
    expect(options).toEqual(['All feeds', 'Ejemplo'])
  })

  it('emits add from the button and from enter, and remove with the rule', async () => {
    const wrapper = mountDialog()
    await wrapper.find('[data-testid="auto-read-pattern"]').setValue('nuevo')
    expect(wrapper.emitted('update:pattern')?.at(-1)).toEqual(['nuevo'])
    await wrapper.find('[data-testid="auto-read-pattern"]').trigger('keydown.enter')
    expect(wrapper.emitted('add')).toHaveLength(1)

    await wrapper.find('[data-testid="auto-read-feed"]').setValue('4')
    expect(wrapper.emitted('update:feedId')?.at(-1)).toEqual([4])

    const remove = wrapper.find('[data-testid="auto-read-list"] button')!
    await remove.trigger('click')
    expect(wrapper.emitted('remove')?.[0]).toEqual([rules[0]])
  })

  it('closes from the button and the backdrop', async () => {
    const wrapper = mountDialog()
    await wrapper.findAll('button').at(-1)!.trigger('click')
    await wrapper.find('[role="dialog"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(2)
  })
})

describe('ScraperDialog', () => {
  it('shows the selector and only offers clear when there is one', () => {
    const empty = mount(ScraperDialog, { props: { modelValue: '' }, global })
    expect(empty.text()).not.toContain('Clear')
    const filled = mount(ScraperDialog, { props: { modelValue: 'article' }, global })
    expect((filled.find('[data-testid="scraper-selector"]').element as HTMLInputElement).value).toBe('article')
    expect(filled.text()).toContain('Clear')
  })

  it('emits confirm, clear and cancel', async () => {
    const wrapper = mount(ScraperDialog, { props: { modelValue: 'article' }, global })
    await wrapper.find('[data-testid="scraper-selector"]').setValue('main')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['main'])
    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    await buttons.at(-1)!.trigger('click')
    await buttons.at(-2)!.trigger('click')
    expect(wrapper.emitted('clear')).toHaveLength(1)
    expect(wrapper.emitted('confirm')).toHaveLength(1)
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })
})

describe('FeedRulesDialog', () => {
  it('shows the title and both rule fields', () => {
    const wrapper = mount(FeedRulesDialog, {
      props: { title: 'Rules for Ejemplo', hasRules: true, modelValue: { block: 'EntryTitle=a', keep: 'EntryTitle=b' } },
      global
    })
    expect(wrapper.find('h3').text()).toBe('Rules for Ejemplo')
    expect((wrapper.find('[data-testid="rules-block"]').element as HTMLTextAreaElement).value).toBe('EntryTitle=a')
    expect((wrapper.find('[data-testid="rules-keep"]').element as HTMLTextAreaElement).value).toBe('EntryTitle=b')
  })

  it('writes the edits into the rules object it was given', async () => {
    const form = { block: '', keep: '' }
    const wrapper = mount(FeedRulesDialog, { props: { title: 'Global rules', hasRules: false, modelValue: form }, global })
    await wrapper.find('[data-testid="rules-block"]').setValue('EntryURL=spam')
    await wrapper.find('[data-testid="rules-keep"]').setValue('EntryTitle=ok')
    expect(form).toEqual({ block: 'EntryURL=spam', keep: 'EntryTitle=ok' })
  })

  it('only offers clear when there are rules', () => {
    const withRules = mount(FeedRulesDialog, { props: { title: 'x', hasRules: true, modelValue: { block: '', keep: '' } }, global })
    expect(withRules.text()).toContain('Clear rules')
    const without = mount(FeedRulesDialog, { props: { title: 'x', hasRules: false, modelValue: { block: '', keep: '' } }, global })
    expect(without.text()).not.toContain('Clear rules')
  })

  it('emits confirm, clear and cancel', async () => {
    const wrapper = mount(FeedRulesDialog, { props: { title: 'x', hasRules: true, modelValue: { block: '', keep: '' } }, global })
    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    await buttons.at(-1)!.trigger('click')
    await buttons.at(-2)!.trigger('click')
    expect(wrapper.emitted('clear')).toHaveLength(1)
    expect(wrapper.emitted('confirm')).toHaveLength(1)
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })
})

describe('ShortcutsDialog', () => {
  it('lists every shortcut with its keys and label', () => {
    const wrapper = mount(ShortcutsDialog, { props: { shortcuts: SHORTCUTS }, global })
    const keys = wrapper.findAll('dt code').map((k) => k.text())
    expect(keys).toHaveLength(SHORTCUTS.length)
    expect(keys[0]).toBe('j / n / ↓')
    expect(wrapper.text()).toContain('Next article')
  })

  it('closes from the button and the backdrop', async () => {
    const wrapper = mount(ShortcutsDialog, { props: { shortcuts: SHORTCUTS }, global })
    await wrapper.find('button').trigger('click')
    await wrapper.find('[role="dialog"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(2)
  })
})

describe('the shortcut list', () => {
  it('keeps the keys unique and the labels translatable', () => {
    const keys = SHORTCUTS.map((s) => s.keys)
    expect(new Set(keys).size).toBe(keys.length)
    for (const shortcut of SHORTCUTS) {
      expect(shortcut.label.length).toBeGreaterThan(0)
      expect(shortcut.keys.length).toBeGreaterThan(0)
    }
  })
})
