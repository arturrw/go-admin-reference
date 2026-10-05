import { Link } from '@tanstack/react-router'
import { ArrowRight, Check, ChevronRight, Download, Pencil, X } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { AreaChart } from '@/components/charts/area-chart'
import { Donut, ProgressRing, SERIES_COLORS } from '@/components/charts/donut'
import { Heatmap, HeatmapScale } from '@/components/charts/heatmap'
import { Sparkline } from '@/components/charts/sparkline'
import { Button } from '@/components/ui/button'
import { Card, CardHeader } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Avatar, Delta, IconTile, PageHeader, Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Segmented } from '@/components/ui/segmented'
import { ActivityItem } from '@/features/activity/activity-item'
import { OrderListItem } from '@/features/orders/order-list-item'
import { ApiError, type Dashboard, type KPI, type Target } from '@/lib/api'
import { useCan, useMe } from '@/lib/auth'
import { downloadCsv } from '@/lib/download'
import { compact, int, money, timeAgo } from '@/lib/format'
import { PeekButton, usePeek } from '@/lib/peek'
import { useDashboard, useSetTarget } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { formatKpi, kpiStyle } from './kpi'
import { KpiSheet } from './kpi-sheet'
import { LiveSheet } from './live-sheet'
import { GoRuntimeCard, LiveTraffic } from './live-traffic'

const RANGES = [7, 30, 90] as const

export function DashboardPage() {
  const [range, setRange] = useState<number>(30)
  const { data, isPending } = useDashboard(range)
  const me = useMe()

  return (
    <>
      <PageHeader title={`${greeting()}, ${me.user.name.split(' ')[0]}`} description="Here’s what’s happening across your store today.">
        <Segmented value={range} onChange={setRange} options={RANGES.map((r) => ({ value: r, label: `${r}d` }))} />
        <Button disabled={!data} onClick={() => data && exportDashboard(data)}>
          <Download />
          Export
        </Button>
      </PageHeader>

      {isPending || !data ? <DashboardSkeleton /> : <Bento data={data} />}
    </>
  )
}

/** Dashboard snapshot as one CSV: summary, daily series, categories, top products, markets. */
function exportDashboard(d: Dashboard) {
  const dollars = (c: number) => (c / 100).toFixed(2)
  const total = d.categories.reduce((s, c) => s + c.salesCents, 0) || 1
  const kpi = (key: string) => d.kpis.find((k) => k.key === key)
  const rows: unknown[][] = [
    ['Dashboard export', new Date().toISOString(), `last ${d.rangeDays} days`],
    [],
    ['Metric', 'Value', 'Change vs previous %'],
    ['Net revenue', dollars(d.revenueCents), ((d.revenueCents / d.prevRevenueCents - 1) * 100).toFixed(1)],
    ['Previous revenue', dollars(d.prevRevenueCents), ''],
    ...d.kpis.map((k) => [k.label, k.unit === 'cents' ? dollars(k.value) : k.value, k.deltaPct]),
    [],
    ['Date', 'Revenue', 'Previous revenue', 'Orders', 'New customers', 'Conversion %', 'Avg. order value'],
    ...d.revenue.map((p, i) => [
      p.date,
      dollars(p.current),
      dollars(p.previous),
      kpi('orders')?.series[i]?.current ?? '',
      kpi('customers')?.series[i]?.current ?? '',
      kpi('conversion')?.series[i]?.current ?? '',
      kpi('aov')?.series[i] ? dollars(kpi('aov')!.series[i].current) : '',
    ]),
    [],
    ['Category', 'Sales', 'Share %'],
    ...d.categories.map((c) => [c.category, dollars(c.salesCents), ((c.salesCents / total) * 100).toFixed(1)]),
    [],
    ['Top product', 'Category', 'Units sold', 'Revenue'],
    ...d.topProducts.map((p) => [p.name, p.category, p.sold, dollars(p.revenueCents)]),
    [],
    ['Market', 'Share %'],
    ...d.markets.map((m) => [m.name, m.sharePct]),
  ]
  downloadCsv(`dashboard-${d.rangeDays}d`, rows)
}

