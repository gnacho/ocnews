<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Settings')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('cancel')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 400px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 16px; font-size: 15px; font-weight: 600">{{ $gettext('News settings') }}</h3>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Reader theme') }}</label>
      <select v-model="form.theme" style="width: 100%; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 12px">
        <option value="system">{{ $gettext('System') }}</option>
        <option value="light">{{ $gettext('Light') }}</option>
        <option value="dark">{{ $gettext('Dark') }}</option>
      </select>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Reader width') }}</label>
      <select v-model="form.readerMaxWidth" style="width: 100%; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 12px">
        <option value="narrow">{{ $gettext('Narrow') }}</option>
        <option value="normal">{{ $gettext('Normal') }}</option>
        <option value="wide">{{ $gettext('Wide') }}</option>
      </select>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Font') }}</label>
      <select v-model="form.readerFont" style="width: 100%; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 12px">
        <option value="default">{{ $gettext('Default') }}</option>
        <option value="serif">{{ $gettext('Serif') }}</option>
        <option value="sans">{{ $gettext('Sans-serif') }}</option>
        <option value="mono">{{ $gettext('Monospace') }}</option>
      </select>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Font size') }}</label>
      <select v-model="form.readerFontSize" style="width: 100%; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 12px">
        <option v-for="n in [13, 14, 15, 16, 17, 18, 19, 20]" :key="n" :value="String(n)">{{ n }} px</option>
      </select>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Refresh interval (minutes)') }}</label>
      <input
        :value="form.feedIntervalMin"
        type="number"
        min="5"
        max="1440"
        placeholder=""
        style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 16px"
        @input="setInterval"
      />
      <p style="margin: 0 0 16px; font-size: 11px; opacity: 0.6">{{ $gettext('Leave empty for the server default (15 min).') }}</p>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Notification topic (ntfy)') }}</label>
      <input
        v-model="form.ntfyTopic"
        type="text"
        spellcheck="false"
        placeholder="ocnews"
        style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 4px"
      />
      <p style="margin: 0 0 16px; font-size: 11px; opacity: 0.6">{{ $gettext('Get a push notification when new articles arrive. Leave empty to use the server default (or none).') }}</p>
      <div style="display: flex; gap: 8px; align-items: center; margin-bottom: 16px">
        <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="emit('exportOpml')">
          <Download style="width: 16px; height: 16px" />&nbsp;{{ $gettext('Export OPML') }}
        </oc-button>
        <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="emit('importOpml')">
          <Upload style="width: 16px; height: 16px" />&nbsp;{{ $gettext('Import OPML') }}
        </oc-button>
        <span style="font-size: 11px; opacity: 0.6">{{ $gettext('Back up or restore your subscriptions.') }}</span>
      </div>
      <oc-button color-role="secondary" appearance="outline" style="font-size: 13px; width: 100%; justify-content: flex-start; margin-bottom: 8px" @click="emit('openGlobalRules')">
        <Filter style="width: 16px; height: 16px" />&nbsp;{{ $gettext('Global rules…') }}
      </oc-button>
      <oc-button color-role="secondary" appearance="outline" style="font-size: 13px; width: 100%; justify-content: flex-start; margin-bottom: 16px" @click="emit('openAutoRead')">
        <CheckCheck style="width: 16px; height: 16px" />&nbsp;{{ $gettext('Auto-read rules…') }}
      </oc-button>
      <div style="display: flex; gap: 8px; justify-content: flex-end">
        <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="emit('cancel')">
          {{ $gettext('Cancel') }}
        </oc-button>
        <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="saving" @click="emit('save')">
          {{ $gettext('Save') }}
        </oc-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CheckCheck, Download, Filter, Upload } from 'lucide-vue-next'

import type { UserSettings } from '../api'

const form = defineModel<UserSettings>({ required: true })

// v-model on a number input hands back a number and the API expects the
// string the user typed (saveSettings trims it), so the value is read off the
// event instead.
function setInterval(event: Event) {
  form.value.feedIntervalMin = (event.target as HTMLInputElement).value
}
defineProps<{ saving?: boolean }>()

const emit = defineEmits<{
  save: []
  cancel: []
  exportOpml: []
  importOpml: []
  openGlobalRules: []
  openAutoRead: []
}>()
</script>
