<template>
  <main style="display: flex; height: 100%; width: 100%; overflow: hidden">
    <!-- Sidebar -->
    <aside
      style="display: flex; flex-direction: column; width: 280px; flex-shrink: 0; border-right: 1px solid rgba(125, 125, 125, 0.25); overflow: hidden"
    >
      <!-- Add feed -->
      <div style="padding: 12px; border-bottom: 1px solid rgba(125, 125, 125, 0.2)">
        <label
          for="news-add-feed"
          style="display: block; font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.04em; opacity: 0.65; margin-bottom: 6px"
        >
          {{ $gettext('Add feed') }}
        </label>
        <div style="display: flex; gap: 8px; align-items: center">
          <input
            id="news-add-feed"
            v-model="newFeedUrl"
            type="url"
            :placeholder="$gettext('https://site.example/feed')"
            :aria-label="$gettext('Feed URL')"
            style="flex: 1; min-width: 0; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid #94a3b8; background: transparent; color: inherit"
            @keydown.enter="subscribeFeed"
          />
          <oc-button
            variation="primary"
            appearance="filled"
            :disabled="!newFeedUrl.trim() || subscribing"
            @click="subscribeFeed"
          >
            <Plus style="width: 16px; height: 16px" />&nbsp;{{ $gettext('Add') }}
          </oc-button>
        </div>
      </div>

      <!-- Sidebar nav -->
      <nav style="flex: 1; overflow-y: auto; padding: 8px">
        <div style="display: flex; align-items: center; justify-content: space-between; padding: 4px 8px">
          <span style="font-size: 11px; font-weight: 600; text-transform: uppercase; opacity: 0.6">
            {{ $gettext('Subscriptions') }}
          </span>
          <oc-button
            variation="passive"
            appearance="raw"
            :aria-label="$gettext('New folder')"
            :title="$gettext('New folder')"
            @click="createFolder"
          >
            <FolderPlus style="width: 16px; height: 16px" />
          </oc-button>
        </div>

        <template v-for="entry in navEntries" :key="entry.key">
          <div
            style="display: flex; align-items: center; gap: 8px; padding: 5px 8px; border-radius: 6px; cursor: pointer"
            :style="{
              background: isActive(entry) ? 'rgba(0, 100, 200, 0.12)' : 'transparent',
              paddingLeft: entry.indent ? '28px' : '8px',
              fontWeight: entry.unread ? 600 : 400
            }"
            @mouseenter="hovered = entry.key"
            @mouseleave="hovered = ''"
            @click="select(entry)"
          >
            <component :is="entry.icon" style="width: 16px; height: 16px; opacity: 0.7; flex-shrink: 0" />
            <span style="flex: 1; min-width: 0; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              {{ entry.label }}
            </span>
            <span v-if="entry.error" :title="entry.error" style="flex-shrink: 0; display: inline-flex">
              <AlertTriangle style="width: 14px; height: 14px; color: #d97706" />
            </span>
            <span v-if="entry.unread" style="font-size: 12px; opacity: 0.6; flex-shrink: 0">{{ entry.unread }}</span>
            <oc-button
              v-if="entry.kind === 'feed' || entry.kind === 'folder'"
              variation="passive"
              appearance="raw"
              style="flex-shrink: 0"
              :aria-label="$gettext('Options')"
              @click.stop="openMenu = openMenu === entry.key ? '' : entry.key"
            >
              <MoreHorizontal v-if="hovered === entry.key || openMenu === entry.key" style="width: 16px; height: 16px" />
              <span v-else style="width: 16px; display: inline-block"></span>
            </oc-button>
          </div>

          <!-- Context menu -->
          <div
            v-if="openMenu === entry.key"
            style="margin: 2px 8px; border: 1px solid rgba(125, 125, 125, 0.3); border-radius: 8px; padding: 4px; display: flex; flex-direction: column; gap: 2px; background: #fff; box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15)"
          >
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Rename')" @click="renameFeed(entry)" />
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Move to folder…')" @click="moveFeed(entry)" />
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Filter articles…')" @click="openFilter(entry)" />
            <MenuBtn v-if="entry.kind === 'feed'" :label="$gettext('Retention…')" @click="openRetention(entry)" />
            <MenuBtn v-if="entry.kind === 'folder'" :label="$gettext('Rename')" @click="renameFolder(entry)" />
            <MenuBtn
              v-if="entry.kind === 'feed' || entry.kind === 'folder'"
              :label="$gettext('Mark all read')"
              @click="markEntryRead(entry)"
            />
            <MenuBtn
              v-if="entry.kind === 'feed' || entry.kind === 'folder'"
              :label="$gettext('Delete')"
              danger
              @click="deleteEntry(entry)"
            />
          </div>
        </template>
      </nav>

      <!-- OPML -->
      <div style="padding: 8px; border-top: 1px solid rgba(125, 125, 125, 0.2); display: flex; gap: 12px">
        <oc-button variation="passive" appearance="raw" style="font-size: 13px" @click="exportOpml">
          <Download style="width: 16px; height: 16px" />&nbsp;{{ $gettext('Export OPML') }}
        </oc-button>
        <oc-button variation="passive" appearance="raw" style="font-size: 13px" @click="opmlInputEl?.click()">
          <Upload style="width: 16px; height: 16px" />&nbsp;{{ $gettext('Import OPML') }}
        </oc-button>
        <oc-button variation="passive" appearance="raw" style="font-size: 13px" :aria-label="$gettext('Settings')" :title="$gettext('Settings')" @click="openSettings">
          <Settings style="width: 16px; height: 16px" />
        </oc-button>
        <input
          ref="opmlInputEl"
          type="file"
          accept=".opml,.xml,text/xml,text/x-opml"
          style="display: none"
          @change="importOpml"
        />
      </div>
    </aside>

    <!-- Lista -->
    <section style="display: flex; flex-direction: column; flex: 1; min-width: 0">
      <header
        style="display: flex; align-items: center; gap: 12px; padding: 8px 16px; border-bottom: 1px solid rgba(125, 125, 125, 0.25); flex-wrap: wrap"
      >
        <h1
          style="font-size: 16px; font-weight: 600; margin: 0; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap"
        >
          {{ currentTitle }}
        </h1>
        <div style="display: flex; align-items: center; gap: 6px; min-width: 0">
          <Search style="width: 15px; height: 15px; opacity: 0.5; flex-shrink: 0" />
          <input
            v-model="searchQuery"
            type="search"
            :placeholder="$gettext('Search articles…')"
            :aria-label="$gettext('Search articles')"
            style="width: 200px; max-width: 30vw; font-size: 13px; padding: 4px 8px; border-radius: 6px; border: 1px solid #94a3b8; background: transparent; color: inherit"
            @keydown.enter="doSearch"
          />
          <button
            v-if="searchQuery"
            style="background: none; border: 0; cursor: pointer; padding: 2px; display: inline-flex"
            :aria-label="$gettext('Clear search')"
            :title="$gettext('Clear search')"
            @click="clearSearch"
          >
            <X style="width: 14px; height: 14px" />
          </button>
        </div>
        <label style="font-size: 12px; display: flex; align-items: center; gap: 4px">
          {{ $gettext('Show') }}
          <select v-model="showAll" style="font-size: 12px; padding: 2px 4px" :aria-label="$gettext('Show')">
            <option :value="false">{{ $gettext('Unread') }}</option>
            <option :value="true">{{ $gettext('All') }}</option>
          </select>
        </label>
        <label style="font-size: 12px; display: flex; align-items: center; gap: 4px">
          {{ $gettext('Order') }}
          <select v-model="oldestFirst" style="font-size: 12px; padding: 2px 4px" :aria-label="$gettext('Order')">
            <option :value="false">{{ $gettext('Newest first') }}</option>
            <option :value="true">{{ $gettext('Oldest first') }}</option>
          </select>
        </label>
        <oc-button
          variation="passive"
          appearance="raw"
          :aria-label="$gettext('Refresh')"
          :title="$gettext('Refresh')"
          :disabled="refreshing"
          @click="refreshNow"
        >
          <RefreshCw style="width: 16px; height: 16px" :style="refreshing ? 'animation: news-spin 1s linear infinite' : ''" />
        </oc-button>
        <oc-button v-if="unreadCount > 0" variation="passive" appearance="raw" style="font-size: 13px" @click="markAllRead">
          <CheckCheck style="width: 16px; height: 16px" />&nbsp;{{ $gettext('Mark all read') }}
        </oc-button>
      </header>

      <p
        v-if="error"
        style="padding: 8px 16px; color: #b91c1c; font-size: 13px; border-bottom: 1px solid rgba(125, 125, 125, 0.2); margin: 0"
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
          style="display: flex; gap: 12px; padding: 10px 16px; border-bottom: 1px solid rgba(125, 125, 125, 0.15); cursor: pointer"
          :style="{
            background: detail?.id === item.id ? 'rgba(0, 100, 200, 0.08)' : 'transparent',
            opacity: item.unread ? 1 : 0.65
          }"
          @click="openItem(item)"
        >
          <div style="flex: 1; min-width: 0">
            <p style="margin: 0; font-size: 14px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              {{ item.title }}
            </p>
            <p style="margin: 2px 0 0; font-size: 12px; opacity: 0.6; overflow: hidden; text-overflow: ellipsis; white-space: nowrap">
              {{ feedTitle(item.feedId) }}<span v-if="item.author"> · {{ item.author }}</span> · {{ fmtDate(item.pubDate) }}
            </p>
          </div>
          <button
            style="align-self: flex-start; background: none; border: 0; cursor: pointer; padding: 2px"
            :style="{ color: item.unread ? '#2563eb' : 'rgba(0,0,0,0.35)' }"
            :aria-label="item.unread ? $gettext('Mark as read') : $gettext('Mark as unread')"
            :title="item.unread ? $gettext('Mark as read') : $gettext('Mark as unread')"
            @click.stop="toggleUnread(item)"
          >
            <component :is="item.unread ? Mail : MailOpen" style="width: 16px; height: 16px" />
          </button>
          <button
            style="align-self: flex-start; background: none; border: 0; cursor: pointer; padding: 2px"
            :style="{ color: item.starred ? '#d97706' : 'rgba(0,0,0,0.35)' }"
            :aria-label="item.starred ? $gettext('Unstar') : $gettext('Star')"
            @click.stop="toggleStar(item)"
          >
            <Star style="width: 16px; height: 16px" :fill="item.starred ? 'currentColor' : 'none'" />
          </button>
        </li>
        <li v-if="moreAvailable" style="padding: 12px; text-align: center; list-style: none">
          <oc-button variation="passive" appearance="filled" @click="loadMore">
            {{ $gettext('Load more') }}
          </oc-button>
        </li>
      </ul>
    </section>

    <!-- Detalle -->
    <section
      v-if="detail"
      style="display: flex; flex-direction: column; flex: 1; min-width: 0; border-left: 1px solid rgba(125, 125, 125, 0.25)"
    >
      <header style="display: flex; align-items: flex-start; gap: 8px; padding: 8px 16px; border-bottom: 1px solid rgba(125, 125, 125, 0.25)">
        <oc-button variation="passive" appearance="raw" :aria-label="$gettext('Back')" @click="detail = null">
          <X style="width: 16px; height: 16px" />
        </oc-button>
        <h2 style="flex: 1; font-size: 15px; font-weight: 600; margin: 0">
          <a :href="detail.url" target="_blank" rel="noopener noreferrer" style="color: inherit; text-decoration: none">
            {{ detail.title }}
          </a>
        </h2>
        <oc-button
          v-if="!detail.feedFullContent"
          variation="passive"
          appearance="raw"
          :aria-label="$gettext('Full article')"
          :title="fullBody ? $gettext('Show summary') : $gettext('Full article')"
          :disabled="fullLoading"
          @click="toggleFull"
        >
          <BookOpenText style="width: 16px; height: 16px" :style="fullBody ? 'color: #2563eb' : ''" />
        </oc-button>
        <oc-button
          variation="passive"
          appearance="raw"
          :aria-label="detail.unread ? $gettext('Mark as unread') : $gettext('Mark as read')"
          :title="detail.unread ? $gettext('Mark as read') : $gettext('Mark as unread')"
          @click="toggleUnread(detail)"
        >
          <MailOpen style="width: 16px; height: 16px" />
        </oc-button>
        <oc-button
          variation="passive"
          appearance="raw"
          :aria-label="detail.starred ? $gettext('Unstar') : $gettext('Star')"
          @click="toggleStar(detail)"
        >
          <Star style="width: 16px; height: 16px" :fill="detail.starred ? 'currentColor' : 'none'" />
        </oc-button>
        <a
          :href="detail.url"
          target="_blank"
          rel="noopener noreferrer"
          :aria-label="$gettext('Open in new tab')"
          :title="$gettext('Open in new tab')"
          style="color: inherit; display: inline-flex; padding: 4px"
        >
          <ExternalLink style="width: 16px; height: 16px" />
        </a>
      </header>
      <div style="flex: 1; overflow-y: auto; padding: 12px 16px">
        <audio
          v-if="detail.enclosureMime?.startsWith('audio/') && detail.enclosureLink"
          :src="detail.enclosureLink"
          controls
          preload="metadata"
          style="width: 100%; margin: 0 0 12px"
        />
        <video
          v-else-if="detail.enclosureMime?.startsWith('video/') && detail.enclosureLink"
          :src="detail.enclosureLink"
          controls
          preload="metadata"
          style="width: 100%; border-radius: 8px; margin: 0 0 12px; max-height: 60vh; background: #000"
        />
        <p style="font-size: 12px; opacity: 0.6; margin: 0 0 12px">
          {{ feedTitle(detail.feedId) }} · {{ fmtDate(detail.pubDate) }}
          <span v-if="fullLoading" style="opacity: 0.7"> · {{ $gettext('loading full article…') }}</span>
          <span v-else-if="fullFailed" style="color: #d97706">
            · {{ $gettext('full article not available — open the original') }}
          </span>
        </p>
        <!-- body sanitizado server-side; imágenes via proxy propio (CSP) -->
        <div
          class="oc-prose news-body"
          :class="readerClass"
          :style="{ maxWidth: readerMaxWidth, ['--news-bg' as any]: readerBg, ['--news-fg' as any]: readerFg }"
          v-html="detailBody"
        />
      </div>
    </section>

    <!-- Filtro de artículos por feed (News 28.4.0) -->
    <div
      v-if="filterOpen"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="closeFilter"
    >
      <div style="background: #fff; border-radius: 12px; padding: 20px; width: 420px; max-width: 92vw; box-shadow: 0 8px 32px rgba(0,0,0,0.25); color: #1a1a1a">
        <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">
          {{ $gettext('Filter articles') }}
        </h3>
        <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.6">
          {{ $gettext('Hide articles whose title, body or URL contains any keyword (comma-separated, case-insensitive).') }}
        </p>
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Title keywords') }}</label>
        <input
          v-model="filterForm.titleKeywords"
          type="text"
          placeholder="sponsored,ads,offer"
          style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid #94a3b8; margin-bottom: 10px"
        />
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Body keywords') }}</label>
        <input
          v-model="filterForm.bodyKeywords"
          type="text"
          placeholder="tracking,cookie"
          style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid #94a3b8; margin-bottom: 10px"
        />
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('URL keywords') }}</label>
        <input
          v-model="filterForm.urlKeywords"
          type="text"
          placeholder="utm_,/tag/"
          style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid #94a3b8; margin-bottom: 16px"
        />
        <div style="display: flex; gap: 8px; justify-content: flex-end; align-items: center">
          <oc-button v-if="filterHasFilter" variation="passive" appearance="raw" style="font-size: 13px" @click="clearFilter">
            {{ $gettext('Clear filter') }}
          </oc-button>
          <span style="flex: 1"></span>
          <oc-button variation="passive" appearance="outline" style="font-size: 13px" @click="closeFilter">
            {{ $gettext('Cancel') }}
          </oc-button>
          <oc-button variation="primary" appearance="filled" style="font-size: 13px" :disabled="filterSaving" @click="saveFilter">
            {{ $gettext('Save') }}
          </oc-button>
        </div>
      </div>
    </div>

    <!-- Retención por feed -->
    <div
      v-if="retentionOpen"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="retentionOpen = false"
    >
      <div style="background: #fff; border-radius: 12px; padding: 20px; width: 380px; max-width: 92vw; box-shadow: 0 8px 32px rgba(0,0,0,0.25); color: #1a1a1a">
        <h3 style="margin: 0 0 4px; font-size: 15px; font-weight: 600">{{ $gettext('Article retention') }}</h3>
        <p style="margin: 0 0 12px; font-size: 12px; opacity: 0.6">
          {{ $gettext('Days to keep read, non-starred articles for this feed. 0 = use the server default.') }}
        </p>
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Retention (days)') }}</label>
        <input
          v-model.number="retentionForm"
          type="number"
          min="0"
          max="3650"
          style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid #94a3b8; margin-bottom: 16px"
        />
        <div style="display: flex; gap: 8px; justify-content: flex-end">
          <oc-button variation="passive" appearance="outline" style="font-size: 13px" @click="retentionOpen = false">
            {{ $gettext('Cancel') }}
          </oc-button>
          <oc-button variation="primary" appearance="filled" style="font-size: 13px" :disabled="retentionSaving" @click="saveRetention">
            {{ $gettext('Save') }}
          </oc-button>
        </div>
      </div>
    </div>

    <!-- Ajustes de usuario -->
    <div
      v-if="settingsOpen"
      style="position: fixed; inset: 0; background: rgba(0, 0, 0, 0.4); display: flex; align-items: center; justify-content: center; z-index: 1000"
      @click.self="settingsOpen = false"
    >
      <div style="background: #fff; border-radius: 12px; padding: 20px; width: 400px; max-width: 92vw; box-shadow: 0 8px 32px rgba(0,0,0,0.25); color: #1a1a1a">
        <h3 style="margin: 0 0 16px; font-size: 15px; font-weight: 600">{{ $gettext('Settings') }}</h3>
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Reader theme') }}</label>
        <select v-model="settingsForm.theme" style="width: 100%; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid #94a3b8; margin-bottom: 12px">
          <option value="system">{{ $gettext('System') }}</option>
          <option value="light">{{ $gettext('Light') }}</option>
          <option value="dark">{{ $gettext('Dark') }}</option>
        </select>
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Reader width') }}</label>
        <select v-model="settingsForm.readerMaxWidth" style="width: 100%; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid #94a3b8; margin-bottom: 12px">
          <option value="narrow">{{ $gettext('Narrow') }}</option>
          <option value="normal">{{ $gettext('Normal') }}</option>
          <option value="wide">{{ $gettext('Wide') }}</option>
        </select>
        <label style="display: block; font-size: 12px; margin-bottom: 4px">{{ $gettext('Refresh interval (minutes)') }}</label>
        <input
          v-model="settingsForm.feedIntervalMin"
          type="number"
          min="5"
          max="1440"
          placeholder=""
          style="width: 100%; box-sizing: border-box; font-size: 13px; padding: 6px 8px; border-radius: 6px; border: 1px solid #94a3b8; margin-bottom: 16px"
        />
        <p style="margin: 0 0 16px; font-size: 11px; opacity: 0.6">{{ $gettext('Leave empty for the server default (15 min).') }}</p>
        <div style="display: flex; gap: 8px; justify-content: flex-end">
          <oc-button variation="passive" appearance="outline" style="font-size: 13px" @click="settingsOpen = false">
            {{ $gettext('Cancel') }}
          </oc-button>
          <oc-button variation="primary" appearance="filled" style="font-size: 13px" :disabled="settingsSaving" @click="saveSettings">
            {{ $gettext('Save') }}
          </oc-button>
        </div>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref, watch } from 'vue'
