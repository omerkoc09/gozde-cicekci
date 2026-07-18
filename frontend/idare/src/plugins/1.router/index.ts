import { setupLayouts } from 'virtual:generated-layouts'
import type { App } from 'vue'

import type { RouteRecordRaw } from 'vue-router/auto'

import { createRouter, createWebHistory } from 'vue-router/auto'
import { useUserStore } from '@/store/user'

// setupLayouts(pages) tüm sayfaları düz uygular; unplugin-vue-router'ın
// dizin bazlı grup route'larına (örn. component'siz /auth parent'ı) da
// layout sarar ve iç içe iki layout render edilir (sidebar login'de sızar).
// Leaf route'ları teker teker sarmak bunu önler.
//
// /siparisler.vue + /siparisler/[id].vue gibi "aynı isimde dosya + klasör"
// desenlerinde route'un hem component'i hem children'ı olur — bu durumda
// children'a inmeden önce route'un kendisini de setupLayouts ile sarmak
// gerekir, yoksa o sayfa layout'suz (sidebar'sız) render edilir.
function recursiveLayouts(route: RouteRecordRaw): RouteRecordRaw {
  if (route.children) {
    for (let i = 0; i < route.children.length; i++)
      route.children[i] = recursiveLayouts(route.children[i])

    return route.component ? setupLayouts([route])[0] : route
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

// Oturum bir kez sunucuya sorulur. Token HttpOnly cookie'de olduğu için
// JavaScript ona bakamaz — sayfa ilk açıldığında /me çağrısı yapılmadan
// oturumun var olup olmadığı bilinemez.
let sessionChecked = false

router.beforeEach(async to => {
  const userStore = useUserStore()

  if (!sessionChecked) {
    sessionChecked = true
    await userStore.checkSession()
  }

  // Girişliyken login sayfası → panele gönder
  if (to.meta.redirectIfLoggedIn)
    return userStore.isAuthenticated ? '/' : true

  if (!userStore.isAuthenticated)
    return { name: 'auth-login', query: { to: to.name !== 'root' ? to.fullPath : undefined } }

  return true
})

export { router }

export default function (app: App) {
  app.use(router)
}
