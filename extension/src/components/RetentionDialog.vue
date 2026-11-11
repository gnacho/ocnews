<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Article retention')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('cancel')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 380px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">{{ $gettext('Article retention') }}</h3>
      <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.6">
        {{ $gettext('Days to keep read, non-starred articles for this feed. 0 = use the server default.') }}
      </p>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Retention (days)') }}</label>
      <input
        :value="days"
        type="number"
        min="0"
        max="3650"
        data-testid="retention-days"
        style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 16px"
        @input="setDays"
      />
      <div style="display: flex; gap: 8px; justify-content: flex-end">
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

const days = defineModel<number>({ required: true })

const emit = defineEmits<{ confirm: []; cancel: [] }>()

// Read off the event so an empty field becomes 0 instead of NaN.
function setDays(event: Event) {
  const raw = (event.target as HTMLInputElement).value
  const parsed = Number(raw)
  days.value = raw === '' || Number.isNaN(parsed) ? 0 : parsed
}
</script>