// useRouter SIEMPRE desde web-pkg (inyecta el router del host); el de
// vue-router no tiene inyección en el contexto de extensión (issues #002/#004)
import { useRouter } from '@opencloud-eu/web-pkg'
import { useGettext } from 'vue3-gettext'
import {
  Newspaper,
  Star,
  Mail,
  FolderOpen,
  FolderPlus,
  Plus,
  X,
  Rss,
  MoreHorizontal,
  RefreshCw,
  CheckCheck,
  Download,
  Upload,
  AlertTriangle,
  MailOpen,
  ExternalLink,
  BookOpenText,
  Filter,
  Search,
  Settings
} from 'lucide-vue-next'
import { useNewsApi, Item, Feed, Folder, Selection, FeedFilter, UserSettings } from '../api'

const { $gettext } = useGettext()
const router = useRouter()
const route = computed(() => router.currentRoute.value)
const api = useNewsApi()

// botón de menú contextual (componente local mínimo)
const MenuBtn = defineComponent({
  props: { label: { type: String, required: true }, danger: { type: Boolean, default: false } },
  emits: ['click'],
  setup(props, { emit }) {
    return () =>
      h(
        'button',
        {
          style: {
            all: 'unset',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            width: '100%',
            padding: '6px 10px',
            borderRadius: '6px',
            fontSize: '13px',
            color: props.danger ? '#b91c1c' : 'inherit'
          },
          onClick: (e: MouseEvent) => {
            e.stopPropagation()
            emit('click')
          }
        },
        props.label
      )
  }
})

