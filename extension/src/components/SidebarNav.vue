<template>
  <aside class="news-sidebar">
    <!-- Add feed: solo el botón; URL y credenciales se piden al pulsar (#49) -->
    <div class="news-sidebar-top">
      <button class="news-cta" type="button" @click="emit('addFeed')">
        <Plus style="width: 16px; height: 16px" />
        {{ $gettext('Add feed') }}
      </button>
    </div>

    <!-- Sidebar nav -->
    <nav class="news-sidebar-nav">
      <div class="news-section">
        <span class="news-section-label">{{ $gettext('Subscriptions') }}</span>
        <span style="display: flex; align-items: center; gap: 4px">
          <oc-button
            color-role="secondary"
            appearance="raw"
            :aria-label="$gettext('Refresh')"
            :title="$gettext('Refresh')"
            :disabled="refreshing"
            @click="emit('refresh')"
          >
            <RefreshCw style="width: 15px; height: 15px" :style="refreshing ? 'animation: news-spin 1s linear infinite' : ''" />
          </oc-button>
          <oc-button
            color-role="secondary"
            appearance="raw"
            :aria-label="$gettext('New folder')"
            :title="$gettext('New folder')"
            @click="emit('createFolder')"
          >
            <FolderPlus style="width: 16px; height: 16px" />
          </oc-button>
        </span>
      </div>

      <template v-for="entry in entries" :key="entry.key">
        <div
          class="news-nav-entry"
          data-testid="nav-entry"
          :data-kind="entry.kind"
          :style="{
            background: entry.key === activeKey ? 'var(--oc-role-secondary-container)' : 'transparent',
            paddingLeft: 8 + (entry.depth ?? 0) * 20 + 'px',
            fontWeight: entry.unread ? 600 : 400
          }"
          @mouseenter="hovered = entry.key"
          @mouseleave="hovered = ''"
          @click="emit('select', entry)"
        >
          <img
            v-if="entry.kind === 'feed' && entry.favicon"
            :src="entry.favicon"
            alt=""
            width="16"
            height="16"
            loading="lazy"
            style="width: 16px; height: 16px; border-radius: 4px; object-fit: contain; flex-shrink: 0"
          />
          <component :is="entry.icon" v-else style="width: 16px; height: 16px; opacity: 0.7; flex-shrink: 0" />
          <span style="flex: 1; min-width: 0; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
            {{ entry.label }}
          </span>
          <span v-if="entry.error" :title="entry.error" style="flex-shrink: 0; display: inline-flex">
            <AlertTriangle style="width: 14px; height: 14px; color: var(--news-warning)" />
          </span>
          <span v-if="entry.unread" style="font-size: 12px; opacity: 0.6; flex-shrink: 0">{{ entry.unread }}</span>
          <oc-button
            v-if="entry.kind === 'feed' || entry.kind === 'folder' || entry.kind === 'search'"
            color-role="secondary"
            appearance="raw"
            style="flex-shrink: 0"
            :aria-label="$gettext('Options')"
            :title="$gettext('Options')"
            @click.stop="openMenu = openMenu === entry.key ? '' : entry.key"
          >
            <MoreHorizontal style="width: 16px; height: 16px" />
          </oc-button>
        </div>

        <!-- Context menu -->
        <div
          v-if="openMenu === entry.key"
          data-testid="context-menu"
          style="margin: 2px 8px; border: 1px solid var(--news-border-medium); border-radius: 8px; padding: 4px; display: flex; flex-direction: column; gap: 2px; background: var(--news-bg); box-shadow: 0 4px 12px var(--news-shadow)"
        >
          <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Rename')" @click="emit('rename', entry)" />
          <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Move to folder…')" @click="emit('move', entry)" />
          <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Credentials…')" @click="emit('credentials', entry)" />
          <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Filter articles…')" @click="emit('filter', entry)" />
          <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Rules…')" @click="emit('rules', entry)" />
          <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Scraper…')" @click="emit('scraper', entry)" />
          <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Retention…')" @click="emit('retention', entry)" />
          <MenuBtn v-if="entry.kind === 'folder'" :label="$gettext('Rename')" @click="emit('renameFolder', entry)" />
          <MenuBtn v-if="entry.kind === 'folder'" :label="$gettext('New subfolder…')" @click="emit('newSubfolder', entry)" />
          <MenuBtn
            v-if="entry.kind === 'feed' || entry.kind === 'folder'"
            :label="$gettext('Mark all read')"
            @click="emit('markRead', entry)"
          />
          <MenuBtn
            v-if="entry.kind === 'feed' || entry.kind === 'folder' || entry.kind === 'search'"
            :label="$gettext('Delete')"
            danger
            @click="emit('delete', entry)"
          />
        </div>
      </template>
    </nav>

    <!-- Pie: Ajustes (OPML vive dentro del diálogo de ajustes) + atajos -->
    <div class="news-sidebar-bottom">
      <button class="news-quiet-action" type="button" @click="emit('settings')">
        <Settings style="width: 16px; height: 16px" />
        {{ $gettext('News settings') }}
      </button>
      <button class="news-quiet-action" type="button" @click="emit('shortcuts')">
        <Keyboard style="width: 16px; height: 16px" />
        {{ $gettext('Keyboard shortcuts') }}
      </button>
      <input
        ref="opmlInputEl"
        type="file"
        accept=".opml,.xml,text/xml,text/x-opml"
        style="display: none"
        @change="onOpmlFile"
      />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { AlertTriangle, FolderPlus, Keyboard, MoreHorizontal, Plus, RefreshCw, Settings } from 'lucide-vue-next'

import MenuBtn from './MenuBtn.vue'
import type { NavEntry } from '../lib/nav'

defineProps<{ entries: NavEntry[]; activeKey: string; refreshing?: boolean }>()

const openMenu = defineModel<string>('openMenu', { required: true })

const emit = defineEmits<{
  addFeed: []
  refresh: []
  createFolder: []
  select: [entry: NavEntry]
  rename: [entry: NavEntry]
  move: [entry: NavEntry]
  credentials: [entry: NavEntry]
  filter: [entry: NavEntry]
  rules: [entry: NavEntry]
  scraper: [entry: NavEntry]
  retention: [entry: NavEntry]
  renameFolder: [entry: NavEntry]
  newSubfolder: [entry: NavEntry]
  markRead: [entry: NavEntry]
  delete: [entry: NavEntry]
  settings: []
  shortcuts: []
  importOpml: [file: File]
}>()

const hovered = ref('')
const opmlInputEl = ref<HTMLInputElement | null>(null)

function onOpmlFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) emit('importOpml', file)
  input.value = ''
}

function openOpmlPicker() {
  opmlInputEl.value?.click()
}

defineExpose({ hovered, openOpmlPicker })
</script>