function Bento({ data }: { data: Dashboard }) {
  const [kpi, setKpi] = useState<string | null>(null)
  const [live, setLive] = useState(false)
  return (
    <div className="grid grid-cols-12 gap-3.5">
      <RevenueCard data={data} className="col-span-12 xl:col-span-8" />

      <div className="col-span-12 grid grid-cols-2 gap-3.5 sm:grid-cols-4 xl:col-span-4 xl:grid-cols-2">
        {data.kpis.map((k) => (
          <KpiCard key={k.key} kpi={k} onOpen={() => setKpi(k.key)} />
        ))}
        <LiveTraffic onOpen={() => setLive(true)} />
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

      {kpi && <KpiSheet dashboard={data} initial={kpi} onClose={() => setKpi(null)} />}
      {live && <LiveSheet onClose={() => setLive(false)} />}
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

function KpiCard({ kpi, onOpen }: { kpi: KPI; onOpen: () => void }) {
  const { icon: Icon, color } = kpiStyle(kpi.key)
  return (
    <Card
      role="button"
      tabIndex={0}
      aria-label={`${kpi.label}: open details`}
      onClick={onOpen}
      onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), onOpen())}
      className="group flex cursor-pointer flex-col gap-1.5 p-4 transition-colors hover:border-line-2"
    >
      <div className="flex items-center gap-2 text-[12.5px] text-muted">
        <IconTile icon={Icon} color={color} />
        {kpi.label}
        <ChevronRight className="ml-auto size-3.5 text-dim opacity-0 transition-opacity group-hover:opacity-100" />
      </div>
      <div className="num mt-1 text-2xl font-semibold tracking-[-0.03em]">{formatKpi(kpi)}</div>
      <div className="flex items-center justify-between gap-2">
        <Delta value={kpi.deltaPct} />
        <Sparkline data={kpi.trend} color={kpi.deltaPct < 0 ? 'var(--color-danger)' : color} className="w-auto max-w-21 min-w-0 flex-1" />
      </div>
    </Card>
  )
}

function CategoryCard({ data, className }: { data: Dashboard; className?: string }) {
  const [active, setActive] = useState<number | null>(null)
  const total = data.categories.reduce((s, c) => s + c.salesCents, 0) || 1
  const share = (cents: number) => `${((cents / total) * 100).toFixed(1)}%`
  const cur = active === null ? null : data.categories[active]
  return (
    <Card className={className}>
      <CardHeader title="Sales by category" />
      <div className="flex flex-col items-stretch gap-4.5">
        <div className="self-center">
          <Donut
            values={data.categories.map((c) => c.salesCents)}
            labels={data.categories.map((c) => `${c.category}: ${share(c.salesCents)}`)}
            active={active}
            onActive={setActive}
          >
            {cur ? (
              <>
                <b className="num text-lg font-semibold" style={{ color: SERIES_COLORS[active!] }}>
                  {share(cur.salesCents)}
                </b>
                <small className="text-[11px] text-muted">{cur.category}</small>
                <small className="num text-[10.5px] text-dim">{money(cur.salesCents)}</small>
              </>
            ) : (
              <>
                <b className="num text-lg font-semibold">{compact(total / 100)}</b>
                <small className="text-[11px] text-dim">gross sales</small>
              </>
            )}
          </Donut>
        </div>
        <div className="flex flex-col gap-0.5 text-[12.5px]" onMouseLeave={() => setActive(null)}>
          {data.categories.map((c, i) => (
            <div
              key={c.category}
              onMouseEnter={() => setActive(i)}
              className={cn(
                '-mx-1.5 flex items-center gap-2 rounded-md px-1.5 py-[3px] transition-[background-color,opacity]',
                active === i && 'bg-panel-2',
                active !== null && active !== i && 'opacity-50',
              )}
            >
              <i className="size-2 rounded-[2px]" style={{ background: SERIES_COLORS[i] }} />
              <span className="flex-1 text-muted">{c.category}</span>
              <b className="num text-xs font-medium">{share(c.salesCents)}</b>
            </div>
          ))}
        </div>
      </div>
    </Card>
  )
}

function TopProductsCard({ data, className }: { data: Dashboard; className?: string }) {
  const top = data.topProducts[0]?.revenueCents || 1
  const canOpen = useCan('products:read')
  return (
    <Card className={className}>
      <CardHeader title="Top products" sub="by revenue">
        <Link to="/products" className="flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-[12.5px] text-muted hover:bg-panel-2 hover:text-fg">
          View all <ArrowRight className="size-3.5" />
        </Link>
      </CardHeader>
      <div className="flex flex-col">
        {data.topProducts.map((p, i) => (
          <PeekButton
            key={p.id}
            kind="product"
            id={p.id}
            disabled={!canOpen}
            label={`Open ${p.name}`}
            className="-mx-2 flex items-center gap-3 rounded-lg border-b border-dashed border-line px-2 py-2.5 text-left transition-colors last:border-0 enabled:hover:bg-panel-2"
          >
            <span className="num w-4 text-[11px] text-dim">0{i + 1}</span>
            <ProductThumb category={p.category} hue={p.hue} src={p.imageUrl} size={34} />
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
          </PeekButton>
        ))}
      </div>
    </Card>
  )
}

