import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  Download,
  type LucideIcon,
  MousePointerClick,
  PackagePlus,
  Receipt,
  Rocket,
  ShoppingBag,
  TriangleAlert,
  Undo2,
  UserCheck,
  UserPlus,
} from 'lucide-react'
import { useState } from 'react'
import { AreaChart } from '@/components/charts/area-chart'
import { Donut, ProgressRing, SERIES_COLORS } from '@/components/charts/donut'
import { Heatmap, HeatmapScale } from '@/components/charts/heatmap'
import { Sparkline } from '@/components/charts/sparkline'
import { Button } from '@/components/ui/button'
import { Card, CardHeader } from '@/components/ui/card'
import { Avatar, Delta, PageHeader, Skeleton } from '@/components/ui/misc'
import { StatusPill, type Tone, toneColor } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Segmented } from '@/components/ui/segmented'
import type { Dashboard, KPI } from '@/lib/api'
import { compact, int, money, timeAgo } from '@/lib/format'
import { useDashboard } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { GoRuntimeCard, LiveTraffic } from './live-traffic'

const RANGES = [7, 30, 90] as const

export function DashboardPage() {
  const [range, setRange] = useState<number>(30)
  const { data, isPending } = useDashboard(range)

  return (
    <>
      <PageHeader title={greeting() + ', Anna'} description="Here’s what’s happening across your store today.">
        <Segmented value={range} onChange={setRange} options={RANGES.map((r) => ({ value: r, label: `${r}d` }))} />
        <Button>
          <Download />
          Export
        </Button>
      </PageHeader>

      {isPending || !data ? <DashboardSkeleton /> : <Bento data={data} />}
    </>
  )
}

function Bento({ data }: { data: Dashboard }) {
  return (
    <div className="grid grid-cols-12 gap-3.5">
      <RevenueCard data={data} className="col-span-12 xl:col-span-8" />

      <div className="col-span-12 grid grid-cols-2 gap-3.5 sm:grid-cols-4 xl:col-span-4 xl:grid-cols-2">
        {data.kpis.map((k) => (
          <KpiCard key={k.key} kpi={k} />
        ))}
        <LiveTraffic />
      </div>

      <Card className="col-span-12 lg:col-span-6 xl:col-span-5">
        <CardHeader title="Orders by hour" sub="last 4 weeks">
          <HeatmapScale />
        </CardHeader>
        <Heatmap data={data.ordersHeatmap} />
      </Card>

      <CategoryCard data={data} className="col-span-12 sm:col-span-6 xl:col-span-3" />
      <GoRuntimeCard className="col-span-12 sm:col-span-6 xl:col-span-4" />

      <TopProductsCard data={data} className="col-span-12 lg:col-span-6 xl:col-span-5" />
      <RecentOrdersCard data={data} className="col-span-12 lg:col-span-6 xl:col-span-7" />

      <ActivityCard data={data} className="col-span-12 md:col-span-6 xl:col-span-4" />
      <MarketsCard data={data} className="col-span-12 md:col-span-6 xl:col-span-4" />
      <TargetCard data={data} className="col-span-12 xl:col-span-4" />
    </div>
  )
}

function RevenueCard({ data, className }: { data: Dashboard; className?: string }) {
  const total = money(data.revenueCents)
  return (
    <Card className={className}>
      <div className="mb-3.5 flex items-center gap-2.5">
        <div className="eyebrow">Net revenue · last {data.rangeDays} days</div>
        <div className="ml-auto flex items-center gap-3.5 text-xs text-muted">
          <span className="flex items-center gap-1.5">
            <i className="h-[3px] w-2.5 rounded-xs bg-accent" />
            This period
          </span>
          <span className="flex items-center gap-1.5">
            <i className="h-[3px] w-2.5 rounded-xs bg-dim" />
            Previous
          </span>
        </div>
      </div>
      <div className="flex flex-wrap items-end gap-3.5">
        <div className="num text-[34px] leading-none font-semibold tracking-[-0.04em] md:text-[42px]">
          <span className="text-dim">$</span>
          {total.slice(1)}
        </div>
        <div className="flex items-center gap-2 pb-1.5">
          <Delta value={(data.revenueCents / data.prevRevenueCents - 1) * 100} />
          <span className="text-[12.5px] text-dim">vs {money(data.prevRevenueCents)}</span>
        </div>
      </div>
      <AreaChart points={data.revenue} />
    </Card>
  )
}