const folders = ref<Folder[]>([])
const feeds = ref<Feed[]>([])
const starredCount = ref(0)
const items = ref<Item[]>([])
const detail = ref<Item | null>(null)
const fullBody = ref('')
const fullLoading = ref(false)
const fullFailed = ref(false)
const loading = ref(false)
const error = ref('')
const newFeedUrl = ref('')
const subscribing = ref(false)
const refreshing = ref(false)
const openMenu = ref('')
const hovered = ref('')
const showAll = ref(false)
const oldestFirst = ref(false)
const moreAvailable = ref(false)
const opmlInputEl = ref<HTMLInputElement | null>(null)
const searchQuery = ref('')

const searchActive = computed(() => searchQuery.value.trim() !== '')

const userTheme = ref('system')
const userWidth = ref('normal')

const readerClass = computed(() => `news-theme-${userTheme.value}`)
const readerMaxWidth = computed(() => ({ narrow: '52ch', normal: '72ch', wide: '100%' })[userWidth.value] ?? '72ch')
const readerBg = computed(() =>
  userTheme.value === 'dark' ? '#161616' : userTheme.value === 'light' ? '#ffffff' : ''
)
const readerFg = computed(() =>
  userTheme.value === 'dark' ? '#e5e5e5' : userTheme.value === 'light' ? '#1a1a1a' : ''
)

