import { ChevronRight, Flame } from 'lucide-react'
import { Card, CardHeader } from '@/components/ui/card'
import { Pill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { duration, int } from '@/lib/format'
import { useLive, useRuntime } from '@/lib/queries'
import { cn } from '@/lib/utils'

const BARS = 44

/**
 * Live storefront traffic. Bars are requests/s over the last two minutes from
 * /api/v1/live; clicking opens the detailed view.
 */
export function LiveTraffic({ onOpen }: { onOpen: () => void }) {
  const { data } = useLive()
  const history = data?.history ?? []
  const rps = history.map((h) => h.requestsPerSec)
  const lo = Math.min(...rps) * 0.85
  const hi = Math.max(...rps)
  const bars = history.slice(-BARS).map((h) => 12 + ((h.requestsPerSec - lo) / (hi - lo || 1)) * 88)
  const hot = data?.hot

  return (
    <Card
      role="button"
      tabIndex={0}
      aria-label="Open live traffic details"
      onClick={onOpen}
      onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), onOpen())}
      className="col-span-full flex cursor-pointer flex-col gap-1.5 p-4 transition-colors hover:border-line-2"
    >
      <div className="flex items-center gap-2 text-[12.5px] text-muted">
        <span className="live-dot" />
        Live now
        <span className="num ml-auto flex items-center gap-1 text-[11.5px] text-dim">
          req/s <ChevronRight className="size-3.5" />
        </span>
      </div>
      <div className="flex items-baseline gap-2.5">
        <div className="num mt-1 text-2xl font-semibold tracking-[-0.03em]">{data ? int(data.requestsPerSec) : '—'}</div>
        <span className="text-[12.5px] text-dim">
          <b className="num font-medium text-fg">{data?.onlineUsers ?? '—'}</b> shoppers online
        </span>
      </div>
      <div className="mt-2.5 flex h-16 items-end gap-[3px]">
        {(bars.length ? bars : Array.from({ length: BARS }, () => 12)).map((v, i) => (
          <i
            key={i}
            className={cn(
              'min-h-[3px] flex-1 rounded-t-[3px] rounded-b-[1px] transition-[height] duration-400 ease-[cubic-bezier(.2,.8,.2,1)]',
              v > 94 ? 'bg-linear-to-b from-danger to-danger/20' : 'bg-linear-to-b from-accent to-accent/25',
            )}
            style={{ height: `${v}%` }}
          />
        ))}
      </div>
      {hot && (
        <div className="mt-2 flex items-center gap-2.5 rounded-[10px] border border-line bg-panel-2/50 px-2.5 py-2">
          <ProductThumb category={hot.category} hue={hot.hue} src={hot.imageUrl} size={30} />
          <div className="min-w-0 flex-1">
            <small className="flex items-center gap-1 text-[11px] text-dim">
              <Flame className="size-3 text-warn" /> Hottest product
            </small>
            <b className="block truncate text-[12.5px] font-medium">{hot.name}</b>
          </div>
          <div className="text-right text-[11.5px] text-dim">
            <b className="num block text-[13px] font-medium text-fg">{hot.viewers}</b>
            viewing
          </div>
        </div>
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
