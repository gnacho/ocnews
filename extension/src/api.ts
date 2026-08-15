import { useClientService } from '@opencloud-eu/web-pkg'

// News REST API v1.3 client (backend: ocnews service under the same origin)
const BASE = '/index.php/apps/news/api/v1-3'

export interface Folder {
  id: number
  name: string
}

export interface Feed {
  id: number
  url: string
  title: string
  faviconLink: string
  added: number
  folderId: number | null
  unreadCount: number
  ordering: number
  link: string
  pinned: boolean
  updateErrorCount: number
  lastUpdateError: string | null
}

export interface Item {
  id: number
  guid: string
  guidHash: string
  url: string
  title: string
  author: string | null
  pubDate: number
  body: string
  enclosureMime: string | null
  enclosureLink: string | null
  mediaThumbnail: string | null
  feedId: number
  unread: boolean
  starred: boolean
  lastModified: number
}

export interface FeedsResponse {
  feeds: Feed[]
  starredCount: number
  newestItemId?: number
}

// type: 0 feed, 1 folder, 2 starred, 3 all
export type Selection =
  | { kind: 'all' }
  | { kind: 'starred' }
  | { kind: 'feed'; id: number }
  | { kind: 'folder'; id: number }

export function useNewsApi() {
  const client = useClientService()

  async function get(path: string, params: Record<string, string> = {}) {
    const usp = new URLSearchParams(params)
    const qs = usp.toString()
    const { data } = await client.httpAuthenticated.get(`${BASE}${path}${qs ? `?${qs}` : ''}`)
    return data
  }

  async function post(path: string, body?: unknown) {
    await client.httpAuthenticated.post(`${BASE}${path}`, body ?? {})
  }

  return {
    folders: (): Promise<{ folders: Folder[] }> => get('/folders'),
    feeds: (): Promise<FeedsResponse> => get('/feeds'),
    items: (sel: Selection, opts: { batchSize?: number; offset?: number; getRead?: boolean }) => {
      const params: Record<string, string> = {
        type: String(
          sel.kind === 'feed' ? 0 : sel.kind === 'folder' ? 1 : sel.kind === 'starred' ? 2 : 3
        ),
        getRead: String(opts.getRead ?? false),
        batchSize: String(opts.batchSize ?? -1),
        oldestFirst: 'false'
      }
      if (sel.kind === 'feed' || sel.kind === 'folder') params.id = String(sel.id)
      if (opts.offset) params.offset = String(opts.offset)
      return get('/items', params) as Promise<{ items: Item[] }>
    },
    markRead: (id: number) => post(`/items/${id}/read`),
    markUnread: (id: number) => post(`/items/${id}/unread`),
    star: (id: number) => post(`/items/${id}/star`),
    unstar: (id: number) => post(`/items/${id}/unstar`),
    markAllRead: (newestItemId: number) => post('/items/read', { newestItemId }),
    addFeed: (url: string, folderId: number | null) =>
      client.httpAuthenticated.post(`${BASE}/feeds`, { url, folderId }),
    addFolder: (name: string) => post('/folders', { name }),
    deleteFeed: (feedId: number) => client.httpAuthenticated.delete(`${BASE}/feeds/${feedId}`)
  }
}
