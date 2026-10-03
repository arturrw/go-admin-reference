import { Activity, LayoutDashboard, type LucideIcon, Package, Settings, ShieldCheck, ShoppingBag, UsersRound } from 'lucide-react'

export type AppPath = '/' | '/products' | '/orders' | '/customers' | '/team' | '/requests' | '/settings'

export interface NavItem {
  to: AppPath
  label: string
  icon: LucideIcon
  badge?: 'products' | 'pendingOrders'
}

export const NAV: { label: string; items: NavItem[] }[] = [
  { label: 'Overview', items: [{ to: '/', label: 'Dashboard', icon: LayoutDashboard }] },
  {
    label: 'Commerce',
    items: [
      { to: '/products', label: 'Products', icon: Package, badge: 'products' },
      { to: '/orders', label: 'Orders', icon: ShoppingBag, badge: 'pendingOrders' },
      { to: '/customers', label: 'Customers', icon: UsersRound },
    ],
  },
  {
    label: 'System',
    items: [
      { to: '/team', label: 'Team & roles', icon: ShieldCheck },
      { to: '/requests', label: 'Request log', icon: Activity },
      { to: '/settings', label: 'Settings', icon: Settings },
    ],
  },
]

export const NAV_ITEMS = NAV.flatMap((g) => g.items)
