import type { QueryClient } from '@tanstack/react-query'
import { createRootRouteWithContext, createRoute, createRouter, lazyRouteComponent, Link, Outlet, redirect } from '@tanstack/react-router'
import { Compass, ShieldX } from 'lucide-react'
import { AppShell } from '@/components/layout/app-shell'
import { buttonVariants } from '@/components/ui/button'
import { LoginPage } from '@/features/auth/login-page'
import type { Me, Permission } from '@/lib/api'
import { meQuery } from '@/lib/auth'

interface RouterContext {
  queryClient: QueryClient
}

/** `?edit=new` or `?edit=<id>` opens the create/edit sheet on list pages. */
const editSearch = (search: Record<string, unknown>): { edit?: number | 'new' } => {
  if (search.edit === 'new') return { edit: 'new' }
  const id = Number(search.edit)
  return Number.isInteger(id) && id > 0 ? { edit: id } : {}
}

/** `?view=<id>` opens a detail sheet (orders, customers). */
const viewSearch = (search: Record<string, unknown>): { view?: number } => {
  const id = Number(search.view)
  return Number.isInteger(id) && id > 0 ? { view: id } : {}
}

const rootRoute = createRootRouteWithContext<RouterContext>()({
  component: Outlet,
  notFoundComponent: NotFound,
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  validateSearch: (s: Record<string, unknown>): { redirect?: string } => (typeof s.redirect === 'string' ? { redirect: s.redirect } : {}),
  component: LoginPage,
})

// Every page except /login lives under this layout, which requires a session.
const appRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'app',
  component: AppShell,
  beforeLoad: async ({ context, location }) => {
    const me = await context.queryClient.ensureQueryData(meQuery).catch(() => null)
    if (!me) throw redirect({ to: '/login', search: { redirect: location.href } })
    return { me }
  },
})

/** Route guard: sends members whose role lacks `perm` to the 403 page. */
const guard =
  (perm: Permission) =>
  ({ context }: { context: { me: Me } }) => {
    if (!context.me.permissions.includes(perm)) throw redirect({ to: '/forbidden' })
  }

// Each page is its own chunk, loaded on navigation (or on hover via defaultPreload).
const routeTree = rootRoute.addChildren([
  loginRoute,
  appRoute.addChildren([
    createRoute({
      getParentRoute: () => appRoute,
      path: '/',
      beforeLoad: guard('dashboard:read'),
      component: lazyRouteComponent(() => import('@/features/dashboard/dashboard-page'), 'DashboardPage'),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: '/products',
      validateSearch: editSearch,
      beforeLoad: guard('products:read'),
      component: lazyRouteComponent(() => import('@/features/products/products-page'), 'ProductsPage'),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: '/orders',
      validateSearch: viewSearch,
      beforeLoad: guard('orders:read'),
      component: lazyRouteComponent(() => import('@/features/orders/orders-page'), 'OrdersPage'),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: '/customers',
      validateSearch: viewSearch,
      beforeLoad: guard('customers:read'),
      component: lazyRouteComponent(() => import('@/features/customers/customers-page'), 'CustomersPage'),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: '/team',
      validateSearch: editSearch,
      beforeLoad: guard('team:read'),
      component: lazyRouteComponent(() => import('@/features/team/team-page'), 'TeamPage'),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: '/activity',
      validateSearch: (s: Record<string, unknown>): { actor?: number } => {
        const id = Number(s.actor)
        return Number.isInteger(id) && id > 0 ? { actor: id } : {}
      },
      beforeLoad: guard('team:read'),
      component: lazyRouteComponent(() => import('@/features/activity/activity-page'), 'ActivityPage'),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: '/requests',
      beforeLoad: guard('requests:read'),
      component: lazyRouteComponent(() => import('@/features/requests/requests-page'), 'RequestsPage'),
    }),
    createRoute({
      getParentRoute: () => appRoute,
      path: '/settings',
      component: lazyRouteComponent(() => import('@/features/settings/settings-page'), 'SettingsPage'),
    }),
    createRoute({ getParentRoute: () => appRoute, path: '/forbidden', component: Forbidden }),
  ]),
])

export function createAppRouter(queryClient: QueryClient) {
  return createRouter({ routeTree, context: { queryClient }, defaultPreload: 'intent', scrollRestoration: true })
}

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof createAppRouter>
  }
}

function NotFound() {
  return (
    <div className="flex flex-col items-center gap-3 py-24 text-center">
      <Compass className="size-8 text-dim" />
      <h1 className="text-xl font-semibold">Page not found</h1>
      <p className="text-muted">The page you’re looking for doesn’t exist.</p>
      <Link to="/" className={buttonVariants({ variant: 'primary' })}>
        Back to dashboard
      </Link>
    </div>
  )
}

function Forbidden() {
  return (
    <div className="flex flex-col items-center gap-3 py-24 text-center">
      <ShieldX className="size-8 text-danger" />
      <h1 className="text-xl font-semibold">No access</h1>
      <p className="max-w-sm text-muted">Your role doesn’t include this section. Ask an owner or admin if you need it.</p>
      <Link to="/" className={buttonVariants({ variant: 'primary' })}>
        Back to dashboard
      </Link>
    </div>
  )
}
