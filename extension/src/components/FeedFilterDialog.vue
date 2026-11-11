<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Filter articles')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('cancel')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 420px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">
        {{ $gettext('Filter articles') }}
      </h3>
      <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.6">
        {{ $gettext('Hide articles whose title, body or URL contains any keyword (comma-separated, case-insensitive).') }}
      </p>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Title keywords') }}</label>
      <input
        v-model="form.titleKeywords"
        type="text"
        data-testid="filter-title"
        placeholder="sponsored,ads,offer"
        style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 10px"
      />
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Body keywords') }}</label>
      <input
        v-model="form.bodyKeywords"
        type="text"
        data-testid="filter-body"
        placeholder="tracking,cookie"
        style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 10px"
      />
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('URL keywords') }}</label>
      <input
        v-model="form.urlKeywords"
        type="text"
        data-testid="filter-url"
        placeholder="utm_,/tag/"
        style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 16px"
      />
      <div style="display: flex; gap: 8px; justify-content: flex-end; align-items: center">
        <oc-button v-if="hasFilter" color-role="secondary" appearance="raw" style="font-size: 13px" @click="emit('clear')">
          {{ $gettext('Clear filter') }}
        </oc-button>
        <span style="flex: 1"></span>
        <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="emit('cancel')">
          {{ $gettext('Cancel') }}
        </oc-button>
        <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="saving" @click="emit('confirm')">
          {{ $gettext('Save') }}
        </oc-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FeedFilter } from '../api'

defineProps<{ hasFilter: boolean; saving?: boolean }>()

const form = defineModel<Omit<FeedFilter, 'feedId'>>({ required: true })

const emit = defineEmits<{ confirm: []; cancel: []; clear: [] }>()
</script>
