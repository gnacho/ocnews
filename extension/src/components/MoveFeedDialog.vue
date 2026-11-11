<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Move to folder')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('cancel')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 360px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 12px; font-size: 15px; font-weight: 600">{{ $gettext('Move to folder') }}</h3>
      <select
        v-model.number="folderId"
        data-testid="folder-select"
        style="width: 100%; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); margin-bottom: 16px"
      >
        <option :value="0">{{ $gettext('No folder') }}</option>
        <option v-for="folder in folders" :key="folder.id" :value="folder.id">{{ folder.name }}</option>
      </select>
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
import type { Folder } from '../api'

defineProps<{ folders: Folder[]; saving?: boolean }>()

const folderId = defineModel<number>({ required: true })

const emit = defineEmits<{ confirm: []; cancel: [] }>()
</script>
