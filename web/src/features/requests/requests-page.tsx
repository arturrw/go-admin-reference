import { Download, Pause, Terminal } from 'lucide-react'
import { useDeferredValue, useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardHeader } from '@/components/ui/card'
import { Input, Select } from '@/components/ui/input'
import { PageHeader, Skeleton, StatStrip } from '@/components/ui/misc'
import { Segmented } from '@/components/ui/segmented'
import type { RequestStats, RequestSummary } from '@/lib/api'
import { int } from '@/lib/format'
import { useRequests } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { RequestSheet } from './request-sheet'

export const METHOD_COLOR: Record<string, string> = {
  GET: 'text-info',
  POST: 'text-accent',
  PUT: 'text-warn',
  PATCH: 'text-violet',
  DELETE: 'text-danger',
}
export const STATUS_COLOR = ['', '', 'text-accent', 'text-info', 'text-warn', 'text-danger']
const COLS =
  'grid grid-cols-[96px_64px_minmax(0,1fr)_56px_110px_150px] gap-3 px-4.5 py-1.5 max-lg:grid-cols-[96px_64px_minmax(0,1fr)_56px_110px] max-md:grid-cols-[64px_52px_minmax(0,1fr)_40px] max-md:px-3.5'

export function RequestsPage() {
  const [live, setLive] = useState(true)
  const [cls, setCls] = useState('all')
  const [method, setMethod] = useState('')
  const [q, setQ] = useState('')
  const [openId, setOpenId] = useState<string>()
  const { data, isPending } = useRequests({ class: cls, method, q: useDeferredValue(q), limit: 150 }, live && !openId)
  const st = data?.stats
  const seen = useRef(new Set<string>())

  // Rows that appear after the first render flash briefly.
  const firstLoad = seen.current.size === 0
  const fresh = new Set(data?.items.filter((e) => !seen.current.has(e.id)).map((e) => e.id))
  useEffect(() => {
    data?.items.forEach((e) => seen.current.add(e.id))
  }, [data])

  return (
    <>
      <PageHeader title="Request log" description="Every /api call handled by this Go process. Click a row for headers, bodies and timing.">
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
            {
              label: 'Latency p95',
              value: (
                <>
                  {fmtMs(st.p95Ms)}{' '}
                  <small className="text-xs font-normal tracking-normal text-dim">
                    p50 {fmtMs(st.p50Ms)} · p99 {fmtMs(st.p99Ms)}
                  </small>
                </>
              ),
            },
            { label: '4xx', value: st.client4xx, tone: 'var(--color-warn)' },
            { label: '5xx', value: st.server5xx, tone: 'var(--color-danger)' },
          ]}
        />
      ) : (
        <Skeleton className="mb-4 h-19" />
      )}

      {st && <Analytics stats={st} onRoute={(route) => setQ(route.split(' ')[1].replace('{id}', ''))} />}

      <div className="mb-3.5 flex flex-wrap items-center gap-2.5">
        <label className="relative block min-w-55 flex-1 sm:max-w-115">
          <Terminal className="pointer-events-none absolute top-2.5 left-3 size-4 text-dim" />
          <Input className="num pl-9 text-[12.5px]" placeholder="filter: path, status, user…" value={q} onChange={(e) => setQ(e.target.value)} />
        </label>
        <Select className="w-32.5" value={method} onChange={(e) => setMethod(e.target.value)} aria-label="Method">
          <option value="">All methods</option>
          {Object.keys(METHOD_COLOR).map((m) => (
            <option key={m}>{m}</option>
          ))}
        </Select>
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
            <span className="max-lg:hidden">User</span>
          </div>
          <div className="max-h-[calc(100vh-290px)] overflow-y-auto" data-testid="request-rows">
            {isPending && <Skeleton className="m-4 h-80" />}
            {data?.items.map((e) => <Row key={e.id + e.time} e={e} flash={!firstLoad && fresh.has(e.id)} onOpen={() => setOpenId(e.id)} />)}
            {data && data.items.length === 0 && <div className="py-14 text-center font-sans text-muted">No matching requests.</div>}
          </div>
        </div>
      </div>

      {openId && <RequestSheet id={openId} onClose={() => setOpenId(undefined)} />}
    </>
  )
}

export const fmtMs = (ms: number) => (ms < 1 ? `${Math.round(ms * 1000)}µs` : ms < 1000 ? `${Math.round(ms)}ms` : `${(ms / 1000).toFixed(2)}s`)

