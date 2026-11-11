import { describe, expect, it, vi } from 'vitest'

vi.mock('vue3-gettext', async (importOriginal) => ({
  ...(await importOriginal<object>()),
  useGettext: () => ({ $gettext: (message: string) => message })
}))

import app from './index'

function setup() {
  return (app as unknown as { setup: (args: unknown) => any }).setup({ applicationConfig: {} })
}

describe('the news app registers what the host expects', () => {
  it('keeps the app id, icon and role colour', () => {
    const { appInfo } = setup()
    expect(appInfo.id).toBe('news')
    expect(appInfo.icon).toBe('rss')
    expect(appInfo.color).toBe('var(--oc-role-secondary)')
  })

  it('exposes one route per list with its own name', () => {
    const { routes } = setup()
    const byPath = Object.fromEntries(routes.map((r: any) => [r.path, r]))
    expect(Object.keys(byPath).sort()).toEqual(['/', '/feed/:feedId', '/folder/:folderId', '/podcasts', '/starred'])
    expect(routes.map((r: any) => r.name)).toEqual([
      'news-root',
      'news-feed',
      'news-folder',
      'news-starred',
      'news-podcasts'
    ])
  })

  it('keeps the root route mountable instead of redirecting it', () => {
    // A redirect on '/' used to shadow the component and loop (issue #001):
    // every route must resolve to the view and carry the user auth context.
    const { routes } = setup()
    for (const route of routes) {
      expect(route.redirect).toBeUndefined()
      expect(typeof route.component).toBe('function')
      expect(route.meta).toMatchObject({ authContext: 'user', title: 'News' })
    }
  })

  it('adds a single menu item pointing at the app path', () => {
    const { extensions } = setup()
    const items = extensions.value
    expect(items).toHaveLength(1)
    expect(items[0]).toMatchObject({
      id: 'app.news.menuItem',
      type: 'appMenuItem',
      icon: 'rss',
      path: '/news'
    })
    expect(items[0].label()).toBe('News')
  })

  it('ships the translations the host loads', () => {
    const { translations } = setup()
    expect(translations).toBeTypeOf('object')
  })
})
