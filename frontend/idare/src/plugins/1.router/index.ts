import { setupLayouts } from 'virtual:generated-layouts'
import type { App } from 'vue'

import type { RouteRecordRaw } from 'vue-router/auto'

import { createRouter, createWebHistory } from 'vue-router/auto'
import JwtService from '@/services/JwtService'
import { useUserStore } from '@/store/user'

function recursiveLayouts(route: RouteRecordRaw): RouteRecordRaw {
  if (route.children) {
    for (let i = 0; i < route.children.length; i++)
      route.children[i] = recursiveLayouts(route.children[i])

    return route
  }

  return setupLayouts([route])[0]
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  scrollBehavior(to) {
    if (to.hash)
      return { el: to.hash, behavior: 'smooth', top: 60 }

    return { top: 0 }
  },
  extendRoutes: pages => [
    ...[...pages].map(route => recursiveLayouts(route)),
  ],
})

router.beforeEach((to, from, next) => {
  const isLoggedIn = !!JwtService.getAccessToken()
  if (!isLoggedIn) {
    if (to.meta.redirectIfLoggedIn)
      return next()

    return next({ name: 'auth-login', query: { to: to.name !== 'root' ? to.fullPath : undefined } })
  }

  if (to.meta.redirectIfLoggedIn)
    return next('/')

  if (!useUserStore().hasRole(to.meta.role))
    return next({ name: 'not-authorized' })

  return next()
})

export { router }

export default function (app: App) {
  app.use(router)
}