async function loadSettings() {
  try {
    const s = await api.mySettings()
    userTheme.value = s.theme || 'system'
    userWidth.value = s.readerMaxWidth || 'normal'
  } catch {
    /* defaults */
  }
}

function doSearch() {
  if (searchQuery.value.trim()) {
    detail.value = null
    loadItems()
  }
}

function clearSearch() {
  searchQuery.value = ''
  detail.value = null
  loadItems()
}
const filterOpen = ref(false)
const filterSaving = ref(false)
const filterFeedId = ref<number | null>(null)
const filterForm = ref<Omit<FeedFilter, 'feedId'>>({ titleKeywords: '', bodyKeywords: '', urlKeywords: '' })

const filterHasFilter = computed(
  () =>
    filterForm.value.titleKeywords.trim() !== '' ||
    filterForm.value.bodyKeywords.trim() !== '' ||
    filterForm.value.urlKeywords.trim() !== ''
)

const retentionOpen = ref(false)
const retentionSaving = ref(false)
const retentionFeedId = ref<number | null>(null)
const retentionForm = ref(0)

const settingsOpen = ref(false)
const settingsSaving = ref(false)
const settingsForm = ref<UserSettings>({ theme: 'system', readerMaxWidth: 'normal', feedIntervalMin: '' })

