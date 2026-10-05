import { type LucideIcon, MousePointerClick, Receipt, ShoppingBag, UserPlus } from 'lucide-react'
import type { KPI } from '@/lib/api'
import { compact, int, money } from '@/lib/format'

export const KPI_STYLE: Record<string, { icon: LucideIcon; color: string }> = {
  orders: { icon: ShoppingBag, color: 'var(--color-accent)' },
  customers: { icon: UserPlus, color: 'var(--color-info)' },
  conversion: { icon: MousePointerClick, color: 'var(--color-danger)' },
  aov: { icon: Receipt, color: 'var(--color-violet)' },
}

export const kpiStyle = (key: string) => KPI_STYLE[key] ?? KPI_STYLE.orders

/** Formats one value of a KPI; `short` is for axis labels. */
export function formatUnit(unit: KPI['unit'], v: number, short = false) {
  if (unit === 'cents') return short ? '$' + compact(v / 100) : money(v, 2)
  if (unit === 'percent') return `${v.toFixed(2)}%`
  return short ? compact(v) : int(v)
}

export const formatKpi = (k: KPI) => (k.unit === 'count' ? compact(k.value) : formatUnit(k.unit, k.value))

/** Counts add up over a period; rates and averages are averaged. */
export const isAdditive = (k: KPI) => k.unit === 'count'
