import {
  defineWebApplication,
  ApplicationSetupOptions,
  Extension,
  AppMenuItemExtension
} from '@opencloud-eu/web-pkg'
import { urlJoin } from '@opencloud-eu/web-client'
import '@opencloud-eu/extension-sdk/tailwind.css'
import { RouteRecordRaw } from 'vue-router'
import { computed } from 'vue'
import { useGettext } from 'vue3-gettext'
import translations from '../l10n/translations.json'

export default defineWebApplication({
  setup(args) {
    const { $gettext } = useGettext()

    const appInfo = {
      id: 'news',
      name: $gettext('News'),
      icon: 'rss',
      color: '#ee7318'
    }

    // El host monta estas rutas bajo /news (via el menu item path).
    // UNA sola ruta por path: un redirect extra en '/' pisa el matching
    // del componente y provoca un bucle (issue #001).
    const routes: RouteRecordRaw[] = [
      {
        path: '/',
        name: 'news-root',
        component: () => import('./views/NewsApp.vue'),
        meta: {
          authContext: 'user',
          title: $gettext('News')
        }
      },
      {
        path: '/feed/:feedId',
        name: 'news-feed',
        component: () => import('./views/NewsApp.vue'),
        meta: {
          authContext: 'user',
          title: $gettext('News')
        }
      },
      {
        path: '/folder/:folderId',
        name: 'news-folder',
        component: () => import('./views/NewsApp.vue'),
        meta: {
          authContext: 'user',
          title: $gettext('News')
        }
      },
      {
        path: '/starred',
        name: 'news-starred',
        component: () => import('./views/NewsApp.vue'),
        meta: {
          authContext: 'user',
          title: $gettext('News')
        }
      },
      {
        path: '/podcasts',
        name: 'news-podcasts',
        component: () => import('./views/NewsApp.vue'),
        meta: {
          authContext: 'user',
          title: $gettext('News')
        }
      }
    ]

    const extensions = ({ applicationConfig }: ApplicationSetupOptions) => {
      return computed<Extension[]>(() => {
        const menuItems: AppMenuItemExtension[] = [
          {
            id: `app.${appInfo.id}.menuItem`,
            type: 'appMenuItem',
            label: () => appInfo.name,
            color: appInfo.color,
            icon: appInfo.icon,
            path: urlJoin(appInfo.id)
          }
        ]
        return [...menuItems]
      })
    }

    return {
      appInfo,
      routes,
      extensions: extensions(args),
      translations
    }
  }
})
