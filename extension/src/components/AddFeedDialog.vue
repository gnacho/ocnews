<template>
  <div
    role="dialog"
    aria-modal="true"
    :aria-label="$gettext('Add feed')"
    style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
    @click.self="emit('cancel')"
  >
    <div style="background: var(--news-bg); border-radius: 12px; padding: 20px; width: 420px; max-width: 92vw; box-shadow: 0 8px 32px var(--news-shadow); color: var(--news-fg)">
      <h3 style="margin: 0 0 12px; font-size: 15px; font-weight: 600">{{ $gettext('Add feed') }}</h3>
      <label for="news-add-feed-url" style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Feed URL') }}</label>
      <input
        id="news-add-feed-url"
        ref="urlEl"
        v-model="url"
        type="url"
        data-testid="add-feed-url"
        :placeholder="$gettext('https://site.example/feed')"
        style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit; margin-bottom: 10px"
        @keydown.enter="emit('confirm')"
      />
      <label style="display: flex; align-items: center; gap: 6px; font-size: 12px; cursor: pointer; opacity: 0.8">
        <input v-model="auth" type="checkbox" data-testid="add-feed-auth" />
        {{ $gettext('Requires authentication') }}
      </label>
      <div v-if="auth" style="display: flex; flex-direction: column; gap: 6px; margin-top: 6px">
        <input
          v-model="user"
          type="text"
          autocomplete="off"
          data-testid="add-feed-user"
          :placeholder="$gettext('Username')"
          :aria-label="$gettext('Username')"
          style="font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit"
        />
        <input
          v-model="pass"
          type="password"
          autocomplete="new-password"
          data-testid="add-feed-pass"
          :placeholder="$gettext('Password')"
          :aria-label="$gettext('Password')"
          style="font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit"
          @keydown.enter="emit('confirm')"
        />
      </div>
      <p v-if="error" data-testid="add-feed-error" style="margin: 12px 0 0; font-size: 12px; color: var(--news-error)">{{ error }}</p>
      <div style="display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px">
        <oc-button color-role="secondary" appearance="outline" style="font-size: 13px" @click="emit('cancel')">
          {{ $gettext('Cancel') }}
        </oc-button>
        <oc-button color-role="primary" appearance="filled" style="font-size: 13px" :disabled="!url.trim() || submitting" @click="emit('confirm')">
          {{ $gettext('Subscribe') }}
        </oc-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'

defineProps<{ error?: string; submitting?: boolean }>()

const url = defineModel<string>('url', { required: true })
const auth = defineModel<boolean>('auth', { required: true })
const user = defineModel<string>('user', { required: true })
const pass = defineModel<string>('pass', { required: true })

const emit = defineEmits<{ confirm: []; cancel: [] }>()

const urlEl = ref<HTMLInputElement | null>(null)

onMounted(() => urlEl.value?.focus())
</script>
