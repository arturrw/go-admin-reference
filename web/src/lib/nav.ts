import { Activity, LayoutDashboard, type LucideIcon, Package, Settings, ShieldCheck, ShoppingBag, UsersRound } from 'lucide-react'
import type { Permission } from './api'

export type AppPath = '/' | '/products' | '/orders' | '/customers' | '/team' | '/requests' | '/settings'

export interface NavItem {
  to: AppPath
  label: string
  icon: LucideIcon
  /** Hidden from members without this permission; the route is guarded too. */
  perm?: Permission
  badge?: 'products' | 'pendingOrders'
}

export const NAV: { label: string; items: NavItem[] }[] = [
  { label: 'Overview', items: [{ to: '/', label: 'Dashboard', icon: LayoutDashboard, perm: 'dashboard:read' }] },
  {
    label: 'Commerce',
    items: [
      { to: '/products', label: 'Products', icon: Package, perm: 'products:read', badge: 'products' },
      { to: '/orders', label: 'Orders', icon: ShoppingBag, perm: 'orders:read', badge: 'pendingOrders' },
      { to: '/customers', label: 'Customers', icon: UsersRound, perm: 'customers:read' },
    ],
  },
  {
    label: 'System',
    items: [
      { to: '/team', label: 'Team & roles', icon: ShieldCheck, perm: 'team:read' },
      { to: '/requests', label: 'Request log', icon: Activity, perm: 'requests:read' },
      { to: '/settings', label: 'Settings', icon: Settings },
    ],
  },
]

export const NAV_ITEMS = NAV.flatMap((g) => g.items)

export const visibleNav = (perms: Permission[]) =>
  NAV.map((g) => ({ ...g, items: g.items.filter((i) => !i.perm || perms.includes(i.perm)) })).filter((g) => g.items.length > 0)
