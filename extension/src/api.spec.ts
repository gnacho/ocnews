import { beforeEach, describe, expect, it, vi } from 'vitest'

const { httpAuthenticated } = vi.hoisted(() => ({
  httpAuthenticated: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

vi.mock('@opencloud-eu/web-pkg', () => ({
  useClientService: () => ({ httpAuthenticated })
}))

import { useNewsApi } from './api'

const BASE = '/index.php/apps/news/api/v1-3'

function lastGetUrl() {
  return httpAuthenticated.get.mock.calls.at(-1)?.[0] as string
}

function queryOf(url: string) {
  return Object.fromEntries(new URL(url, 'http://host').searchParams.entries())
}

beforeEach(() => {
  vi.clearAllMocks()
  httpAuthenticated.get.mockResolvedValue({ data: {} })
  httpAuthenticated.post.mockResolvedValue({ data: {} })
  httpAuthenticated.put.mockResolvedValue({ data: {} })
  httpAuthenticated.delete.mockResolvedValue({ data: {} })
})

describe('every call goes through the news API prefix', () => {
  it('prefixes reads, writes and deletes', async () => {
    const api = useNewsApi()
    await api.folders()
    expect(lastGetUrl()).toBe(`${BASE}/folders`)

    await api.markRead(7)
    expect(httpAuthenticated.post).toHaveBeenLastCalledWith(`${BASE}/items/7/read`, {})

    await api.deleteFeed(3)
    expect(httpAuthenticated.delete).toHaveBeenLastCalledWith(`${BASE}/feeds/3`)
  })
})

describe('items()', () => {
  it('maps every selection kind to the API type number', async () => {
    const api = useNewsApi()
    const cases = [
      { sel: { kind: 'all' as const }, type: '3' },
      { sel: { kind: 'starred' as const }, type: '2' },
      { sel: { kind: 'podcasts' as const }, type: '3' },
      { sel: { kind: 'feed' as const, id: 4 }, type: '0' },
      { sel: { kind: 'folder' as const, id: 9 }, type: '1' }
    ]
    for (const { sel, type } of cases) {
      await api.items(sel, {})
      expect(queryOf(lastGetUrl()).type).toBe(type)
    }
  })

  it('sends the id only for feeds and folders', async () => {
    const api = useNewsApi()
    await api.items({ kind: 'feed', id: 4 }, {})
    expect(queryOf(lastGetUrl()).id).toBe('4')

    await api.items({ kind: 'folder', id: 9 }, {})
    expect(queryOf(lastGetUrl()).id).toBe('9')

    await api.items({ kind: 'all' }, {})
    expect(queryOf(lastGetUrl()).id).toBeUndefined()
  })

  it('defaults to unread, no limit and newest first', async () => {
    const api = useNewsApi()
    await api.items({ kind: 'all' }, {})
    expect(queryOf(lastGetUrl())).toEqual({
      type: '3',
      getRead: 'false',
      batchSize: '-1',
      oldestFirst: 'false'
    })
  })

  it('passes the offset and the explicit options', async () => {
    const api = useNewsApi()
    await api.items({ kind: 'starred' }, { getRead: true, batchSize: 50, oldestFirst: true, offset: 100 })
    expect(queryOf(lastGetUrl())).toEqual({
      type: '2',
      getRead: 'true',
      batchSize: '50',
      oldestFirst: 'true',
      offset: '100'
    })
  })

  it('returns the body it received', async () => {
    const api = useNewsApi()
    httpAuthenticated.get.mockResolvedValue({ data: { items: [{ id: 1 }] } })
    await expect(api.items({ kind: 'all' }, {})).resolves.toEqual({ items: [{ id: 1 }] })
  })
})

describe('search()', () => {
  it('defaults to read and unread items with a batch of 100', async () => {
    const api = useNewsApi()
    await api.search('cafe')
    expect(lastGetUrl()).toContain(`${BASE}/items/search`)
    expect(queryOf(lastGetUrl())).toEqual({
      query: 'cafe',
      getRead: 'true',
      batchSize: '100',
      oldestFirst: 'false'
    })
  })

  it('keeps an empty query and overrides the options', async () => {
    const api = useNewsApi()
    await api.search('', { getRead: false, batchSize: 5, oldestFirst: true })
    expect(queryOf(lastGetUrl())).toEqual({
      query: '',
      getRead: 'false',
      batchSize: '5',
      oldestFirst: 'true'
    })
  })
})

describe('writes carry the payload the API expects', () => {
  it('sends the newest item when marking a whole list as read', async () => {
    const api = useNewsApi()
    await api.markAllRead(42)
    expect(httpAuthenticated.post).toHaveBeenLastCalledWith(`${BASE}/items/read`, { newestItemId: 42 })

    await api.markFeedRead(2, 41)
    expect(httpAuthenticated.post).toHaveBeenLastCalledWith(`${BASE}/feeds/2/read`, { newestItemId: 41 })

    await api.markFolderRead(8, 40)
    expect(httpAuthenticated.post).toHaveBeenLastCalledWith(`${BASE}/folders/8/read`, { newestItemId: 40 })
  })

  it('adds a feed with its folder and credentials', async () => {
    const api = useNewsApi()
    await api.addFeed('https://example.test/rss', 5, 'user', 'secret')
    expect(httpAuthenticated.post).toHaveBeenLastCalledWith(`${BASE}/feeds`, {
      url: 'https://example.test/rss',
      folderId: 5,
      username: 'user',
      password: 'secret'
    })
  })

  it('moves a feed to the root with a null folder', async () => {
    const api = useNewsApi()
    await api.moveFeed(6, null)
    expect(httpAuthenticated.post).toHaveBeenLastCalledWith(`${BASE}/feeds/6/move`, { folderId: null })
  })
})

describe('responses that wrap the payload', () => {
  it('reads the rules out of the /api/me/rules body', async () => {
    const api = useNewsApi()
    httpAuthenticated.get.mockResolvedValue({ data: { rules: { markRead: true } } })
    await expect(api.myRules()).resolves.toEqual({ markRead: true })

    httpAuthenticated.put.mockResolvedValue({ data: { rules: { markRead: false } } })
    await expect(api.updateMyRules({ markRead: false } as never)).resolves.toEqual({ markRead: false })
  })

  it('returns the share token of a shared item', async () => {
    const api = useNewsApi()
    httpAuthenticated.post.mockResolvedValue({ data: { share: { token: 'abc', url: '/s/abc' } } })
    await expect(api.shareItem(11)).resolves.toEqual({ share: { token: 'abc', url: '/s/abc' } })
    expect(httpAuthenticated.post).toHaveBeenLastCalledWith(`${BASE}/items/11/share`, {})
  })
})
