<template>
  <main class="ext:flex ext:h-full ext:bg-surface-normal">
    <!-- Sidebar -->
    <aside
      class="ext:flex ext:flex-col ext:w-64 ext:shrink-0 ext:border-r ext:border-surface-normal-secondary ext:bg-surface-normal-subtle"
    >
      <div class="ext:flex ext:items-center ext:gap-2 ext:px-4 ext:py-3">
        <input
          v-model="newFeedUrl"
          type="url"
          :placeholder="$gettext('Feed URL…')"
          :aria-label="$gettext('Feed URL')"
          class="ext:flex-1 ext:text-sm ext:px-2 ext:py-1 ext:rounded ext:border ext:border-surface-normal-emphasis ext:bg-surface-normal ext:min-w-0"
          @keydown.enter="subscribeFeed"
        />
        <oc-button
          variation="primary"
          appearance="raw"
          :aria-label="$gettext('Subscribe')"
          :disabled="!newFeedUrl.trim() || subscribing"
          @click="subscribeFeed"
        >
          <Plus class="ext:w-4 ext:h-4" />
        </oc-button>
      </div>

      <nav class="ext:flex-1 ext:overflow-y-auto ext:py-2 ext:px-2 ext:space-y-0.5">
        <button
          v-for="entry in navEntries"
          :key="entry.key"
          class="ext:w-full ext:flex ext:items-center ext:gap-2 ext:px-2 ext:py-1.5 ext:rounded ext:text-sm ext:text-left"
          :class="isActive(entry) ? 'ext:bg-surface-highlight ext:text-infobox-brand' : 'hover:ext:bg-surface-highlight'"
          @click="select(entry)"
        >
          <component :is="entry.icon" class="ext:w-4 ext:h-4 ext:opacity-70" />
          <span class="ext:flex-1 ext:truncate">{{ entry.label }}</span>
          <span v-if="entry.unread" class="ext:text-xs ext:opacity-60">{{ entry.unread }}</span>
        </button>
      </nav>
    </aside>

    <!-- Lista de items -->
    <section class="ext:flex ext:flex-col ext:flex-1 ext:min-w-0">
      <header
        class="ext:flex ext:items-center ext:justify-between ext:px-4 ext:py-2 ext:border-b ext:border-surface-normal-secondary"
      >
        <h1 class="ext:text-lg ext:font-semibold ext:truncate">{{ currentTitle }}</h1>
        <oc-button v-if="unreadCount > 0" variation="passive" appearance="raw" @click="markAllRead">
          {{ $gettext('Mark all read') }}
        </oc-button>
      </header>

      <p
        v-if="error"
        class="ext:px-4 ext:py-2 ext:text-status-danger ext:text-sm ext:border-b ext:border-surface-normal-secondary"
      >
        {{ error }}
      </p>
      <p v-if="loading" class="ext:px-4 ext:py-8 ext:text-muted ext:text-sm ext:animate-pulse">
        {{ $gettext('Loading…') }}
      </p>
      <p v-else-if="items.length === 0" class="ext:px-4 ext:py-8 ext:text-muted ext:text-sm">
        {{ $gettext('No articles') }}
      </p>

      <ul v-else class="ext:flex-1 ext:overflow-y-auto" :class="detail ? 'ext:hidden sm:ext:block' : ''">
        <li
          v-for="item in items"
          :key="item.id"
          class="ext:flex ext:gap-3 ext:px-4 ext:py-3 ext:border-b ext:border-surface-normal-secondary ext:cursor-pointer"
          :class="[
            detail?.id === item.id ? 'ext:bg-surface-highlight' : 'hover:ext:bg-surface-highlight',
            item.unread ? 'ext-font-medium' : 'ext:opacity-70'
          ]"
          @click="openItem(item)"
        >
          <div class="ext:flex-1 ext:min-w-0">
            <p class="ext:truncate">{{ item.title }}</p>
            <p class="ext:text-xs ext:opacity-60 ext:truncate">
              {{ feedTitle(item.feedId) }}<span v-if="item.author"> · {{ item.author }}</span>
              · {{ fmtDate(item.pubDate) }}
            </p>
          </div>
          <button
            class="ext:self-start ext:p-1 ext:rounded"
            :class="item.starred ? 'ext:text-amber-500' : 'ext:opacity-40 hover:ext:opacity-90'"
            :aria-label="item.starred ? $gettext('Unstar') : $gettext('Star')"
            @click.stop="toggleStar(item)"
          >
            <Star class="ext:w-4 ext:h-4" :fill="item.starred ? 'currentColor' : 'none'" />
          </button>
        </li>
      </ul>
    </section>

    <!-- Detalle -->
    <section
      v-if="detail"
      class="ext:flex ext:flex-col ext:flex-1 ext:min-w-0 ext:border-l ext:border-surface-normal-secondary"
    >
      <header
        class="ext:flex ext:items-start ext:gap-2 ext:px-4 ext:py-2 ext:border-b ext:border-surface-normal-secondary"
      >
        <oc-button variation="passive" appearance="raw" :aria-label="$gettext('Back')" @click="detail = null">
          <X class="ext:w-4 ext:h-4" />
        </oc-button>
        <h2 class="ext:flex-1 ext:font-semibold ext:leading-tight">
          <a :href="detail.url" target="_blank" rel="noopener noreferrer" class="hover:ext:underline">
            {{ detail.title }}
          </a>
        </h2>
        <oc-button
          variation="passive"
          appearance="raw"
          :aria-label="detail.starred ? $gettext('Unstar') : $gettext('Star')"
          @click="toggleStar(detail)"
        >
          <Star class="ext:w-4 ext:h-4" :fill="detail.starred ? 'currentColor' : 'none'" />
        </oc-button>
      </header>
      <div class="ext:flex-1 ext:overflow-y-auto ext:px-4 ext:py-3">
        <p class="ext:text-xs ext:opacity-60 ext:mb-3">
          {{ feedTitle(detail.feedId) }} · {{ fmtDate(detail.pubDate) }}
        </p>
        <!-- body sanitizado server-side -->
        <div class="oc-prose ext:max-w-prose" v-html="detail.body" />
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
// useRouter SIEMPRE desde web-pkg (inyecta el router del host); el de
// vue-router no tiene inyección en el contexto de extensión
import { useRouter } from '@opencloud-eu/web-pkg'
import { useGettext } from 'vue3-gettext'
import { Newspaper, Rss, Star, FolderOpen, Plus, X } from 'lucide-vue-next'
import { useNewsApi, Item, Feed, Folder, Selection } from '../api'

