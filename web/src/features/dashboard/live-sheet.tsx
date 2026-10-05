import { Link } from '@tanstack/react-router'
import { ArrowRight, Flame, Star } from 'lucide-react'
import { useState } from 'react'
import { AreaChart } from '@/components/charts/area-chart'
import { SERIES_COLORS } from '@/components/charts/donut'
import { Sparkline } from '@/components/charts/sparkline'
import { buttonVariants } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/misc'
import { Pill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Segmented } from '@/components/ui/segmented'
import { Sheet } from '@/components/ui/sheet'
import type { LiveStats } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { int, money } from '@/lib/format'
import { useLive } from '@/lib/queries'
import { cn } from '@/lib/utils'

const LOW_STOCK = 15

type Metric = 'onlineUsers' | 'requestsPerSec' | 'hotViewers'
const METRICS: { value: Metric; label: string; color: string }[] = [
  { value: 'onlineUsers', label: 'Shoppers', color: 'var(--color-info)' },
  { value: 'requestsPerSec', label: 'Requests/s', color: 'var(--color-accent)' },
  { value: 'hotViewers', label: 'Hot product', color: 'var(--color-warn)' },
]

const clock = (iso: string) => new Date(iso).toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })

/** Detailed live view: storefront counters, a two-minute chart and the best seller's live numbers. */
export function LiveSheet({ onClose }: { onClose: () => void }) {
  const { data } = useLive()
  const [metric, setMetric] = useState<Metric>('onlineUsers')
  const m = METRICS.find((x) => x.value === metric)!

  return (
    <Sheet
      open
      onOpenChange={(v) => !v && onClose()}
      size="lg"
      title={
        <span className="flex items-center gap-2">
          <span className="live-dot" /> Live storefront
        </span>
      }
      description="Updates every 2 seconds · simulated visitors, real catalogue data"
    >
      {!data ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-24" />
          <Skeleton className="h-64" />
          <Skeleton className="h-48" />
        </div>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
            <Stat label="Requests / s" value={int(data.requestsPerSec)} />
            <Stat label="Shoppers online" value={int(data.onlineUsers)} />
            <Stat label="Active carts" value={int(data.activeCarts)} />
            <Stat label="Checkouts / min" value={data.checkoutsPerMin.toFixed(1)} />
            <Stat label="Conversion" value={`${data.conversionPct.toFixed(2)}%`} />
            <Stat label="Avg. session" value={`${Math.floor(data.avgSessionSec / 60)}m ${String(data.avgSessionSec % 60).padStart(2, '0')}s`} />
            <Stat label="Bounce rate" value={`${data.bounceRatePct.toFixed(1)}%`} />
            <Stat label="Carts / shopper" value={`${((data.activeCarts / Math.max(data.onlineUsers, 1)) * 100).toFixed(1)}%`} />
          </div>

          <div className="rounded-xl border border-line bg-panel-2/40 p-3.5 pb-2.5">
            <div className="flex flex-wrap items-center gap-2.5">
              <div className="eyebrow">Last 2 minutes</div>
              <Segmented className="ml-auto" value={metric} onChange={setMetric} options={METRICS.map(({ value, label }) => ({ value, label }))} />
            </div>
            <AreaChart
              points={data.history.map((h) => ({ date: h.at, current: h[metric] }))}
              label={m.label}
              color={m.color}
              format={int}
              axisFormat={int}
              xFormat={clock}
              tipDate={clock}
              gutter={28}
            />
          </div>

          {data.hot && <HotProduct live={data} />}

          <div className="grid gap-3.5 sm:grid-cols-2">
            <Shares title="Devices" items={data.devices} />
            <Shares title="Traffic sources" items={data.sources} />
          </div>

          <div>
            <div className="eyebrow mb-2">Top pages right now</div>
            <div className="overflow-hidden rounded-xl border border-line">
              {data.topPages.map((p, i) => (
                <div key={p.path} className="flex items-center gap-3 border-b border-line px-3.5 py-2 last:border-0">
                  <span className="num w-4 text-[11px] text-dim">{i + 1}</span>
                  <code className="num min-w-0 flex-1 truncate text-[12.5px] text-muted">{p.path}</code>
                  <b className="num text-[13px] font-medium">{p.viewers}</b>
                  <small className="text-[11.5px] text-dim">viewing</small>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </Sheet>
  )
}

function HotProduct({ live }: { live: LiveStats }) {
  const p = live.hot!
  const canOpen = useCan('products:read')
  const lowStock = p.stock < LOW_STOCK
  return (
    <div className="rounded-xl border border-warn/25 bg-warn/5 p-3.5">
      <div className="mb-3 flex items-center gap-2 text-[12.5px] text-muted">
        <Flame className="size-4 text-warn" />
        Hottest product right now
        <span className="ml-auto text-[11.5px] text-dim">best seller · last 30 days</span>
      </div>
      <div className="flex flex-wrap items-center gap-3.5">
        <ProductThumb category={p.category} hue={p.hue} src={p.imageUrl} alt={p.name} size={64} className="rounded-xl" />
        <div className="min-w-0 flex-1">
          <b className="block truncate text-[15px] font-semibold">{p.name}</b>
          <div className="num mt-0.5 flex flex-wrap items-center gap-2 text-[12px] text-dim">
            <span>{p.sku}</span>·<span className="text-fg">{money(p.priceCents, 2)}</span>·
            <span className="flex items-center gap-0.5">
              <Star className="size-3 fill-warn text-warn" />
              {p.rating.toFixed(1)}
            </span>
          </div>
        </div>
        {canOpen && (
          <Link to="/products" search={{ edit: p.id }} className={buttonVariants({ size: 'sm' })}>
            Open product <ArrowRight />
          </Link>
        )}
      </div>

      <div className="mt-3.5 grid grid-cols-2 gap-2.5 sm:grid-cols-4">
        <Stat label="Viewing now" value={int(p.viewers)} accent />
        <Stat label="In carts" value={int(p.inCarts)} />
        <Stat label="Sold today" value={int(p.soldToday)} />
        <Stat label="Revenue today" value={money(p.revenueTodayCents)} />
        <Stat label="Cart → order" value={`${p.conversionPct.toFixed(1)}%`} />
        <Stat label="Share of traffic" value={`${p.shareOfTrafficPct.toFixed(1)}%`} />
        <Stat label="Sold · 30 days" value={int(p.sold30d)} />
        <div className="rounded-xl border border-line bg-panel px-3 py-2.5">
          <small className="block text-[11.5px] text-dim">Stock left</small>
          <div className="flex items-center gap-1.5">
            <b className={cn('num text-[15px] font-medium', lowStock && 'text-warn')}>{int(p.stock)}</b>
            {lowStock && <Pill tone="amber">low</Pill>}
          </div>
        </div>
      </div>

      <div className="mt-3 flex items-center gap-3">
        <small className="text-[11.5px] text-dim">Viewers · 2 min</small>
        <Sparkline data={live.history.map((h) => h.hotViewers)} color="var(--color-warn)" className="h-8 flex-1" />
      </div>
    </div>
  )
}

function Shares({ title, items }: { title: string; items: { name: string; pct: number }[] }) {
  return (
    <div className="rounded-xl border border-line p-3.5">
      <div className="eyebrow mb-2.5">{title}</div>
      <div className="flex h-2 overflow-hidden rounded bg-panel-3">
        {items.map((s, i) => (
          <i key={s.name} className="block h-full" style={{ width: `${s.pct}%`, background: SERIES_COLORS[i % SERIES_COLORS.length] }} />
        ))}
      </div>
      <div className="mt-2.5 flex flex-col gap-1.5 text-[12.5px]">
        {items.map((s, i) => (
          <div key={s.name} className="flex items-center gap-2">
            <i className="size-2 rounded-[2px]" style={{ background: SERIES_COLORS[i % SERIES_COLORS.length] }} />
            <span className="flex-1 text-muted">{s.name}</span>
            <b className="num text-xs font-medium">{s.pct.toFixed(1)}%</b>
          </div>
        ))}
      </div>
    </div>
  )
}

function Stat({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div className="rounded-xl border border-line bg-panel px-3 py-2.5">
      <small className="block text-[11.5px] text-dim">{label}</small>
      <b className={cn('num text-[15px] font-medium', accent && 'text-accent')}>{value}</b>
    </div>
  )
}
