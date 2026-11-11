<template>
  <section
    style="display: flex; flex-direction: column; flex-shrink: 0; min-width: 0; border-right: 1px solid var(--news-border-lighter)"
    :style="{ width: width + 'px' }"
  >
    <header
      style="display: flex; flex-direction: column; gap: 8px; padding: 8px 16px; border-bottom: 1px solid var(--news-border); flex-shrink: 0"
    >
      <div style="display: flex; align-items: center; gap: 12px; min-width: 0">
        <h1
          style="font-size: 14px; font-weight: 600; margin: 0; flex: 1; min-width: 0; display: flex; align-items: center; gap: 6px"
        >
          <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap">{{ title }}</span>
          <span v-if="unreadCount > 0" style="font-weight: 400; opacity: 0.6; flex-shrink: 0">· {{ unreadCount }}</span>
          <oc-button
            v-if="unreadCount > 0"
            color-role="secondary"
            appearance="raw"
            style="flex-shrink: 0; padding: 1px"
            :aria-label="$gettext('Mark all read')"
            :title="$gettext('Mark all read')"
            @click="emit('markAllRead')"
          >
            <CheckCheck style="width: 14px; height: 14px" />
          </oc-button>
        </h1>
        <label style="font-size: 12px; display: flex; align-items: center; gap: 4px">
          {{ $gettext('Show') }}
          <select v-model="showAll" style="font-size: 12px; padding: 2px 4px" :aria-label="$gettext('Show')">
            <option :value="false">{{ $gettext('Unread') }}</option>
            <option :value="true">{{ $gettext('All') }}</option>
          </select>
        </label>
      </div>
      <div style="display: flex; align-items: center; gap: 12px; min-width: 0; flex-wrap: wrap">
        <label style="font-size: 12px; display: flex; align-items: center; gap: 4px">
          {{ $gettext('Order') }}
          <select v-model="oldestFirst" style="font-size: 12px; padding: 2px 4px" :aria-label="$gettext('Order')">
            <option :value="false">{{ $gettext('Newest first') }}</option>
            <option :value="true">{{ $gettext('Oldest first') }}</option>
          </select>
        </label>
        <div style="display: flex; align-items: center; gap: 6px; min-width: 0; flex: 1">
          <Search style="width: 15px; height: 15px; opacity: 0.5; flex-shrink: 0" />
          <input
            ref="searchInputEl"
            v-model="searchQuery"
            type="search"
            :placeholder="$gettext('Search articles…')"
            :aria-label="$gettext('Search articles')"
            style="flex: 1; min-width: 0; font-size: 13px; padding: 4px 8px; border-radius: 6px; border: 1px solid var(--news-input-border); background: transparent; color: inherit"
            @keydown.enter="emit('search')"
          />
          <button
            v-if="searchQuery"
            style="background: none; border: 0; cursor: pointer; padding: 2px; display: inline-flex"
            :aria-label="$gettext('Clear search')"
            :title="$gettext('Clear search')"
            @click="emit('clearSearch')"
          >
            <X style="width: 14px; height: 14px" />
          </button>
          <oc-button
            v-if="searchActive"
            color-role="secondary"
            appearance="raw"
            style="font-size: 12px; flex-shrink: 0"
            :aria-label="$gettext('Save search')"
            :title="$gettext('Save search')"
            @click="emit('saveSearch')"
          >
            <BookmarkPlus style="width: 15px; height: 15px" />&nbsp;{{ $gettext('Save') }}
          </oc-button>
        </div>
      </div>
    </header>

    <p
      v-if="error"
      style="padding: 8px 16px; color: var(--news-error); font-size: 13px; border-bottom: 1px solid var(--news-border-light); margin: 0"
    >
      {{ error }}
    </p>
    <p v-if="loading" style="padding: 32px 16px; opacity: 0.6; font-size: 13px">{{ $gettext('Loading…') }}</p>
    <p v-else-if="items.length === 0" style="padding: 32px 16px; opacity: 0.6; font-size: 13px">
      {{ $gettext('No articles') }}
    </p>

    <ul v-else style="flex: 1; overflow-y: auto; list-style: none; margin: 0; padding: 0">
      <li
        v-for="item in items"
        :key="item.id"
        data-testid="note-row"
        style="display: flex; gap: 12px; padding: 10px 16px; border-bottom: 1px solid var(--news-border-lighter); cursor: pointer"
        :style="{
          background: selectedId === item.id ? 'var(--news-hover-bg)' : 'transparent',
          opacity: item.unread ? 1 : 0.65
        }"
        @click="emit('select', item)"
      >
        <div style="flex: 1; min-width: 0">
          <p style="margin: 0; font-size: 17px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
            {{ item.title }}
          </p>
          <p
            v-if="snippet(item)"
            style="margin: 3px 0 0; font-size: 13px; opacity: 0.75; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden"
          >
            {{ snippet(item) }}
          </p>
          <p style="margin: 3px 0 0; font-size: 12px; opacity: 0.6; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
            <img v-if="iconOf(item.feedId)" :src="iconOf(item.feedId)" alt="" width="12" height="12" loading="lazy" style="width: 12px; height: 12px; border-radius: 3px; object-fit: contain; vertical-align: -1px; margin-right: 4px" />
            {{ titleOf(item.feedId) }}<span v-if="item.author"> · {{ item.author }}</span> · {{ relative(item.pubDate) }}
            <span
              v-if="item.clusterSize && item.clusterSize > 1"
              :title="cluster(item)"
              style="display: inline-flex; align-items: center; gap: 3px; margin-left: 6px; font-weight: 600; opacity: 0.8"
            >
              <Copy style="width: 11px; height: 11px" />{{ item.clusterSize }}
            </span>
          </p>
        </div>
        <button
          style="align-self: flex-start; background: none; border: 0; cursor: pointer; padding: 2px"
          :style="{ color: item.unread ? 'var(--news-primary)' : 'var(--news-muted)' }"
          :aria-label="item.unread ? $gettext('Mark as read') : $gettext('Mark as unread')"
          :title="item.unread ? $gettext('Mark as read') : $gettext('Mark as unread')"
          @click.stop="emit('toggleUnread', item)"
        >
          <component :is="item.unread ? Mail : MailOpen" style="width: 16px; height: 16px" />
        </button>
        <button
          style="align-self: flex-start; background: none; border: 0; cursor: pointer; padding: 2px"
          :style="{ color: item.starred ? 'var(--news-warning)' : 'var(--news-muted)' }"
          :aria-label="item.starred ? $gettext('Unstar') : $gettext('Star')"
          @click.stop="emit('toggleStar', item)"
        >
          <Star style="width: 16px; height: 16px" :fill="item.starred ? 'currentColor' : 'none'" />
        </button>
      </li>
      <li v-if="moreAvailable" style="padding: 12px; text-align: center; list-style: none">
        <oc-button color-role="secondary" appearance="filled" @click="emit('loadMore')">
          {{ $gettext('Load more') }}
        </oc-button>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { BookmarkPlus, CheckCheck, Copy, Mail, MailOpen, Search, Star, X } from 'lucide-vue-next'
