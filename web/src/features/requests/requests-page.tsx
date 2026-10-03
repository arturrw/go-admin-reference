import { Download, Pause, Terminal } from 'lucide-react'
import { useDeferredValue, useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PageHeader, Skeleton, StatStrip } from '@/components/ui/misc'
import { Segmented } from '@/components/ui/segmented'
import type { RequestEntry } from '@/lib/api'
import { int } from '@/lib/format'
import { useRequests } from '@/lib/queries'
import { cn } from '@/lib/utils'

const METHOD_COLOR: Record<string, string> = {
  GET: 'text-info',
  POST: 'text-accent',
  PUT: 'text-warn',
  PATCH: 'text-violet',
  DELETE: 'text-danger',
}
const STATUS_COLOR = ['', '', 'text-accent', 'text-info', 'text-warn', 'text-danger']
const COLS = 'grid grid-cols-[96px_64px_minmax(0,1fr)_56px_120px_100px] gap-3 px-4.5 py-1.5 max-md:grid-cols-[64px_52px_minmax(0,1fr)_40px] max-md:px-3.5'

export function RequestsPage() {
  const [live, setLive] = useState(true)
  const [cls, setCls] = useState('all')
  const [q, setQ] = useState('')
  const { data, isPending } = useRequests({ class: cls, q: useDeferredValue(q), limit: 150 }, live)
  const st = data?.stats
  const seen = useRef(new Set<string>())

  // Rows that appear after the first render flash briefly.
  const fresh = new Set(data?.items.filter((e) => !seen.current.has(e.requestId)).map((e) => e.requestId))
  const firstLoad = seen.current.size === 0
  useEffect(() => {
    data?.items.forEach((e) => seen.current.add(e.requestId))
  }, [data])

  return (
    <>
      <PageHeader title="Request log" description="Every /api call handled by this Go process, newest first.">
        <Button onClick={() => setLive(!live)}>
          {live ? <span className="live-dot" /> : <Pause />}
          {live ? 'Live' : 'Paused'}
        </Button>
        <Button>
          <Download />
          Download
        </Button>
      </PageHeader>

      {st ? (
        <StatStrip
          items={[
            { label: 'Requests (buffer)', value: int(st.total) },
            { label: 'Success rate', value: `${st.successRate.toFixed(1)}%` },
            { label: 'p95 latency', value: `${st.p95Ms.toFixed(0)}ms` },
            { label: '4xx', value: st.client4xx, tone: 'var(--color-warn)' },
            { label: '5xx', value: st.server5xx, tone: 'var(--color-danger)' },
          ]}
        />
      ) : (
        <Skeleton className="mb-4 h-19" />
      )}

      <div className="mb-3.5 flex flex-wrap items-center gap-2.5">
        <label className="relative block min-w-55 flex-1 sm:max-w-115">
          <Terminal className="pointer-events-none absolute top-2.5 left-3 size-4 text-dim" />
          <Input className="num pl-9 text-[12.5px]" placeholder="filter: path, method, status…" value={q} onChange={(e) => setQ(e.target.value)} />
        </label>
        <Segmented
          value={cls}
          onChange={setCls}
          options={[
            { value: 'all', label: 'All' },
            { value: '2', label: '2xx' },
            { value: '4', label: '4xx' },
            { value: '5', label: '5xx' },
          ]}
        />
      </div>

      <div className="card overflow-hidden p-0">
        <div className="num text-[12.5px] leading-relaxed">
          <div className={cn(COLS, 'border-b border-line bg-white/[.012] text-[10.5px] tracking-widest text-dim uppercase')}>
            <span>Time</span>
            <span>Method</span>
            <span>Path</span>
            <span>Status</span>
            <span className="max-md:hidden">Latency</span>
            <span className="max-md:hidden">Request ID</span>
          </div>
          <div className="max-h-[calc(100vh-290px)] overflow-y-auto">
            {isPending && <Skeleton className="m-4 h-80" />}
            {data?.items.map((e) => <Row key={e.requestId + e.time} e={e} flash={!firstLoad && fresh.has(e.requestId)} />)}
            {data && data.items.length === 0 && <div className="py-14 text-center font-sans text-muted">No matching requests.</div>}
          </div>
        </div>
      </div>
    </>
  )
}

function Row({ e, flash }: { e: RequestEntry; flash: boolean }) {
  const t = new Date(e.time)
  const time = t.toTimeString().slice(0, 8) + '.' + String(t.getMilliseconds()).padStart(3, '0')
  const slow = e.durationMs > 800 ? 'bg-danger' : e.durationMs > 200 ? 'bg-warn' : 'bg-muted/60'
  return (
    <div className={cn(COLS, 'items-center border-b border-line hover:bg-white/2', flash && 'animate-flash')}>
      <span className="text-dim">{time}</span>
      <span className={cn('font-semibold', METHOD_COLOR[e.method])}>{e.method}</span>
      <span className="truncate" title={e.path}>
        {e.path}
      </span>
      <span className={STATUS_COLOR[Math.floor(e.status / 100)]}>{e.status}</span>
      <span className="flex items-center gap-2 max-md:hidden">
        <i className={cn('h-1 rounded-xs', slow)} style={{ width: Math.max(2, Math.min(60, e.durationMs / 12)) }} />
        {e.durationMs < 1 ? e.durationMs.toFixed(2) : Math.round(e.durationMs)}ms
      </span>
      <span className="text-dim max-md:hidden">{e.requestId}</span>
    </div>
  )
}