function RecentOrdersCard({ data, className }: { data: Dashboard; className?: string }) {
  const peek = usePeek()
  const canOrders = useCan('orders:read')
  const canCustomers = useCan('customers:read')
  return (
    <Card className={cn('flex flex-col pb-0', className)}>
      <CardHeader title="Recent orders">
        <Link to="/orders" className="flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-[12.5px] text-muted hover:bg-panel-2 hover:text-fg">
          All orders <ArrowRight className="size-3.5" />
        </Link>
      </CardHeader>
      <div className="-mx-4.5 border-t border-line sm:hidden">
        {data.recentOrders.map((o) => (
          <OrderListItem key={o.id} o={o} onOpen={canOrders ? () => peek('order', o.id) : undefined} />
        ))}
      </div>
      <div className="-mx-4.5 overflow-x-auto border-t border-line max-sm:hidden">
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
              <tr
                key={o.id}
                className={cn(canOrders && 'cursor-pointer')}
                onClick={canOrders ? () => peek('order', o.id) : undefined}
              >
                <td className="num">
                  <PeekButton kind="order" id={o.id} disabled={!canOrders} className="hover:text-accent">
                    #{o.id}
                  </PeekButton>
                </td>
                <td>
                  {canCustomers ? (
                    <PeekButton
                      kind="customer"
                      id={o.customer.id}
                      label={`Open customer ${o.customer.name}`}
                      className="group/c -my-1 -ml-1.5 inline-flex items-center gap-2.5 rounded-lg py-1 pr-2 pl-1.5 hover:bg-panel-3"
                    >
                      <Avatar name={o.customer.name} size={26} />
                      <b className="font-medium group-hover/c:text-accent">{o.customer.name}</b>
                    </PeekButton>
                  ) : (
                    <div className="flex items-center gap-2.5">
                      <Avatar name={o.customer.name} size={26} />
                      <b className="font-medium">{o.customer.name}</b>
                    </div>
                  )}
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

function ActivityCard({ data, className }: { data: Dashboard; className?: string }) {
  const canAll = useCan('team:read')
  return (
    <Card className={className}>
      <CardHeader title="Activity" sub="live">
        {canAll && (
          <Link to="/activity" className="flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-[12.5px] text-muted hover:bg-panel-2 hover:text-fg">
            View all <ArrowRight className="size-3.5" />
          </Link>
        )}
      </CardHeader>
      <div className="flex flex-col">
        {data.activity.length === 0 && <p className="py-6 text-center text-[13px] text-dim">Nothing yet.</p>}
        {data.activity.map((a) => (
          <ActivityItem key={a.id} a={a} />
        ))}
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
  const canEdit = useCan('workspace:manage')
  const [editing, setEditing] = useState(false)
  const ahead = t.pacePct >= 0
  return (
    <Card className={className}>
      <CardHeader title={t.label} sub={t.period}>
        {canEdit && !editing && (
          <Button variant="ghost" size="icon-sm" aria-label="Edit target" onClick={() => setEditing(true)}>
            <Pencil />
          </Button>
        )}
      </CardHeader>
      <div className="flex items-center gap-4.5 max-sm:flex-col max-sm:items-start">
        <ProgressRing value={Math.min(ratio, 1)} label={`${Math.round(ratio * 100)}%`} />
        <div className="flex flex-1 flex-col gap-2.5">
          <Stat label="Booked" value={money(t.bookedCents)} />
          {editing ? <GoalEditor target={t} onDone={() => setEditing(false)} /> : <Stat label="Goal" value={money(t.goalCents)} />}
          <Stat label="Pace" value={`${ahead ? '+' : ''}${t.pacePct.toFixed(1)}% ${ahead ? 'ahead' : 'behind'}`} tone={ahead ? 'accent' : 'danger'} />
        </div>
      </div>
      {t.updatedBy && t.updatedAt && (
        <p className="mt-3.5 text-[11.5px] text-dim" data-testid="target-updated">
          Goal set by {t.updatedBy} · {timeAgo(t.updatedAt)}
        </p>
      )}
    </Card>
  )
}

/** Inline goal editor for the owner: dollars in, cents to the API. */
function GoalEditor({ target, onDone }: { target: Target; onDone: () => void }) {
  const [value, setValue] = useState(String(target.goalCents / 100))
  const save = useSetTarget()
  const cents = Math.round(parseFloat(value.replace(/[$,\s]/g, '')) * 100)
  const valid = Number.isFinite(cents) && cents > 0
  const error = save.error instanceof ApiError ? save.error.fields?.goalCents : undefined
  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (valid) save.mutate(cents, { onSuccess: onDone })
  }
  return (
    <form onSubmit={submit} onKeyDown={(e) => e.key === 'Escape' && onDone()}>
      <label className="block text-[11.5px] text-dim" htmlFor="goal-input">
        Goal, USD
      </label>
      <div className="mt-0.5 flex items-center gap-1.5">
        <Input id="goal-input" className="num h-8 min-w-0 flex-1 sm:max-w-36" inputMode="decimal" value={value} onChange={(e) => setValue(e.target.value)} autoFocus aria-invalid={!valid || undefined} />
        <Button type="submit" size="icon-sm" variant="primary" aria-label="Save target" disabled={!valid || save.isPending}>
          <Check />
        </Button>
        <Button size="icon-sm" variant="ghost" aria-label="Cancel" onClick={onDone}>
          <X />
        </Button>
      </div>
      {error && <span className="text-xs text-danger">{error}</span>}
    </form>
  )
}

function Stat({ label, value, tone }: { label: string; value: string; tone?: 'accent' | 'danger' }) {
  return (
    <div>
      <small className="block text-[11.5px] text-dim">{label}</small>
      <b className={cn('num text-[15px] font-medium', tone === 'accent' && 'text-accent', tone === 'danger' && 'text-danger')}>{value}</b>
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
