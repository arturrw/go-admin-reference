import { ChevronRight, Flame } from 'lucide-react'
import { Bars } from '@/components/charts/bars'
import { Card, CardHeader } from '@/components/ui/card'
import { Pill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { useCan } from '@/lib/auth'
import { duration, int } from '@/lib/format'
import { PeekButton } from '@/lib/peek'
import { useLive, useRuntime } from '@/lib/queries'
import { cn } from '@/lib/utils'

const BARS = 44

const clock = (iso: string) => new Date(iso).toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })

/**
 * Live storefront traffic. Bars are requests/s over the last two minutes from
 * /api/v1/live; the traffic part opens the detailed view. The hottest product
 * is its own target: hovering highlights just it, clicking opens it.
 */
export function LiveTraffic({ onOpen }: { onOpen: () => void }) {
  const { data } = useLive()
  const canProducts = useCan('products:read')
  const history = data?.history ?? []
  const rps = history.map((h) => h.requestsPerSec)
  const lo = Math.min(...rps) * 0.85
  const hi = Math.max(...rps)
  const shown = history.slice(-BARS)
  const bars = shown.map((h) => 12 + ((h.requestsPerSec - lo) / (hi - lo || 1)) * 88)
  const hot = data?.hot

  return (
    <Card className="col-span-full flex flex-col gap-2 p-4 transition-colors has-[[data-live-open]:hover]:border-line-2">
      <div
        role="button"
        tabIndex={0}
        aria-label="Open live traffic details"
        data-live-open
        onClick={onOpen}
        onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), onOpen())}
        className="group flex cursor-pointer flex-col gap-1.5 rounded-lg"
      >
        <div className="flex items-center gap-2 text-[12.5px] text-muted">
          <span className="live-dot" />
          Live now
          <span className="num ml-auto flex items-center gap-1 text-[11.5px] text-dim transition-colors group-hover:text-fg">
            req/s <ChevronRight className="size-3.5" />
          </span>
        </div>
        <div className="flex items-baseline gap-2.5">
          <div className="num mt-1 text-2xl font-semibold tracking-[-0.03em]">{data ? int(data.requestsPerSec) : '—'}</div>
          <span className="text-[12.5px] text-dim">
            <b className="num font-medium text-fg">{data?.onlineUsers ?? '—'}</b> shoppers online
          </span>
        </div>
        <Bars
          className="mt-2.5 h-16"
          label="Requests per second, last two minutes"
          heights={bars.length ? bars : Array.from({ length: BARS }, () => 12)}
          tone={(i) => ((bars[i] ?? 0) > 94 ? 'bg-linear-to-b from-danger to-danger/20' : 'bg-linear-to-b from-accent to-accent/25')}
          tip={(i) =>
            shown[i] ? (
              <>
                <span className="text-dim">{clock(shown[i].at)}</span> · <b className="font-medium">{int(shown[i].requestsPerSec)}</b> req/s ·{' '}
                <b className="font-medium">{shown[i].onlineUsers}</b> online
              </>
            ) : (
              'waiting for data'
            )
          }
        />
      </div>
      {hot && (
        <PeekButton
          kind="product"
          id={hot.id}
          disabled={!canProducts}
          label={`Hottest product: ${hot.name}, open product`}
          data-testid="hot-product"
          className={cn(
            'group/hot flex w-full items-center gap-2.5 rounded-[10px] text-left border border-line bg-panel-2/50 px-2.5 py-2 transition-[border-color,background-color,box-shadow]',
            canProducts && 'hover:border-warn/45 hover:bg-warn/7 hover:shadow-[0_0_0_3px_color-mix(in_srgb,var(--color-warn)_12%,transparent)]',
          )}
        >
          <ProductThumb category={hot.category} hue={hot.hue} src={hot.imageUrl} size={30} />
          <div className="min-w-0 flex-1">
            <small className="flex items-center gap-1 text-[11px] text-dim">
              <Flame className="size-3 text-warn transition-transform group-hover/hot:scale-125" /> Hottest product
            </small>
            <b key={hot.id} className="block animate-fade-in truncate text-[12.5px] font-medium group-hover/hot:text-warn">
              {hot.name}
            </b>
          </div>
          <div className="text-right text-[11.5px] text-dim">
            <b className="num block text-[13px] font-medium text-fg">{hot.viewers}</b>
            viewing
          </div>
          {canProducts && <ChevronRight className="size-3.5 text-dim opacity-0 transition-opacity group-hover/hot:opacity-100" />}
        </PeekButton>
      )}
    </Card>
  )
}

/** Real numbers from the running Go process. */
export function GoRuntimeCard({ className }: { className?: string }) {
  const { data: rt } = useRuntime()
  const heapPct = rt ? Math.min(100, (rt.heapAllocMb / Math.max(rt.heapSysMb, 1)) * 100) : 0
  const cells: [string, string, number?][] = rt
    ? [
        ['Goroutines', int(rt.goroutines)],
        ['GC pause max', `${rt.gcPauseMaxMs.toFixed(2)}ms`],
        ['Heap in use', `${rt.heapAllocMb} / ${rt.heapSysMb} MB`, heapPct],
        ['GC cycles', int(rt.numGc)],
        ['GOMAXPROCS', String(rt.gomaxprocs)],
        ['Uptime', duration(rt.uptimeSeconds)],
      ]
    : []

  return (
    <Card className={className}>
      <CardHeader title="Go runtime">
        <Pill tone="lime">healthy</Pill>
      </CardHeader>
      <div className="grid grid-cols-2 gap-px overflow-hidden rounded-[10px] border border-line bg-line">
        {cells.map(([label, value, meter]) => (
          <div key={label} className="bg-panel px-3 py-2.5">
            <small className="block text-[11.5px] text-dim">{label}</small>
            <b className="num text-[15px] font-medium">{value}</b>
            {meter !== undefined && (
              <div className="mt-1.5 h-1.5 overflow-hidden rounded-md bg-panel-3">
                <i className="block h-full rounded-md bg-accent transition-[width]" style={{ width: `${meter}%` }} />
              </div>
            )}
          </div>
        ))}
      </div>
      {rt && (
        <div className="num mt-3 flex flex-wrap gap-2 text-[11.5px] text-dim">
          <span>{rt.goVersion}</span>·<span>{rt.platform}</span>·<span>rev {rt.revision}</span>
        </div>
      )}
    </Card>
  )
}