const BATCH = 50

type NavEntry = {
  key: string
  kind: 'all' | 'starred' | 'folder' | 'feed'
  label: string
  icon: typeof Rss
  sel: Selection
  unread: number
  error?: string
  indent?: boolean
}

const selection = computed<Selection>(() => {
  const r = route.value
  if (r.name === 'news-feed') return { kind: 'feed', id: Number(r.params.feedId) }
  if (r.name === 'news-folder') return { kind: 'folder', id: Number(r.params.folderId) }
  if (r.name === 'news-starred') return { kind: 'starred' }
  return { kind: 'all' }
})

const navEntries = computed<NavEntry[]>(() => {
  const byFolder = new Map<number, Feed[]>()
  for (const f of feeds.value) {
    const k = f.folderId ?? 0
    byFolder.set(k, [...(byFolder.get(k) ?? []), f])
  }
  const entries: NavEntry[] = [
    { key: 'all', kind: 'all', label: $gettext('All articles'), icon: Newspaper, sel: { kind: 'all' }, unread: totalUnread.value },
    { key: 'starred', kind: 'starred', label: $gettext('Starred'), icon: Star, sel: { kind: 'starred' }, unread: starredCount.value }
  ]
  for (const folder of folders.value) {
    entries.push({
      key: `folder-${folder.id}`,
      kind: 'folder',
      label: folder.name,
      icon: FolderOpen,
      sel: { kind: 'folder', id: folder.id },
      unread: (byFolder.get(folder.id) ?? []).reduce((a, f) => a + f.unreadCount, 0)
    })
    for (const f of byFolder.get(folder.id) ?? []) {
      entries.push({
        key: `feed-${f.id}`,
        kind: 'feed',
        label: f.title,
        icon: Rss,
        sel: { kind: 'feed', id: f.id },
        unread: f.unreadCount,
        error: f.updateErrorCount > 0 ? f.lastUpdateError ?? `${f.updateErrorCount} errors` : undefined,
        indent: true
      })
    }
  }
  for (const f of byFolder.get(0) ?? []) {
    entries.push({
      key: `feed-${f.id}`,
      kind: 'feed',
      label: f.title,
      icon: Rss,
      sel: { kind: 'feed', id: f.id },
      unread: f.unreadCount,
      error: f.updateErrorCount > 0 ? f.lastUpdateError ?? `${f.updateErrorCount} errors` : undefined,
      indent: true
    })
  }
  return entries
})

