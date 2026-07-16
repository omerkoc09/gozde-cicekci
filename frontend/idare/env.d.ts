import 'vue-router'
import {UserRole} from "@/store/user";

declare module 'vue-router' {
  interface RouteMeta {
    role?: UserRole[]
    action?: string
    subject?: string
    layoutWrapperClasses?: string
    navActiveLink?: RouteLocationRaw
    layout?: 'blank' | 'default'
    unauthenticatedOnly?: boolean
    public?: boolean
  }
}
