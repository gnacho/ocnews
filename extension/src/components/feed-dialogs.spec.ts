import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import FeedFilterDialog from './FeedFilterDialog.vue'
import RetentionDialog from './RetentionDialog.vue'
import AddFeedDialog from './AddFeedDialog.vue'
import DiscoverFeedsDialog from './DiscoverFeedsDialog.vue'

const stubs = { 'oc-button': { template: '<button><slot /></button>' } }
const global = { stubs, mocks: { $gettext: (m: string) => m } }

const emptyFilter = { titleKeywords: '', bodyKeywords: '', urlKeywords: '' }

describe('FeedFilterDialog', () => {
  it('shows the three keyword fields it receives', () => {
    const wrapper = mount(FeedFilterDialog, {
      props: { hasFilter: true, modelValue: { titleKeywords: 'ads', bodyKeywords: 'cookie', urlKeywords: 'utm_' } },
      global
    })
    expect((wrapper.find('[data-testid="filter-title"]').element as HTMLInputElement).value).toBe('ads')
    expect((wrapper.find('[data-testid="filter-body"]').element as HTMLInputElement).value).toBe('cookie')
    expect((wrapper.find('[data-testid="filter-url"]').element as HTMLInputElement).value).toBe('utm_')
  })

  it('writes the edits into the filter object it was given', async () => {
    const form = { ...emptyFilter }
    const wrapper = mount(FeedFilterDialog, { props: { hasFilter: false, modelValue: form }, global })
    await wrapper.find('[data-testid="filter-title"]').setValue('sponsored')
    await wrapper.find('[data-testid="filter-body"]').setValue('cookie')
    expect(form).toEqual({ ...emptyFilter, titleKeywords: 'sponsored', bodyKeywords: 'cookie' })
  })

  it('only offers the clear action when there is a filter', () => {
    const withFilter = mount(FeedFilterDialog, { props: { hasFilter: true, modelValue: emptyFilter }, global })
    expect(withFilter.text()).toContain('Clear filter')

    const without = mount(FeedFilterDialog, { props: { hasFilter: false, modelValue: emptyFilter }, global })
    expect(without.text()).not.toContain('Clear filter')
  })

  it('emits confirm, clear and cancel', async () => {
    const wrapper = mount(FeedFilterDialog, { props: { hasFilter: true, modelValue: emptyFilter }, global })
    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    expect(wrapper.emitted('clear')).toHaveLength(1)
    await buttons.at(-1)!.trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
    await buttons.at(-2)!.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('disables save while saving', () => {
    const wrapper = mount(FeedFilterDialog, { props: { hasFilter: true, modelValue: emptyFilter, saving: true }, global })
    expect(wrapper.findAll('button').at(-1)!.attributes('disabled')).toBeDefined()
  })
})

describe('RetentionDialog', () => {
  it('shows the days it receives', () => {
    const wrapper = mount(RetentionDialog, { props: { modelValue: 30 }, global })
    expect((wrapper.find('[data-testid="retention-days"]').element as HTMLInputElement).value).toBe('30')
  })

  it('keeps the model a number, and an empty field becomes zero', async () => {
    const wrapper = mount(RetentionDialog, { props: { modelValue: 30 }, global })
    const input = wrapper.find('[data-testid="retention-days"]')
    await input.setValue('45')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([45])
    await input.setValue('')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([0])
  })

  it('confirms and cancels', async () => {
    const wrapper = mount(RetentionDialog, { props: { modelValue: 0 }, global })
    await wrapper.findAll('button').at(-1)!.trigger('click')
    await wrapper.findAll('button').at(0)!.trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })
})

describe('AddFeedDialog', () => {
  function mountAdd(props: Record<string, unknown> = {}) {
    return mount(AddFeedDialog, {
      props: { url: '', auth: false, user: '', pass: '', 'onUpdate:url': () => {}, 'onUpdate:auth': () => {}, 'onUpdate:user': () => {}, 'onUpdate:pass': () => {}, ...props },
      global
    })
  }

  it('focuses the URL field on open', () => {
    const wrapper = mount(AddFeedDialog, {
      props: { url: '', auth: false, user: '', pass: '', 'onUpdate:url': () => {}, 'onUpdate:auth': () => {}, 'onUpdate:user': () => {}, 'onUpdate:pass': () => {} },
      global,
      attachTo: document.body
    })
    expect(document.activeElement).toBe(wrapper.find('[data-testid="add-feed-url"]').element)
    wrapper.unmount()
  })

  it('keeps subscribe disabled until there is a URL', async () => {
    const wrapper = mountAdd()
    const subscribe = () => wrapper.findAll('button').at(-1)!
    expect(subscribe().attributes('disabled')).toBeDefined()

    await wrapper.setProps({ url: 'https://example.test/rss' })
    expect(subscribe().attributes('disabled')).toBeUndefined()
  })

  it('only shows the credentials when authentication is on', async () => {
    const wrapper = mountAdd()
    expect(wrapper.find('[data-testid="add-feed-user"]').exists()).toBe(false)
    await wrapper.setProps({ auth: true })
    expect(wrapper.find('[data-testid="add-feed-user"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="add-feed-pass"]').attributes('type')).toBe('password')
  })

  it('writes the url, the checkbox and the credentials back', async () => {
    const wrapper = mountAdd({ auth: true })
    await wrapper.find('[data-testid="add-feed-url"]').setValue('https://example.test/rss')
    expect(wrapper.emitted('update:url')?.at(-1)).toEqual(['https://example.test/rss'])
    await wrapper.find('[data-testid="add-feed-auth"]').setValue(false)
    expect(wrapper.emitted('update:auth')?.at(-1)).toEqual([false])
    await wrapper.find('[data-testid="add-feed-user"]').setValue('ana')
    expect(wrapper.emitted('update:user')?.at(-1)).toEqual(['ana'])
    await wrapper.find('[data-testid="add-feed-pass"]').setValue('secreto')
    expect(wrapper.emitted('update:pass')?.at(-1)).toEqual(['secreto'])
  })

  it('shows the error it receives', () => {
    expect(mountAdd({ error: 'No se pudo leer el feed' }).find('[data-testid="add-feed-error"]').text()).toBe('No se pudo leer el feed')
  })

  it('emits confirm from the button, from enter and cancel', async () => {
    const wrapper = mountAdd({ url: 'https://example.test/rss' })
    await wrapper.find('[data-testid="add-feed-url"]').trigger('keydown.enter')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
    await wrapper.findAll('button').at(-1)!.trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(2)
    await wrapper.findAll('button').at(-2)!.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })
})

describe('DiscoverFeedsDialog', () => {
  const candidates = [
    { url: 'https://a.test/rss', title: 'Feed A', type: 'rss' },
    { url: 'https://a.test/atom', title: '', type: 'atom' }
  ]

  it('lists the candidates with a title fallback to the url', () => {
    const wrapper = mount(DiscoverFeedsDialog, { props: { candidates, modelValue: 0 }, global })
    const labels = wrapper.findAll('label').map((l) => l.text())
    expect(labels[0]).toContain('Feed A')
    expect(labels[1]).toContain('https://a.test/atom')
    expect(labels[0]).toContain('rss')
  })

  it('selects the candidate as a number', async () => {
    const wrapper = mount(DiscoverFeedsDialog, { props: { candidates, modelValue: 0 }, global })
    await wrapper.findAll('[data-testid="discover-option"]')[1].setValue()
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([1])
  })

  it('emits confirm and cancel, and blocks subscribe while submitting', async () => {
    const wrapper = mount(DiscoverFeedsDialog, { props: { candidates, modelValue: 0, submitting: true }, global })
    expect(wrapper.findAll('button').at(-1)!.attributes('disabled')).toBeDefined()
    await wrapper.findAll('button').at(-2)!.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })
})