async function toggleFull() {
  if (!detail.value) return
  if (fullBody.value) {
    fullBody.value = '' // volver al resumen del feed
    return
  }
  fullLoading.value = true
  fullFailed.value = false
  try {
    const res = await api.itemFull(detail.value.id)
    fullBody.value = res.body
  } catch {
    fullFailed.value = true
  } finally {
    fullLoading.value = false
  }
}

// cuerpo del detalle: enlaces del artículo a pestaña nueva (dentro del host
// navegarían en el propio frame); las imágenes ya llegan vía proxy firmado
const detailBody = computed(() => {
  const base = fullBody.value || detail.value?.body || ""
  return base.replaceAll('<a ', '<a target="_blank" rel="noopener noreferrer" ')
})

const totalUnread = computed(() => feeds.value.reduce((a, f) => a + f.unreadCount, 0))
const unreadCount = computed(() =>
  selection.value.kind === 'starred' ? starredCount.value : items.value.filter((i) => i.unread).length
)
const currentTitle = computed(() => {
  const e = navEntries.value.find((e) => JSON.stringify(e.sel) === JSON.stringify(selection.value))
  return e?.label ?? $gettext('News')
})

function isActive(entry: NavEntry) {
  return JSON.stringify(entry.sel) === JSON.stringify(selection.value)
}

function select(entry: NavEntry) {
  detail.value = null
  switch (entry.sel.kind) {
    case 'all': router.push('/news/'); break
    case 'starred': router.push('/news/starred'); break
    case 'feed': router.push(`/news/feed/${entry.sel.id}`); break
    case 'folder': router.push(`/news/folder/${entry.sel.id}`); break
  }
}

function feedTitle(feedId: number) {
  return feeds.value.find((f) => f.id === feedId)?.title ?? ''
}

function fmtDate(epoch: number) {
  return new Date(epoch * 1000).toLocaleDateString(undefined, {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit'
  })
}

async function loadSidebar() {
  const [f, fd] = await Promise.all([api.feeds(), api.folders()])
  feeds.value = f.feeds
  starredCount.value = f.starredCount
  folders.value = fd.folders
}

async function loadItems(append = false) {
  loading.value = !append
  error.value = ''
  try {
    const getRead = selection.value.kind === 'starred' ? true : showAll.value
    if (searchActive.value) {
      const res = await api.search(searchQuery.value.trim(), { getRead, batchSize: 100, oldestFirst: oldestFirst.value })
      items.value = res.items
      moreAvailable.value = false
      return
    }
    const offset = append ? (oldestFirst.value ? Math.max(...items.value.map((i) => i.id)) : Math.min(...items.value.map((i) => i.id))) : undefined
    const res = await api.items(selection.value, { getRead, batchSize: BATCH, oldestFirst: oldestFirst.value, offset })
    items.value = append ? [...items.value, ...res.items] : res.items
    moreAvailable.value = res.items.length >= BATCH
  } catch (e) {
    error.value = $gettext('Could not load items: ') + errText(e)
  } finally {
    loading.value = false
  }
}

async function refreshCounts() {
  try {
    const f = await api.feeds()
    feeds.value = f.feeds
    starredCount.value = f.starredCount
  } catch {
    /* el sidebar se refrescará en el siguiente load */
  }
}

async function openItem(item: Item) {
  detail.value = item
  fullBody.value = ''
  fullFailed.value = false
  if (item.unread) {
    item.unread = false
    await api.markRead(item.id)
    refreshCounts()
  }
}

async function toggleUnread(item: Item) {
  item.unread = !item.unread
  if (item.unread) {
    await api.markUnread(item.id)
  } else {
    await api.markRead(item.id)
  }
  refreshCounts()
}

async function toggleStar(item: Item) {
  item.starred = !item.starred
  if (item.starred) {
    await api.star(item.id)
  } else {
    await api.unstar(item.id)
  }
  refreshCounts()
}

async function refreshNow() {
  refreshing.value = true
  error.value = ''
  try {
    await api.refresh()
    await loadSidebar()
    await loadItems()
  } catch (e) {
    error.value = $gettext('Refresh failed: ') + errText(e)
  } finally {
    refreshing.value = false
  }
}

async function loadMore() {
  await loadItems(true)
}

