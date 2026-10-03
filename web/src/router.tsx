import { createRootRoute, createRoute, createRouter, Link, lazyRouteComponent } from '@tanstack/react-router'
import { Compass } from 'lucide-react'
import { AppShell } from '@/components/layout/app-shell'
import { buttonVariants } from '@/components/ui/button'

/** `?edit=new` or `?edit=<id>` opens the create/edit sheet on list pages. */
const editSearch = (search: Record<string, unknown>): { edit?: number | 'new' } => {
  if (search.edit === 'new') return { edit: 'new' }
  const id = Number(search.edit)
  return Number.isInteger(id) && id > 0 ? { edit: id } : {}
}

// Each page is its own chunk, loaded on navigation (or on hover via defaultPreload).
const rootRoute = createRootRoute({ component: AppShell, notFoundComponent: NotFound })

const routeTree = rootRoute.addChildren([
  createRoute({ getParentRoute: () => rootRoute, path: '/', component: lazyRouteComponent(() => import('@/features/dashboard/dashboard-page'), 'DashboardPage') }),
  createRoute({ getParentRoute: () => rootRoute, path: '/products', component: lazyRouteComponent(() => import('@/features/products/products-page'), 'ProductsPage'), validateSearch: editSearch }),
  createRoute({ getParentRoute: () => rootRoute, path: '/orders', component: lazyRouteComponent(() => import('@/features/orders/orders-page'), 'OrdersPage') }),
  createRoute({ getParentRoute: () => rootRoute, path: '/customers', component: lazyRouteComponent(() => import('@/features/customers/customers-page'), 'CustomersPage') }),
  createRoute({ getParentRoute: () => rootRoute, path: '/team', component: lazyRouteComponent(() => import('@/features/team/team-page'), 'TeamPage'), validateSearch: editSearch }),
  createRoute({ getParentRoute: () => rootRoute, path: '/requests', component: lazyRouteComponent(() => import('@/features/requests/requests-page'), 'RequestsPage') }),
  createRoute({ getParentRoute: () => rootRoute, path: '/settings', component: lazyRouteComponent(() => import('@/features/settings/settings-page'), 'SettingsPage') }),
])

export const router = createRouter({ routeTree, defaultPreload: 'intent', scrollRestoration: true })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
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
