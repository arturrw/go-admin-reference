import { ArrowLeft, ChevronLeft, ChevronRight, DollarSign } from 'lucide-react'
import { useState } from 'react'
import { AreaChart } from '@/components/charts/area-chart'
import { Button } from '@/components/ui/button'
import { Avatar, Delta, IconTile, Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Sheet } from '@/components/ui/sheet'
import { Tabs } from '@/components/ui/tabs'
import { OrderListItem } from '@/features/orders/order-list-item'
import type { Dashboard, KPI } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { int, money, shortDate } from '@/lib/format'
import { PeekButton, usePeek } from '@/lib/peek'
import { useOrders } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { formatKpi, formatUnit, isAdditive, kpiStyle } from './kpi'

const longDay = (iso: string) => localDay(iso).toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })
const rowDay = (iso: string) => localDay(iso).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })

/** "YYYY-MM-DD" as local midnight (Date parses bare dates as UTC). */
function localDay(iso: string) {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(y, m - 1, d)
}

const change = (cur: number, prev: number) => (prev ? (cur / prev - 1) * 100 : 0)

/**
 * Full-size view of the KPI cards: daily chart, period stats and a per-day
 * table. Clicking a day (row or chart point) opens that day in detail.
 */
export function KpiSheet({ dashboard, initial, onClose }: { dashboard: Dashboard; initial: string; onClose: () => void }) {
  const { kpis, rangeDays } = dashboard
  const [key, setKey] = useState(initial)
  const [day, setDay] = useState<number | null>(null)
  const k = kpis.find((x) => x.key === key) ?? kpis[0]
  const dates = k.series.map((p) => p.date)

  return (
    <Sheet
      open
      onOpenChange={(v) => !v && onClose()}
      size="lg"
      title={day === null ? k.label : longDay(dates[day])}
      description={day === null ? `Daily breakdown · last ${rangeDays} days vs the ${rangeDays} days before` : `${k.label} and every other metric for this day`}
    >
      <Tabs value={k.key} onChange={setKey} tabs={kpis.map((x) => ({ value: x.key, label: x.label }))} />
      {day === null ? (
        <Overview k={k} onDay={setDay} />
      ) : (
        <DayDetail dashboard={dashboard} active={k.key} index={day} onIndex={setDay} onBack={() => setDay(null)} />
      )}
    </Sheet>
  )
}