async function markAllRead() {
  const newest = items.value.reduce((a, i) => Math.max(a, i.id), 0)
  await api.markAllRead(newest)
  await loadSidebar()
  await loadItems()
}

async function subscribeFeed() {
  const url = newFeedUrl.value.trim()
  if (!url) return
  subscribing.value = true
  error.value = ''
  try {
    await api.addFeed(url, null)
    newFeedUrl.value = ''
    await loadSidebar()
    await loadItems()
  } catch (e: unknown) {
    error.value = extractErrorMessage(e, $gettext('Could not subscribe to the feed'))
  } finally {
    subscribing.value = false
  }
}

async function createFolder() {
  const name = window.prompt($gettext('Folder name'))
  if (!name?.trim()) return
  try {
    await api.addFolder(name.trim())
    await loadSidebar()
  } catch (e) {
    error.value = $gettext('Could not create folder: ') + errText(e)
  }
}

function entryFeed(entry: NavEntry): Feed | undefined {
  return feeds.value.find((f) => `feed-${f.id}` === entry.key)
}
function entryFolder(entry: NavEntry): Folder | undefined {
  return folders.value.find((f) => `folder-${f.id}` === entry.key)
}

async function renameFeed(entry: NavEntry) {
  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  const title = window.prompt($gettext('New title'), f.title)
  if (!title?.trim()) return
  await api.renameFeed(f.id, title.trim())
  await loadSidebar()
}

async function moveFeed(entry: NavEntry) {
  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  const names = folders.value.map((fo) => fo.name).join(', ')
  const raw = window.prompt($gettext('Move to folder (name, empty = none)') + (names ? ` [${names}]` : ''), '')
  if (raw === null) return
  const target = folders.value.find((fo) => fo.name.toLowerCase() === raw.trim().toLowerCase())
  await api.moveFeed(f.id, target ? target.id : null)
  await loadSidebar()
}

async function renameFolder(entry: NavEntry) {
  openMenu.value = ''
  const fo = entryFolder(entry)
  if (!fo) return
  const name = window.prompt($gettext('New name'), fo.name)
  if (!name?.trim()) return
  await api.renameFolder(fo.id, name.trim())
  await loadSidebar()
}

async function markEntryRead(entry: NavEntry) {
  openMenu.value = ''
  const newest = await currentNewest()
  if (entry.kind === 'feed') await api.markFeedRead((entry.sel as { id: number }).id, newest)
  if (entry.kind === 'folder') await api.markFolderRead((entry.sel as { id: number }).id, newest)
  await loadSidebar()
  await loadItems()
}

async function currentNewest(): Promise<number> {
  const f = await api.feeds()
  return f.newestItemId ?? 0
}

async function deleteEntry(entry: NavEntry) {
  openMenu.value = ''
  const isFeed = entry.kind === 'feed'
  const label = isFeed ? entryFeed(entry)?.title : entryFolder(entry)?.name
  if (!window.confirm($gettext('Delete') + ` "${label}"?` + (isFeed ? '' : ' ' + $gettext('(feeds inside will be deleted)')))) return
  try {
    if (isFeed) await api.deleteFeed((entry.sel as { id: number }).id)
    else await api.deleteFolder((entry.sel as { id: number }).id)
    if (isActive(entry)) router.push('/news/')
    await loadSidebar()
    await loadItems()
  } catch (e) {
    error.value = $gettext('Delete failed: ') + errText(e)
  }
}

async function openFilter(entry: NavEntry) {
  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  filterFeedId.value = f.id
  filterForm.value = { titleKeywords: '', bodyKeywords: '', urlKeywords: '' }
  try {
    const res = await api.getFilter(f.id)
    filterForm.value = {
      titleKeywords: res.filter.titleKeywords ?? '',
      bodyKeywords: res.filter.bodyKeywords ?? '',
      urlKeywords: res.filter.urlKeywords ?? ''
    }
  } catch {
    /* sin filtro previo: campos vacíos */
  }
  filterOpen.value = true
}

function closeFilter() {
  filterOpen.value = false
  filterFeedId.value = null
}

async function saveFilter() {
  if (filterFeedId.value == null) return
  filterSaving.value = true
  try {
    await api.setFilter(filterFeedId.value, {
      titleKeywords: filterForm.value.titleKeywords.trim(),
      bodyKeywords: filterForm.value.bodyKeywords.trim(),
      urlKeywords: filterForm.value.urlKeywords.trim()
    })
    closeFilter()
  } catch (e) {
    error.value = $gettext('Could not save filter: ') + errText(e)
  } finally {
    filterSaving.value = false
  }
}

async function clearFilter() {
  if (filterFeedId.value == null) return
  filterSaving.value = true
  try {
    await api.deleteFilter(filterFeedId.value)
    closeFilter()
  } catch (e) {
    error.value = $gettext('Could not clear filter: ') + errText(e)
  } finally {
    filterSaving.value = false
  }
}

async function openRetention(entry: NavEntry) {
  openMenu.value = ''
  const f = entryFeed(entry)
  if (!f) return
  retentionFeedId.value = f.id
  retentionForm.value = 0
  try {
    const res = await api.getRetention(f.id)
    retentionForm.value = res.retentionDays ?? 0
  } catch {
    /* usar 0 */
  }
  retentionOpen.value = true
}