const KPI_STYLE: Record<string, { icon: LucideIcon; color: string }> = {
  orders: { icon: ShoppingBag, color: 'var(--color-accent)' },
  customers: { icon: UserPlus, color: 'var(--color-info)' },
  conversion: { icon: MousePointerClick, color: 'var(--color-danger)' },
  aov: { icon: Receipt, color: 'var(--color-violet)' },
}

function formatKpi(k: KPI) {
  if (k.unit === 'cents') return money(k.value, 2)
  if (k.unit === 'percent') return `${k.value.toFixed(2)}%`
  return compact(k.value)
}

function KpiCard({ kpi }: { kpi: KPI }) {
  const { icon: Icon, color } = KPI_STYLE[kpi.key] ?? KPI_STYLE.orders
  return (
    <Card className="flex flex-col gap-1.5 p-4">
      <div className="flex items-center gap-2 text-[12.5px] text-muted">
        <span className="grid size-6.5 place-items-center rounded-lg bg-panel-3">
          <Icon className="size-3.5" />
        </span>
        {kpi.label}
      </div>
      <div className="num mt-1 text-2xl font-semibold tracking-[-0.03em]">{formatKpi(kpi)}</div>
      <div className="flex items-center justify-between gap-2">
        <Delta value={kpi.deltaPct} />
        <Sparkline data={kpi.trend} color={kpi.deltaPct < 0 ? 'var(--color-danger)' : color} />
      </div>
    </Card>
  )
}

function CategoryCard({ data, className }: { data: Dashboard; className?: string }) {
  const total = data.categories.reduce((s, c) => s + c.salesCents, 0) || 1
  return (
    <Card className={className}>
      <CardHeader title="Sales by category" />
      <div className="flex flex-col items-stretch gap-4.5">
        <div className="self-center">
          <Donut values={data.categories.map((c) => c.salesCents)}>
            <b className="num text-lg font-semibold">{compact(total / 100)}</b>
            <small className="text-[11px] text-dim">gross sales</small>
          </Donut>
        </div>
        <div className="flex flex-col gap-2 text-[12.5px]">
          {data.categories.map((c, i) => (
            <div key={c.category} className="flex items-center gap-2">
              <i className="size-2 rounded-[2px]" style={{ background: SERIES_COLORS[i] }} />
              <span className="flex-1 text-muted">{c.category}</span>
              <b className="num text-xs font-medium">{((c.salesCents / total) * 100).toFixed(1)}%</b>
            </div>
          ))}
        </div>
      </div>
    </Card>
  )
}