function Overview({ k, onDay }: { k: KPI; onDay: (i: number) => void }) {
  const { icon: Icon, color } = kpiStyle(k.key)
  const tint = k.deltaPct < 0 ? 'var(--color-danger)' : color
  const fmt = (v: number) => formatUnit(k.unit, v)

  const cur = k.series.map((p) => p.current)
  const prev = k.series.map((p) => p.previous)
  const agg = (xs: number[]) => (isAdditive(k) ? xs.reduce((a, b) => a + b, 0) : xs.reduce((a, b) => a + b, 0) / Math.max(xs.length, 1))
  const peak = k.series.reduce((a, b) => (b.current > a.current ? b : a), k.series[0])
  const low = k.series.reduce((a, b) => (b.current < a.current ? b : a), k.series[0])
  const maxDay = Math.max(...cur, 1)
  const last = k.series.length - 1

  return (
    <>
      <div className="flex flex-wrap items-end gap-3.5">
        <IconTile icon={Icon} color={color} size={40} className="rounded-xl" />
        <div>
          <div className="eyebrow">{isAdditive(k) ? 'Total' : 'Current'}</div>
          <div className="num text-[32px] leading-none font-semibold tracking-[-0.04em]">{formatKpi(k)}</div>
        </div>
        <div className="flex items-center gap-2 pb-1">
          <Delta value={k.deltaPct} />
          <span className="text-[12.5px] text-dim">vs previous period</span>
        </div>
      </div>

      {k.series.length > 1 && (
        <div className="rounded-xl border border-line bg-panel-2/40 p-3.5 pb-2.5">
          <div className="mb-1 flex items-center gap-3.5 text-xs text-muted">
            <span className="flex items-center gap-1.5">
              <i className="h-[3px] w-2.5 rounded-xs" style={{ background: tint }} />
              This period
            </span>
            <span className="flex items-center gap-1.5">
              <i className="h-[3px] w-2.5 rounded-xs bg-dim" />
              Previous
            </span>
          </div>
          <AreaChart points={k.series} label={k.label} color={tint} format={fmt} axisFormat={(v) => formatUnit(k.unit, v, true)} gutter={28} onPick={onDay} />
        </div>
      )}

      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
        <Stat label={isAdditive(k) ? 'This period' : 'Period average'} value={fmt(agg(cur))} />
        <Stat label={isAdditive(k) ? 'Previous period' : 'Previous average'} value={fmt(agg(prev))} muted />
        <Stat label="Best day" value={fmt(peak.current)} sub={shortDate(peak.date)} onClick={() => onDay(k.series.indexOf(peak))} />
        <Stat label="Weakest day" value={fmt(low.current)} sub={shortDate(low.date)} onClick={() => onDay(k.series.indexOf(low))} />
      </div>

      <div className="overflow-x-auto rounded-xl border border-line">
        <table className="data-table">
          <thead>
            <tr>
              <th>Day</th>
              <th className="w-[40%] max-sm:hidden" />
              <th className="num">{k.label}</th>
              <th className="num">Previous</th>
              <th className="num">Change</th>
              <th className="w-8 max-sm:hidden" />
            </tr>
          </thead>
          <tbody>
            {[...k.series].reverse().map((p, r) => {
              const ch = change(p.current, p.previous)
              const open = () => onDay(last - r)
              return (
                <tr
                  key={p.date}
                  role="button"
                  tabIndex={0}
                  aria-label={`Open ${rowDay(p.date)}`}
                  onClick={open}
                  onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), open())}
                  className="group cursor-pointer"
                >
                  <td className="font-mono text-muted tabular-nums group-hover:text-fg">{rowDay(p.date)}</td>
                  <td className="max-sm:hidden">
                    <div className="h-1.5 overflow-hidden rounded bg-panel-3">
                      <i className="block h-full rounded" style={{ width: `${(p.current / maxDay) * 100}%`, background: tint }} />
                    </div>
                  </td>
                  <td className="num">{fmt(p.current)}</td>
                  <td className="num text-muted">{fmt(p.previous)}</td>
                  <td className={cn('num', ch >= 0 ? 'text-accent' : 'text-danger')}>
                    {ch >= 0 ? '+' : ''}
                    {ch.toFixed(1)}%
                  </td>
                  <td className="num max-sm:hidden">
                    <ChevronRight className="inline size-3.5 text-dim opacity-0 transition-opacity group-hover:opacity-100" />
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </>
  )
}

/** One day: every metric against the same day of the previous period, plus the orders actually placed. */
function DayDetail({
  dashboard,
  active,
  index,
  onIndex,
  onBack,
}: {
  dashboard: Dashboard
  active: string
  index: number
  onIndex: (i: number) => void
  onBack: () => void
}) {
  const date = dashboard.revenue[index]?.date ?? dashboard.kpis[0].series[index].date
  const last = dashboard.kpis[0].series.length - 1
  const rev = dashboard.revenue[index]
  const metrics: { key: string; label: string; cur: number; prev: number; fmt: (v: number) => string; icon: typeof DollarSign; color: string }[] = [
    ...(rev ? [{ key: 'revenue', label: 'Net revenue', cur: rev.current, prev: rev.previous, fmt: (v: number) => money(v, 2), icon: DollarSign, color: 'var(--color-teal)' }] : []),
    ...dashboard.kpis.map((k) => ({
      key: k.key,
      label: k.label,
      cur: k.series[index].current,
      prev: k.series[index].previous,
      fmt: (v: number) => formatUnit(k.unit, v),
      ...kpiStyle(k.key),
    })),
  ]

  return (
    <>
      <div className="flex items-center gap-2">
        <Button size="sm" onClick={onBack}>
          <ArrowLeft />
          All days
        </Button>
        <div className="ml-auto flex items-center gap-1">
          <Button variant="ghost" size="icon-sm" aria-label="Previous day" disabled={index === 0} onClick={() => onIndex(index - 1)}>
            <ChevronLeft />
          </Button>
          <span className="num min-w-24 text-center text-[12.5px] text-muted">{rowDay(date)}</span>
          <Button variant="ghost" size="icon-sm" aria-label="Next day" disabled={index === last} onClick={() => onIndex(index + 1)}>
            <ChevronRight />
          </Button>
        </div>
      </div>

      <div className="grid gap-2.5 sm:grid-cols-2" data-testid="day-metrics">
        {metrics.map((m) => {
          const ch = change(m.cur, m.prev)
          return (
            <div
              key={m.key}
              className={cn('group card flex items-center gap-3 px-3.5 py-3', m.key === active && 'border-accent/35')}
            >
              <IconTile icon={m.icon} color={m.color} size={34} className="rounded-[10px]" />
              <div className="min-w-0 flex-1">
                <small className="block text-[11.5px] text-dim">{m.label}</small>
                <b className="num text-[17px] font-semibold tracking-[-0.02em]">{m.fmt(m.cur)}</b>
              </div>
              <div className="text-right">
                <Delta value={ch} />
                <small className="num mt-0.5 block text-[11px] text-dim">prev {m.fmt(m.prev)}</small>
              </div>
            </div>
          )
        })}
      </div>

      <DayOrders date={date} />
    </>
  )
}

function DayOrders({ date }: { date: string }) {
  const canOrders = useCan('orders:read')
  const peek = usePeek()
  const from = localDay(date)
  const to = new Date(from.getFullYear(), from.getMonth(), from.getDate() + 1)
  const { data, isPending } = useOrders({ from: from.toISOString(), to: to.toISOString(), limit: 100 }, canOrders)
  if (!canOrders) return null

  const orders = data?.items ?? []
  const billable = orders.filter((o) => o.status !== 'refunded' && o.status !== 'failed')
  const total = billable.reduce((s, o) => s + o.totalCents, 0)
  const items = billable.reduce((s, o) => s + o.items.reduce((n, i) => n + i.qty, 0), 0)

  return (
    <div>
      <div className="mb-2 flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <div className="eyebrow">Orders placed this day</div>
        {data && (
          <small className="num ml-auto text-[11.5px] text-dim">
            {int(data.total)} orders · {money(total, 2)} billable · {int(items)} items
          </small>
        )}
      </div>
      {isPending ? (
        <Skeleton className="h-40" />
      ) : orders.length === 0 ? (
        <p className="rounded-xl border border-dashed border-line-2 py-8 text-center text-[13px] text-dim">No orders in the store for this day.</p>
      ) : (
        <div className="overflow-hidden rounded-xl border border-line">
          <div className="sm:hidden">
            {orders.map((o) => (
              <OrderListItem key={o.id} o={o} onOpen={() => peek('order', o.id)} />
            ))}
          </div>
          {orders.map((o) => (
            <PeekButton
              key={o.id}
              kind="order"
              id={o.id}
              label={`Open order #${o.id}`}
              className="flex w-full items-center gap-3 border-b border-line px-3.5 py-2.5 text-left transition-colors last:border-0 hover:bg-panel-2 max-sm:hidden"
            >
              <span className="num w-14 text-[12.5px] text-muted">#{o.id}</span>
              <Avatar name={o.customer.name} size={24} />
              <span className="min-w-0 flex-1">
                <b className="block truncate text-[13px] font-medium">{o.customer.name}</b>
                <small className="num text-[11px] text-dim">
                  {new Date(o.placedAt).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })} · {o.items.length} items
                </small>
              </span>
              <span className="flex max-sm:hidden">
                {o.items.slice(0, 3).map((it, i) => (
                  <span key={i} className={cn('rounded-[9px] ring-2 ring-panel', i > 0 && '-ml-2')}>
                    <ProductThumb category={it.category} hue={it.hue} src={it.imageUrl} size={26} />
                  </span>
                ))}
              </span>
              <StatusPill status={o.status} />
              <b className="num w-22 text-right text-[13px] font-medium">{money(o.totalCents, 2)}</b>
            </PeekButton>
          ))}
        </div>
      )}
    </div>
  )
}

function Stat({ label, value, sub, muted, onClick }: { label: string; value: string; sub?: string; muted?: boolean; onClick?: () => void }) {
  const body = (
    <>
      <small className="block text-[11.5px] text-dim">{label}</small>
      <b className={cn('num text-[15px] font-medium', muted && 'text-muted')}>{value}</b>
      {sub && <small className="num block text-[11px] text-dim">{sub}</small>}
    </>
  )
  const cls = 'rounded-xl border border-line bg-panel-2/40 px-3 py-2.5 text-left'
  return onClick ? (
    <button onClick={onClick} className={cn(cls, 'transition-colors hover:border-line-2')}>
      {body}
    </button>
  ) : (
    <div className={cls}>{body}</div>
  )
}