async function saveRetention() {
  if (retentionFeedId.value == null) return
  retentionSaving.value = true
  try {
    const days = Math.max(0, Math.min(3650, retentionForm.value || 0))
    await api.setRetention(retentionFeedId.value, days)
    retentionOpen.value = false
  } catch (e) {
    error.value = $gettext('Could not save retention: ') + errText(e)
  } finally {
    retentionSaving.value = false
  }
}

async function openSettings() {
  settingsOpen.value = true
  try {
    const s = await api.mySettings()
    settingsForm.value = { theme: s.theme || 'system', readerMaxWidth: s.readerMaxWidth || 'normal', feedIntervalMin: s.feedIntervalMin ?? '' }
  } catch {
    /* usar defaults */
  }
}

async function saveSettings() {
  settingsSaving.value = true
  try {
    const updated = await api.updateSettings({
      theme: settingsForm.value.theme,
      readerMaxWidth: settingsForm.value.readerMaxWidth,
      feedIntervalMin: settingsForm.value.feedIntervalMin.trim()
    })
    userTheme.value = updated.theme || 'system'
    userWidth.value = updated.readerMaxWidth || 'normal'
    settingsOpen.value = false
  } catch (e) {
    error.value = $gettext('Could not save settings: ') + errText(e)
  } finally {
    settingsSaving.value = false
  }
}

async function exportOpml() {
  try {
    const res: { data: string } = await api.exportOpml()
    const blob = new Blob([res.data], { type: 'text/x-opml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'subscriptions.opml'
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    error.value = $gettext('Export failed: ') + errText(e)
  }
}

async function importOpml(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  try {
    const res: { data: { imported: number; skipped: number } } = await api.importOpml(file)
    error.value = ''
    await loadSidebar()
    await refreshNow()
    window.alert($gettext('Imported') + `: ${res.data.imported}, ` + $gettext('skipped') + `: ${res.data.skipped}`)
  } catch (e) {
    error.value = $gettext('Import failed: ') + errText(e)
  } finally {
    input.value = ''
  }
}

function errText(e: unknown): string {
  const anyE = e as { response?: { status?: number; data?: unknown }; message?: string }
  if (anyE?.response?.status) return `HTTP ${anyE.response.status}`
  return anyE?.message ?? String(e)
}

function extractErrorMessage(e: unknown, fallback: string): string {
  const anyE = e as { response?: { status?: number } }
  if (anyE?.response?.status === 409) return $gettext('Already subscribed')
  if (anyE?.response?.status === 422) return $gettext('Feed not readable')
  return fallback
}

watch(selection, () => loadItems())
watch(showAll, () => loadItems())
watch(oldestFirst, () => loadItems())
watch(() => route.value.fullPath, () => (detail.value = null))

onMounted(async () => {
  try {
    await loadSettings()
    await loadSidebar()
  } catch (e) {
    error.value = $gettext('Could not load folders/feeds: ') + errText(e)
  }
  await loadItems()
})
</script>

<style>
@keyframes news-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
.news-body p { margin: 0 0 0.9em; line-height: 1.65; }
.news-body h1, .news-body h2, .news-body h3, .news-body h4 {
  margin: 1.2em 0 0.5em;
  line-height: 1.3;
  font-weight: 600;
}
.news-body h2 { font-size: 1.3em; }
.news-body h3 { font-size: 1.15em; }
.news-body ul, .news-body ol { margin: 0 0 1em; padding-left: 1.4em; line-height: 1.6; }
.news-body li { margin-bottom: 0.3em; }
.news-body img {
  max-width: 100%;
  height: auto;
  border-radius: 8px;
  margin: 0.4em 0 1em;
}
.news-body a { color: #2563eb; text-decoration: underline; }
.news-body a:hover { opacity: 0.8; }
.news-theme-dark {
  color: var(--news-fg, #e5e5e5);
  background: var(--news-bg, #161616);
  border-radius: 8px;
  padding: 12px;
}
.news-theme-light {
  color: var(--news-fg, #1a1a1a);
  background: var(--news-bg, #ffffff);
  border-radius: 8px;
  padding: 12px;
}
.news-body blockquote {
  margin: 0 0 1em;
  padding: 0.4em 1em;
  border-left: 3px solid rgba(125, 125, 125, 0.4);
  opacity: 0.9;
}
.news-body pre {
  background: rgba(125, 125, 125, 0.12);
  border-radius: 8px;
  padding: 0.8em 1em;
  overflow-x: auto;
  font-size: 0.9em;
}
.news-body code { background: rgba(125, 125, 125, 0.12); border-radius: 4px; padding: 0.1em 0.35em; }
.news-body pre code { background: none; padding: 0; }
.news-body figure { margin: 0 0 1em; }
.news-body figcaption { font-size: 0.85em; opacity: 0.65; margin-top: 0.3em; }
.news-body table { border-collapse: collapse; margin-bottom: 1em; }
.news-body th, .news-body td { border: 1px solid rgba(125, 125, 125, 0.35); padding: 0.4em 0.7em; }
</style>
