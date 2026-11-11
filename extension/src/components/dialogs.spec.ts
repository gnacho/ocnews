import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import TextPromptDialog from './TextPromptDialog.vue'
import MoveFeedDialog from './MoveFeedDialog.vue'
import FeedCredentialsDialog from './FeedCredentialsDialog.vue'

const stubs = { 'oc-button': { template: '<button><slot /></button>' } }
const global = { stubs, mocks: { $gettext: (m: string) => m } }

describe('TextPromptDialog', () => {
  it('renders the title and the current value', () => {
    const wrapper = mount(TextPromptDialog, { props: { title: 'Nueva carpeta', modelValue: 'Tecnología' }, global })
    expect(wrapper.find('h3').text()).toBe('Nueva carpeta')
    expect((wrapper.find('[data-testid="prompt-input"]').element as HTMLInputElement).value).toBe('Tecnología')
  })

  it('focuses the input when it opens', () => {
    const wrapper = mount(TextPromptDialog, { props: { title: 'Renombrar', modelValue: '' }, global, attachTo: document.body })
    const input = wrapper.find('[data-testid="prompt-input"]').element
    expect(document.activeElement).toBe(input)
    wrapper.unmount()
  })

  it('emits confirm on enter and on save, and keeps the typed value in the model', async () => {
    const wrapper = mount(TextPromptDialog, { props: { title: 'Renombrar', modelValue: '' }, global })
    const input = wrapper.find('[data-testid="prompt-input"]')
    await input.setValue('nuevo nombre')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['nuevo nombre'])

    await input.trigger('keydown.enter')
    expect(wrapper.emitted('confirm')).toHaveLength(1)

    await wrapper.findAll('button').at(-1)!.trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(2)
  })

  it('cancels from the button and the backdrop', async () => {
    const wrapper = mount(TextPromptDialog, { props: { title: 'Renombrar', modelValue: '' }, global })
    await wrapper.findAll('button').at(0)!.trigger('click')
    await wrapper.find('[role="dialog"]').trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(2)
  })

  it('disables save while saving', () => {
    const wrapper = mount(TextPromptDialog, { props: { title: 'Renombrar', modelValue: '', saving: true }, global })
    expect(wrapper.findAll('button').at(-1)!.attributes('disabled')).toBeDefined()
  })
})

describe('MoveFeedDialog', () => {
  const folders = [
    { id: 1, name: 'Tecnología' },
    { id: 2, name: 'Ciencia' }
  ]

  it('offers the root plus every folder', () => {
    const wrapper = mount(MoveFeedDialog, { props: { folders, modelValue: 0 }, global })
    const options = wrapper.findAll('option').map((o) => o.text())
    expect(options).toEqual(['No folder', 'Tecnología', 'Ciencia'])
  })

  it('shows the current folder as the value', () => {
    const wrapper = mount(MoveFeedDialog, { props: { folders, modelValue: 2 }, global })
    expect((wrapper.find('[data-testid="folder-select"]').element as HTMLSelectElement).value).toBe('2')
  })

  it('emits the chosen folder as a number', async () => {
    const wrapper = mount(MoveFeedDialog, { props: { folders, modelValue: 0 }, global })
    await wrapper.find('[data-testid="folder-select"]').setValue('1')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([1])
  })

  it('confirms and cancels', async () => {
    const wrapper = mount(MoveFeedDialog, { props: { folders, modelValue: 0 }, global })
    await wrapper.findAll('button').at(-1)!.trigger('click')
    await wrapper.findAll('button').at(0)!.trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })
})

describe('FeedCredentialsDialog', () => {
  it('hides the remove action when the feed has no credentials', () => {
    const wrapper = mount(FeedCredentialsDialog, {
      props: { hasAuth: false, user: '', pass: '', 'onUpdate:user': () => {}, 'onUpdate:pass': () => {} },
      global
    })
    expect(wrapper.text()).not.toContain('Remove credentials')
  })

  it('offers the remove action when there are credentials', async () => {
    const wrapper = mount(FeedCredentialsDialog, {
      props: { hasAuth: true, user: 'ana', pass: '', 'onUpdate:user': () => {}, 'onUpdate:pass': () => {} },
      global
    })
    const remove = wrapper.findAll('button').find((b) => b.text() === 'Remove credentials')!
    await remove.trigger('click')
    expect(wrapper.emitted('remove')).toHaveLength(1)
  })

  it('writes both fields back and emits confirm and cancel', async () => {
    const wrapper = mount(FeedCredentialsDialog, {
      props: { hasAuth: false, user: '', pass: '', 'onUpdate:user': () => {}, 'onUpdate:pass': () => {} },
      global
    })
    await wrapper.find('[data-testid="cred-user"]').setValue('ana')
    expect(wrapper.emitted('update:user')?.at(-1)).toEqual(['ana'])
    await wrapper.find('[data-testid="cred-pass"]').setValue('secreto')
    expect(wrapper.emitted('update:pass')?.at(-1)).toEqual(['secreto'])

    await wrapper.findAll('button').at(-1)!.trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
    await wrapper.findAll('button').at(-2)!.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('keeps the password field masked', () => {
    const wrapper = mount(FeedCredentialsDialog, {
      props: { hasAuth: false, user: '', pass: '', 'onUpdate:user': () => {}, 'onUpdate:pass': () => {} },
      global
    })
    expect(wrapper.find('[data-testid="cred-pass"]').attributes('type')).toBe('password')
    expect(wrapper.find('[data-testid="cred-pass"]').attributes('autocomplete')).toBe('new-password')
  })
})
