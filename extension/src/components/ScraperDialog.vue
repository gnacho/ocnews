<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Article scraper')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('cancel')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 420px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">{{ $gettext('Article scraper') }}</h3>
      <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.65; line-height: 1.5">
        {{ $gettext('CSS selector of the article body on the site, used before the automatic extraction when the feed only has summaries. Leave empty to use the automatic extraction.') }}
      </p>
      <input
        v-model="selector"
        type="text"
        spellcheck="false"
        data-testid="scraper-selector"
        placeholder="div#articleBody, article"
        style="width: 100%; box-sizing: border-box; font-size: 12px; font-family: ui-monospace, Menlo, Consolas, monospace; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit; margin-bottom: 16px"
      />
      <div style="display: flex; gap: 8px; justify-content: flex-end; align-items: center">
        <oc-button v-if="selector" color-role="secondary" appearance="raw" style="font-size: 13px; color: var(--news-error)" @click="emit('clear')">
          {{ $gettext('Clear') }}
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
defineProps<{ saving?: boolean }>()

const selector = defineModel<string>({ required: true })

const emit = defineEmits<{ confirm: []; cancel: []; clear: [] }>()
</script>
