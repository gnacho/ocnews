<template>
  <section style="display: flex; flex-direction: column; flex: 1; min-width: 0; border-left: 1px solid var(--news-border)">
    <header style="display: flex; align-items: center; gap: 6px; padding: 8px 16px; border-bottom: 1px solid var(--news-border)">
      <oc-button color-role="secondary" appearance="raw" :aria-label="$gettext('Back')" @click="emit('close')">
        <X style="width: 20px; height: 20px" />
      </oc-button>
      <h2 style="flex: 1; font-size: 17px; font-weight: 600; margin: 0">
        <a :href="item.url" target="_blank" rel="noopener noreferrer" style="color: inherit; text-decoration: none">
          {{ item.title }}
        </a>
      </h2>
      <oc-button
        v-if="!item.feedFullContent"
        color-role="secondary"
        appearance="raw"
        :aria-label="$gettext('Full article')"
        :title="fullBodyActive ? $gettext('Show summary') : $gettext('Full article')"
        :disabled="fullLoading"
        @click="emit('toggleFull')"
      >
        <BookOpenText style="width: 20px; height: 20px" :style="fullBodyActive ? 'color: var(--news-primary)' : ''" />
      </oc-button>
      <oc-button
        color-role="secondary"
        appearance="raw"
        :aria-label="item.unread ? $gettext('Mark as unread') : $gettext('Mark as read')"
        :title="item.unread ? $gettext('Mark as read') : $gettext('Mark as unread')"
        @click="emit('toggleUnread', item)"
      >
        <MailOpen style="width: 20px; height: 20px" />
      </oc-button>
      <oc-button
        color-role="secondary"
        appearance="raw"
        :aria-label="item.starred ? $gettext('Unstar') : $gettext('Star')"
        :title="item.starred ? $gettext('Unstar') : $gettext('Star')"
        @click="emit('toggleStar', item)"
      >
        <Star style="width: 20px; height: 20px" :fill="item.starred ? 'currentColor' : 'none'" />
      </oc-button>
      <oc-button
        color-role="secondary"
        appearance="raw"
        :aria-label="$gettext('Open in new tab')"
        :title="$gettext('Open in new tab')"
        @click="emit('openOriginal')"
      >
        <ExternalLink style="width: 20px; height: 20px" />
      </oc-button>
      <oc-button
        color-role="secondary"
        appearance="raw"
        :aria-label="shared ? $gettext('Stop sharing') : $gettext('Share article')"
        :title="shared ? $gettext('Stop sharing') : $gettext('Share article')"
        @click="emit('toggleShare')"
      >
        <Share2 style="width: 20px; height: 20px" :style="shared ? 'color: var(--news-primary)' : ''" />
      </oc-button>
    </header>
    <div style="flex: 1; overflow-y: auto; padding: 12px 16px">
      <audio
        v-if="item.enclosureMime?.startsWith('audio/') && item.enclosureLink"
        :src="item.enclosureLink"
        controls
        preload="metadata"
        style="width: 100%; margin: 0 0 12px"
      />
      <video
        v-else-if="item.enclosureMime?.startsWith('video/') && item.enclosureLink"
        :src="item.enclosureLink"
        controls
        preload="metadata"
        style="width: 100%; border-radius: 8px; margin: 0 0 12px; max-height: 60vh; background: #000"
      />
      <p style="font-size: 12px; opacity: 0.6; margin: 0 0 12px; display: flex; align-items: center; gap: 4px">
        <img v-if="icon" :src="icon" alt="" width="14" height="14" loading="lazy" style="width: 14px; height: 14px; border-radius: 3px; object-fit: contain" />
        <span>{{ feedName }}<span v-if="item.author"> · {{ item.author }}</span> · {{ relative }}</span>
        <span v-if="fullLoading" style="opacity: 0.7"> · {{ $gettext('loading full article…') }}</span>
        <span v-else-if="fullFailed" style="color: var(--news-warning)">
          · {{ $gettext('full article not available — open the original') }}
        </span>
      </p>
      <!-- body sanitizado server-side; imágenes via proxy propio (CSP) -->
      <div
        data-testid="reader-body"
        class="oc-prose news-body"
        :class="reader.class"
        :style="{ maxWidth: reader.maxWidth, fontFamily: reader.fontFamily, fontSize: reader.fontSizePx, ['--news-bg' as any]: reader.bg, ['--news-fg' as any]: reader.fg }"
        v-html="body"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { BookOpenText, ExternalLink, MailOpen, Share2, Star, X } from 'lucide-vue-next'
import { useGettext } from 'vue3-gettext'

import type { Feed, Item } from '../api'
import { feedIcon, feedTitle, relativeLabel } from '../lib/format'

export interface ReaderStyle {
  class: string
  maxWidth: string
  fontFamily: string
  fontSizePx: string
  bg: string
  fg: string
}

const props = defineProps<{
  item: Item
  feeds: Feed[]
  body: string
  reader: ReaderStyle
  shared: boolean
  fullLoading?: boolean
  fullFailed?: boolean
  fullBodyActive?: boolean
}>()

const emit = defineEmits<{
  close: []
  toggleFull: []
  toggleUnread: [item: Item]
  toggleStar: [item: Item]
  toggleShare: []
  openOriginal: []
}>()

const { $gettext, $ngettext } = useGettext()

const icon = computed(() => feedIcon(props.feeds, props.item.feedId))
const feedName = computed(() => feedTitle(props.feeds, props.item.feedId))
const relative = computed(() =>
  relativeLabel(props.item.pubDate, Date.now(), {
    t: $gettext,
    n: $ngettext,
    date: (value) =>
      new Date(value * 1000).toLocaleDateString(undefined, {
        day: 'numeric',
        month: 'short',
        hour: '2-digit',
        minute: '2-digit'
      })
  })
)
</script>
