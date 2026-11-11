<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Feed credentials')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('cancel')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 380px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 12px; font-size: 15px; font-weight: 600">{{ $gettext('Feed credentials') }}</h3>
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Username') }}</label>
      <input
        v-model="user"
        type="text"
        autocomplete="off"
        data-testid="cred-user"
        style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit; margin-bottom: 10px"
      />
      <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Password') }}</label>
      <input
        v-model="pass"
        type="password"
        autocomplete="new-password"
        data-testid="cred-pass"
        style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit; margin-bottom: 4px"
      />
      <p style="margin: 0 0 16px; font-size: 11px; opacity: 0.6">{{ $gettext('Leave the password empty to keep the current one.') }}</p>
      <div style="display: flex; gap: 8px; justify-content: flex-end; align-items: center">
        <oc-button v-if="hasAuth" color-role="secondary" appearance="raw" style="font-size: 13px; color: var(--news-error)" @click="emit('remove')">
          {{ $gettext('Remove credentials') }}
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
defineProps<{ hasAuth: boolean; saving?: boolean }>()

const user = defineModel<string>('user', { required: true })
const pass = defineModel<string>('pass', { required: true })

const emit = defineEmits<{ confirm: []; remove: []; cancel: [] }>()
</script>
