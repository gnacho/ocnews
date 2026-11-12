<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Filter rules')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('cancel')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 520px; max-width: 94vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">
        {{ title }}
      </h3>
      <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.65; line-height: 1.5">
        {{ $gettext('One rule per line: Field=regex (RE2). Fields: EntryTitle, EntryURL, EntryAuthor, EntryContent, EntryDate. For EntryDate: future, before:YYYY-MM-DD, after:YYYY-MM-DD, between:A,B or max-age:7d.') }}
      </p>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Block rules (matching articles are hidden)') }}</label>
      <textarea
        v-model="form.block"
        rows="4"
        spellcheck="false"
        data-testid="rules-block"
        style="width: 100%; box-sizing: border-box; font-size: 12px; font-family: ui-monospace, Menlo, Consolas, monospace; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit; margin-bottom: 12px"
      ></textarea>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Keep rules (only matching articles are kept)') }}</label>
      <textarea
        v-model="form.keep"
        rows="4"
        spellcheck="false"
        data-testid="rules-keep"
        style="width: 100%; box-sizing: border-box; font-size: 12px; font-family: ui-monospace, Menlo, Consolas, monospace; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit; margin-bottom: 16px"
      ></textarea>
      <div style="display: flex; gap: 8px; justify-content: flex-end; align-items: center">
        <oc-button v-if="hasRules" color-role="secondary" appearance="raw" style="font-size: 13px; color: var(--news-error)" @click="emit('clear')">
          {{ $gettext('Clear rules') }}
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
import type { Rules } from '../api'

defineProps<{ title: string; hasRules: boolean; saving?: boolean }>()

const form = defineModel<Rules>({ required: true })

const emit = defineEmits<{ confirm: []; cancel: []; clear: [] }>()
</script>
