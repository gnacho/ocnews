<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Keyboard shortcuts')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('close')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 460px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 12px; font-size: 15px; font-weight: 600">{{ $gettext('Keyboard shortcuts') }}</h3>
      <dl style="margin: 0; font-size: 13px; display: grid; grid-template-columns: 120px 1fr; gap: 6px 12px; max-height: 60vh; overflow-y: auto">
        <template v-for="shortcut in shortcuts" :key="shortcut.keys">
          <dt style="font-weight: 600"><code>{{ shortcut.keys }}</code></dt>
          <dd style="margin: 0; opacity: 0.85">{{ $gettext(shortcut.label) }}</dd>
        </template>
      </dl>
      <div style="display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px">
        <oc-button color-role="primary" appearance="filled" style="font-size: 13px" @click="emit('close')">
          {{ $gettext('Close') }}
        </oc-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Shortcut } from '../lib/shortcuts'

defineProps<{ shortcuts: Shortcut[] }>()

const emit = defineEmits<{ close: [] }>()
</script>