import { useGettext } from 'vue3-gettext'

import type { Feed, Item } from '../api'
import { clusterTooltip, feedIcon, feedTitle, itemSnippet, relativeLabel } from '../lib/format'

const props = defineProps<{
  items: Item[]
  feeds: Feed[]
  loading: boolean
  error: string
  title: string
  unreadCount: number
  searchActive: boolean
  selectedId: number | null
  moreAvailable: boolean
  width: number
}>()

const showAll = defineModel<boolean>('showAll', { required: true })
const oldestFirst = defineModel<boolean>('oldestFirst', { required: true })
const searchQuery = defineModel<string>('searchQuery', { required: true })

const emit = defineEmits<{
  select: [item: Item]
  toggleUnread: [item: Item]
  toggleStar: [item: Item]
  markAllRead: []
  search: []
  clearSearch: []
  saveSearch: []
  loadMore: []
}>()

const { $gettext, $ngettext } = useGettext()
const searchInputEl = ref<HTMLInputElement | null>(null)

function snippet(item: Item) {
  return itemSnippet(item)
}

function titleOf(feedId: number) {
  return feedTitle(props.feeds, feedId)
}

function iconOf(feedId: number) {
  return feedIcon(props.feeds, feedId)
}

function cluster(item: Item) {
  return clusterTooltip(item.clusterSize, $gettext)
}

function relative(epoch: number) {
  return relativeLabel(epoch, Date.now(), {
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
}

function focusSearch() {
  searchInputEl.value?.focus()
}

defineExpose({ focusSearch })
</script>