function Analytics({ stats, onRoute }: { stats: RequestStats; onRoute: (route: string) => void }) {
  const maxMin = Math.max(1, ...stats.perMinute)
  const maxEp = Math.max(1, ...stats.endpoints.map((e) => e.count))
  return (
    <div className="mb-4 grid gap-3.5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)]">
      <Card>
        <CardHeader title="Traffic" sub="requests per minute, last 30 min" />
        <div className="flex h-24 items-end gap-[3px]">
          {stats.perMinute.map((n, i) => (
            <i
              key={i}
              title={`${29 - i} min ago — ${n} requests`}
              className="min-h-0.5 flex-1 rounded-t-[3px] bg-linear-to-b from-accent to-accent/25"
              style={{ height: `${(n / maxMin) * 100}%` }}
            />
          ))}
        </div>
        <div className="num mt-1.5 flex justify-between text-[10.5px] text-dim">
          <span>−30m</span>
          <span>now</span>
        </div>
        <div className="mt-3.5 flex flex-wrap gap-3 text-xs">
          {Object.entries(stats.methods)
            .sort((a, b) => b[1] - a[1])
            .map(([m, n]) => (
              <span key={m} className="num">
                <b className={cn('font-semibold', METHOD_COLOR[m])}>{m}</b> <span className="text-muted">{n}</span>
              </span>
            ))}
        </div>
      </Card>
      <Card className="pb-2">
        <CardHeader title="Top endpoints" sub="by volume" />
        <div className="num text-[12px]">
          <div className="grid grid-cols-[minmax(0,1fr)_56px_64px_64px_48px] gap-2 pb-1.5 text-[10px] tracking-widest text-dim uppercase">
            <span>Route</span>
            <span className="text-right">Calls</span>
            <span className="text-right">Avg</span>
            <span className="text-right">p95</span>
            <span className="text-right">Err</span>
          </div>
          {stats.endpoints.map((e) => (
            <button
              key={e.route}
              onClick={() => onRoute(e.route)}
              className="relative grid w-full grid-cols-[minmax(0,1fr)_56px_64px_64px_48px] items-center gap-2 rounded-md py-1 text-left hover:bg-white/3"
            >
              <i className="absolute inset-y-0.5 left-0 rounded-md bg-accent/7" style={{ width: `${(e.count / maxEp) * 100}%` }} />
              <span className="relative truncate pl-1.5">
                <b className={cn('font-semibold', METHOD_COLOR[e.route.split(' ')[0]])}>{e.route.split(' ')[0]}</b> {e.route.split(' ')[1]}
              </span>
              <span className="relative text-right">{e.count}</span>
              <span className="relative text-right text-muted">{fmtMs(e.avgMs)}</span>
              <span className="relative text-right text-muted">{fmtMs(e.p95Ms)}</span>
              <span className={cn('relative pr-1.5 text-right', e.errors ? 'text-warn' : 'text-dim')}>{e.errors}</span>
            </button>
          ))}
        </div>
      </Card>
    </div>
  )
}

function Row({ e, flash, onOpen }: { e: RequestSummary; flash: boolean; onOpen: () => void }) {
  const t = new Date(e.time)
  const time = t.toTimeString().slice(0, 8) + '.' + String(t.getMilliseconds()).padStart(3, '0')
  const slow = e.durationMs > 800 ? 'bg-danger' : e.durationMs > 200 ? 'bg-warn' : 'bg-muted/60'
  return (
    <button onClick={onOpen} className={cn(COLS, 'w-full items-center border-b border-line text-left hover:bg-white/3', flash && 'animate-flash')}>
      <span className="text-dim">{time}</span>
      <span className={cn('font-semibold', METHOD_COLOR[e.method])}>{e.method}</span>
      <span className="truncate" title={e.path}>
        {e.path}
      </span>
      <span className={STATUS_COLOR[Math.floor(e.status / 100)]}>{e.status}</span>
      <span className="flex items-center gap-2 max-md:hidden">
        <i className={cn('h-1 rounded-xs', slow)} style={{ width: Math.max(2, Math.min(50, e.durationMs / 12)) }} />
        {fmtMs(e.durationMs)}
      </span>
      <span className="truncate text-dim max-lg:hidden">{e.actor || '—'}</span>
    </button>
  )
}