const { $gettext } = useGettext()
const router = useRouter()
// sin useRoute(): en extensiones esa inyección no existe (las apps oficiales
// leen router.currentRoute / useRouteQuery); route.name explotaba el setup
const route = computed(() => router.currentRoute.value)
const api = useNewsApi()

const folders = ref<Folder[]>([])
const feeds = ref<Feed[]>([])
const starredCount = ref(0)
const items = ref<Item[]>([])
const detail = ref<Item | null>(null)
const loading = ref(false)
const error = ref('')
const newFeedUrl = ref('')
const subscribing = ref(false)

type NavEntry = {
  key: string
  label: string
  icon: typeof Rss
  sel: Selection
  unread: number
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
    {
      key: 'all',
      label: $gettext('All articles'),
      icon: Newspaper,
      sel: { kind: 'all' },
      unread: totalUnread.value
    },
    {
      key: 'starred',
      label: $gettext('Starred'),
      icon: Star,
      sel: { kind: 'starred' },
      unread: starredCount.value
    }
  ]
  for (const folder of folders.value) {
    entries.push({
      key: `folder-${folder.id}`,
      label: folder.name,
      icon: FolderOpen,
      sel: { kind: 'folder', id: folder.id },
      unread: (byFolder.get(folder.id) ?? []).reduce((a, f) => a + f.unreadCount, 0)
    })
    for (const f of byFolder.get(folder.id) ?? []) {
      entries.push({
        key: `feed-${f.id}`,
        label: f.title,
        icon: Rss,
        sel: { kind: 'feed', id: f.id },
        unread: f.unreadCount,
        indent: true
      })
    }
  }
  for (const f of byFolder.get(0) ?? []) {
    entries.push({
      key: `feed-${f.id}`,
      label: f.title,
      icon: Rss,
      sel: { kind: 'feed', id: f.id },
      unread: f.unreadCount,
      indent: true
    })
  }
  return entries
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
    case 'all':
      router.push('/news/')
      break
    case 'starred':
      router.push('/news/starred')
      break
    case 'feed':
      router.push(`/news/feed/${entry.sel.id}`)
      break
    case 'folder':
      router.push(`/news/folder/${entry.sel.id}`)
      break
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

async function loadItems() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.items(selection.value, { getRead: selection.value.kind === 'starred' })
    items.value = res.items
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

async function openItem(item: Item) {
  detail.value = item
  if (item.unread) {
    item.unread = false
    await api.markRead(item.id)
    refreshCounts()
  }
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

async function refreshCounts() {
  try {
    const f = await api.feeds()
    feeds.value = f.feeds
    starredCount.value = f.starredCount
  } catch {
    /* el sidebar se refrescará en el siguiente load */
  }
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
  } catch (e: unknown) {
    error.value = extractErrorMessage(e, $gettext('Could not subscribe to the feed'))
  } finally {
    subscribing.value = false
  }
}

function extractErrorMessage(e: unknown, fallback: string): string {
  const anyE = e as { response?: { status?: number } }
  if (anyE?.response?.status === 409) return $gettext('Already subscribed')
  if (anyE?.response?.status === 422) return $gettext('Feed not readable')
  return fallback
}

watch(selection, loadItems)
watch(() => route.value.fullPath, () => (detail.value = null))

onMounted(async () => {
  try {
    await loadSidebar()
  } catch (e) {
    console.error('[news] loadSidebar failed', e)
    error.value = $gettext('Could not load folders/feeds: ') + errText(e)
  }
  try {
    await loadItems()
  } catch (e) {
    console.error('[news] loadItems failed', e)
    error.value = (error.value ? error.value + ' · ' : '') + $gettext('Could not load items: ') + errText(e)
  }
})

function errText(e: unknown): string {
  const anyE = e as { response?: { status?: number; data?: unknown }; message?: string }
  if (anyE?.response?.status) {
    return `HTTP ${anyE.response.status}`
  }
  return anyE?.message ?? String(e)
}
</script>
