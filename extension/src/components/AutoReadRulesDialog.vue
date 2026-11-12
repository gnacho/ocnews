<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Auto-read rules')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('close')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 460px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">{{ $gettext('Auto-read rules') }}</h3>
      <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.65">
        {{ $gettext('Articles whose title matches a rule are marked as read automatically when they arrive.') }}
      </p>
      <div v-if="rules.length" data-testid="auto-read-list" style="margin-bottom: 12px; max-height: 200px; overflow-y: auto">
        <div
          v-for="rule in rules"
          :key="rule.id"
          style="display: flex; align-items: center; gap: 8px; padding: 5px 6px; border-radius: 6px; font-size: 13px"
        >
          <span style="flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
            <span style="opacity: 0.6">{{ feedLabel(rule.feedId) }}</span>
            <code style="font-size: 12px">{{ rule.titlePattern }}</code>
          </span>
          <button
            style="background: none; border: 0; cursor: pointer; padding: 2px; display: inline-flex"
            :aria-label="$gettext('Delete rule')"
            :title="$gettext('Delete rule')"
            @click="emit('remove', rule)"
          >
            <X style="width: 15px; height: 15px; color: var(--news-muted)" />
          </button>
        </div>
      </div>
      <div style="display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px">
        <label style="display: block; font-size: 12px; margin-bottom: -6px">{{ $gettext('New rule (regex on the title)') }}</label>
        <div style="display: flex; gap: 8px; align-items: center">
          <input
            v-model="pattern"
            type="text"
            spellcheck="false"
            data-testid="auto-read-pattern"
            :placeholder="'(?i)urgente'"
            style="flex: 1; min-width: 0; font-size: 12px; font-family: ui-monospace, Menlo, Consolas, monospace; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit"
            @keydown.enter="emit('add')"
          />
          <select v-model.number="feedId" data-testid="auto-read-feed" style="font-size: 12px; padding: 5px 4px" :aria-label="$gettext('Feed')">
            <option :value="0">{{ $gettext('All feeds') }}</option>
            <option v-for="feed in feeds" :key="feed.id" :value="feed.id">{{ feed.title }}</option>
          </select>
          <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="saving" @click="emit('add')">
            {{ $gettext('Add') }}
          </oc-button>
        </div>
      </div>
      <div style="display: flex; gap: 8px; justify-content: flex-end">
        <oc-button color-role="primary" appearance="filled" style="font-size: 13px" @click="emit('close')">
          {{ $gettext('Close') }}
        </oc-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { useGettext } from 'vue3-gettext'

import type { AutoReadRule, Feed } from '../api'

const props = defineProps<{ rules: AutoReadRule[]; feeds: Feed[]; saving?: boolean }>()

const { $gettext } = useGettext()

const pattern = defineModel<string>('pattern', { required: true })
const feedId = defineModel<number>('feedId', { required: true })

const emit = defineEmits<{ add: []; remove: [rule: AutoReadRule]; close: [] }>()

function feedLabel(id: number) {
  if (id === 0) return $gettext('All feeds') + ' · '
  const feed = props.feeds.find((item) => item.id === id)
  return (feed?.title ?? `#${id}`) + ' · '
}
</script>
