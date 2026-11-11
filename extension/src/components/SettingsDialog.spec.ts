import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import SettingsDialog from './SettingsDialog.vue'
import type { UserSettings } from '../api'

function makeForm(overrides: Partial<UserSettings> = {}): UserSettings {
  return {
    theme: 'system',
    readerMaxWidth: 'wide',
    readerFont: 'default',
    readerFontSize: '15',
    feedIntervalMin: '',
    ntfyTopic: '',
    ...overrides
  }
}

const stubs = { 'oc-button': { template: '<button><slot /></button>' } }

function mountDialog(props: Record<string, unknown> = {}) {
  const form = (props.form as UserSettings) ?? makeForm()
  return mount(SettingsDialog, {
    props: { modelValue: form, ...props },
    global: { stubs, mocks: { $gettext: (m: string) => m } }
  })
}

describe('SettingsDialog', () => {
  it('renders the form values it receives', () => {
    const wrapper = mountDialog({
      form: makeForm({ theme: 'dark', readerMaxWidth: 'narrow', readerFont: 'serif', readerFontSize: '18' })
    })
    const selects = wrapper.findAll('select')
    expect((selects[0].element as HTMLSelectElement).value).toBe('dark')
    expect((selects[1].element as HTMLSelectElement).value).toBe('narrow')
    expect((selects[2].element as HTMLSelectElement).value).toBe('serif')
    expect((selects[3].element as HTMLSelectElement).value).toBe('18')
  })

  it('offers every font size the reader supports', () => {
    const wrapper = mountDialog()
    const sizes = wrapper.findAll('select')[3].findAll('option').map((o) => o.text())
    expect(sizes).toEqual(['13 px', '14 px', '15 px', '16 px', '17 px', '18 px', '19 px', '20 px'])
  })

  it('writes the edits back into the model', async () => {
    const form = makeForm()
    const wrapper = mountDialog({ form })
    await wrapper.findAll('select')[0].setValue('light')
    expect(form.theme).toBe('light')

    const number = wrapper.find('input[type="number"]')
    await number.setValue('30')
    expect(form.feedIntervalMin).toBe('30')

    const text = wrapper.find('input[type="text"]')
    await text.setValue('mi-tema')
    expect(form.ntfyTopic).toBe('mi-tema')
  })

  it('emits save and cancel from the footer buttons', async () => {
    const wrapper = mountDialog()
    const buttons = wrapper.findAll('button')
    await buttons.at(-1)!.trigger('click')
    expect(wrapper.emitted('save')).toHaveLength(1)

    await buttons.at(-2)!.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('disables the save button while saving', () => {
    const wrapper = mount(SettingsDialog, {
      props: { modelValue: makeForm(), saving: true },
      global: { stubs, mocks: { $gettext: (m: string) => m } }
    })
    expect(wrapper.findAll('button').at(-1)!.attributes('disabled')).toBeDefined()
  })

  it('emits the OPML and rules actions', async () => {
    const wrapper = mountDialog()
    const texts = wrapper.findAll('button').map((b) => b.text())
    const clickByText = async (label: string) => {
      const index = texts.findIndex((text) => text.includes(label))
      expect(index).toBeGreaterThanOrEqual(0)
      await wrapper.findAll('button')[index].trigger('click')
    }
    await clickByText('Export OPML')
    await clickByText('Import OPML')
    await clickByText('Global rules…')
    await clickByText('Auto-read rules…')
    expect(wrapper.emitted('exportOpml')).toHaveLength(1)
    expect(wrapper.emitted('importOpml')).toHaveLength(1)
    expect(wrapper.emitted('openGlobalRules')).toHaveLength(1)
    expect(wrapper.emitted('openAutoRead')).toHaveLength(1)
  })

  it('closes when the backdrop is clicked', async () => {
    const wrapper = mountDialog()
    await wrapper.find('[role="dialog"]').trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })
})