function TopProductsCard({ data, className }: { data: Dashboard; className?: string }) {
  const top = data.topProducts[0]?.revenueCents || 1
  return (
    <Card className={className}>
      <CardHeader title="Top products" sub="by revenue">
        <Link to="/products" className="flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-[12.5px] text-muted hover:bg-panel-2 hover:text-fg">
          View all <ArrowRight className="size-3.5" />
        </Link>
      </CardHeader>
      <div className="flex flex-col">
        {data.topProducts.map((p, i) => (
          <div key={p.id} className="flex items-center gap-3 border-b border-dashed border-line py-2.5 last:border-0">
            <span className="num w-4 text-[11px] text-dim">0{i + 1}</span>
            <ProductThumb category={p.category} hue={p.hue} size={34} />
            <div className="min-w-0 flex-1">
              <b className="block truncate font-medium">{p.name}</b>
              <div className="mt-1.5 h-1 overflow-hidden rounded bg-panel-3">
                <i className="block h-full rounded bg-linear-to-r from-accent/40 to-accent" style={{ width: `${(p.revenueCents / top) * 100}%` }} />
              </div>
            </div>
            <div className="text-right">
              <b className="num block text-[13px] font-medium">{money(p.revenueCents)}</b>
              <small className="text-[11.5px] text-dim">{int(p.sold)} sold</small>
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}

function RecentOrdersCard({ data, className }: { data: Dashboard; className?: string }) {
  return (
    <Card className={cn('flex flex-col pb-0', className)}>
      <CardHeader title="Recent orders">
        <Link to="/orders" className="flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-[12.5px] text-muted hover:bg-panel-2 hover:text-fg">
          All orders <ArrowRight className="size-3.5" />
        </Link>
      </CardHeader>
      <div className="-mx-4.5 overflow-x-auto border-t border-line">
        <table className="data-table">
          <thead>
            <tr>
              <th>Order</th>
              <th>Customer</th>
              <th>Status</th>
              <th className="num">Total</th>
              <th className="num">When</th>
            </tr>
          </thead>
          <tbody>
            {data.recentOrders.map((o) => (
              <tr key={o.id}>
                <td className="num">#{o.id}</td>
                <td>
                  <div className="flex items-center gap-2.5">
                    <Avatar name={o.customer.name} size={26} />
                    <b className="font-medium">{o.customer.name}</b>
                  </div>
                </td>
                <td>
                  <StatusPill status={o.status} />
                </td>
                <td className="num">{money(o.totalCents, 2)}</td>
                <td className="num text-muted">{timeAgo(o.placedAt)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}

const ACTIVITY: Record<string, { icon: LucideIcon; tone: Tone }> = {
  role: { icon: UserCheck, tone: 'lime' },
  publish: { icon: PackagePlus, tone: 'blue' },
  deploy: { icon: Rocket, tone: 'violet' },
  stock: { icon: TriangleAlert, tone: 'amber' },
  refund: { icon: Undo2, tone: 'red' },
}

function ActivityCard({ data, className }: { data: Dashboard; className?: string }) {
  return (
    <Card className={className}>
      <CardHeader title="Activity" />
      <div className="flex flex-col">
        {data.activity.map((a, i) => {
          const { icon: Icon, tone } = ACTIVITY[a.kind] ?? ACTIVITY.role
          const c = toneColor(tone)
          return (
            <div key={i} className="relative flex gap-3 py-2 not-last:after:absolute not-last:after:top-9 not-last:after:-bottom-1.5 not-last:after:left-[13px] not-last:after:w-px not-last:after:bg-line-2">
              <span className="grid size-[27px] shrink-0 place-items-center rounded-lg" style={{ color: c, background: `color-mix(in srgb, ${c} 13%, transparent)` }}>
                <Icon className="size-3.5" />
              </span>
              <div>
                <p className="text-[13px] text-muted">
                  <b className="font-medium text-fg">{a.actor}</b> {a.message}
                </p>
                <small className="num text-[11px] text-dim">{timeAgo(a.at)}</small>
              </div>
            </div>
          )
        })}
      </div>
    </Card>
  )
}

function MarketsCard({ data, className }: { data: Dashboard; className?: string }) {
  const max = Math.max(...data.markets.map((m) => m.sharePct), 1)
  return (
    <Card className={className}>
      <CardHeader title="Top markets" sub="share of revenue" />
      {data.markets.map((m) => (
        <div key={m.country} className="grid grid-cols-[34px_1fr_auto] items-center gap-2.5 py-1.5">
          <span className="num rounded-[5px] border border-line-2 py-0.5 text-center text-[10.5px] font-semibold text-muted">{m.country}</span>
          <div>
            <div className="mb-1 text-[12.5px] text-muted">{m.name}</div>
            <div className="h-2 overflow-hidden rounded bg-panel-3">
              <i className="block h-full rounded bg-info" style={{ width: `${(m.sharePct / max) * 100}%` }} />
            </div>
          </div>
          <b className="num text-xs font-medium">{m.sharePct.toFixed(1)}%</b>
        </div>
      ))}
    </Card>
  )
}

function TargetCard({ data, className }: { data: Dashboard; className?: string }) {
  const t = data.target
  const ratio = t.bookedCents / t.goalCents
  return (
    <Card className={className}>
      <CardHeader title={t.label} sub="Oct – Dec" />
      <div className="flex items-center gap-4.5 max-sm:flex-col max-sm:items-start">
        <ProgressRing value={ratio} label={`${Math.round(ratio * 100)}%`} />
        <div className="flex flex-1 flex-col gap-2.5">
          <Stat label="Booked" value={money(t.bookedCents)} />
          <Stat label="Goal" value={money(t.goalCents)} />
          <Stat label="Pace" value={`+${t.pacePct}% ahead`} accent />
        </div>
      </div>
    </Card>
  )
}

function Stat({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div>
      <small className="block text-[11.5px] text-dim">{label}</small>
      <b className={cn('num text-[15px] font-medium', accent && 'text-accent')}>{value}</b>
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <div className="grid grid-cols-12 gap-3.5">
      <Skeleton className="col-span-12 h-90 xl:col-span-8" />
      <Skeleton className="col-span-12 h-90 xl:col-span-4" />
      {Array.from({ length: 3 }, (_, i) => (
        <Skeleton key={i} className="col-span-12 h-64 md:col-span-4" />
      ))}
    </div>
  )
}

function greeting() {
  const h = new Date().getHours()
  return h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening'
}
