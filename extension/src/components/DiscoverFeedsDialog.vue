<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Select a feed')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('cancel')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 440px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">{{ $gettext('Select a feed') }}</h3>
      <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.6">
        {{ $gettext('Multiple feeds were found on this site.') }}
      </p>
      <fieldset style="border: 0; margin: 0 0 16px; padding: 0; max-height: 260px; overflow-y: auto">
        <label
          v-for="(candidate, index) in candidates"
          :key="candidate.url"
          style="display: flex; align-items: center; gap: 8px; padding: 6px 4px; border-radius: 6px; cursor: pointer; font-size: 13px"
        >
          <input v-model="selected" type="radio" name="discover-feed" data-testid="discover-option" :value="index" />
          <span style="flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
            {{ candidate.title || candidate.url }}
          </span>
          <span style="font-size: 11px; opacity: 0.5; text-transform: uppercase">{{ candidate.type }}</span>
        </label>
      </fieldset>
      <div style="display: flex; gap: 8px; justify-content: flex-end">
        <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="emit('cancel')">
          {{ $gettext('Cancel') }}
        </oc-button>
        <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="submitting" @click="emit('confirm')">
          {{ $gettext('Subscribe') }}
        </oc-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DiscoveredFeed } from '../api'

defineProps<{ candidates: DiscoveredFeed[]; submitting?: boolean }>()

const selected = defineModel<number>({ required: true })

const emit = defineEmits<{ confirm: []; cancel: [] }>()
</script>
