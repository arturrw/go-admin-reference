import { useEffect, useState } from 'react'
import { Card, CardHeader } from '@/components/ui/card'
import { Pill } from '@/components/ui/pill'
import { duration, int } from '@/lib/format'
import { useRuntime } from '@/lib/queries'
import { cn } from '@/lib/utils'

const BARS = 44

/** Animated traffic stream. Bar heights are local noise; the counters come from /api/v1/runtime. */
export function LiveTraffic() {
  const { data } = useRuntime()
  const [bars, setBars] = useState(() => Array.from({ length: BARS }, () => 30 + Math.random() * 60))

  useEffect(() => {
    const t = setInterval(() => {
      setBars((b) => [...b.slice(1), Math.min(100, Math.max(12, b[b.length - 1] + (Math.random() - 0.5) * 34))])
    }, 1100)
    return () => clearInterval(t)
  }, [])

  return (
    <Card className="col-span-full flex flex-col gap-1.5 p-4">
      <div className="flex items-center gap-2 text-[12.5px] text-muted">
        <span className="live-dot" />
        Live now
        <span className="num ml-auto text-[11.5px] text-dim">req/s</span>
      </div>
      <div className="flex items-baseline gap-2.5">
        <div className="num mt-1 text-2xl font-semibold tracking-[-0.03em]">{data ? int(data.requestsPerSec) : '—'}</div>
        <span className="text-[12.5px] text-dim">
          <b className="num font-medium text-fg">{data?.onlineUsers ?? '—'}</b> shoppers online
        </span>
      </div>
      <div className="mt-3.5 flex h-18 items-end gap-[3px]">
        {bars.map((v, i) => (
          <i
            key={i}
            className={cn(
              'min-h-[3px] flex-1 rounded-t-[3px] rounded-b-[1px] transition-[height] duration-400 ease-[cubic-bezier(.2,.8,.2,1)]',
              v > 94
                ? 'bg-linear-to-b from-danger to-danger/20'
                : 'bg-linear-to-b from-accent to-accent/25',
            )}
            style={{ height: `${v}%` }}
          />
        ))}
      </div>
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
