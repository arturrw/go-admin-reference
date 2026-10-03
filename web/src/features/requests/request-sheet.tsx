import { Copy } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { Sheet } from '@/components/ui/sheet'
import { Tabs } from '@/components/ui/tabs'
import type { RequestEntry } from '@/lib/api'
import { useRequestEntry } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { fmtMs, METHOD_COLOR, STATUS_COLOR } from './requests-page'

type Tab = 'overview' | 'request' | 'response'

export function RequestSheet({ id, onClose }: { id: string; onClose: () => void }) {
  const { data: e, isPending, error } = useRequestEntry(id)
  const [tab, setTab] = useState<Tab>('overview')

  return (
    <Sheet open onOpenChange={(v) => !v && onClose()} size="lg" title="Request detail" description={`ID ${id}`}>
      {error ? (
        <p className="text-danger">{error.message}</p>
      ) : isPending || !e ? (
        <Skeleton className="h-96" />
      ) : (
        <>
          <div className="card num flex flex-wrap items-center gap-x-3 gap-y-1.5 p-3.5 text-[13px]">
            <b className={cn('font-semibold', METHOD_COLOR[e.method])}>{e.method}</b>
            <span className="min-w-0 flex-1 break-all">{e.path}</span>
            <b className={cn('font-semibold', STATUS_COLOR[Math.floor(e.status / 100)])}>{e.status}</b>
            <Button size="icon-sm" variant="ghost" aria-label="Copy as curl" title="Copy as curl" onClick={() => copyCurl(e)}>
              <Copy />
            </Button>
          </div>

          <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
            <Metric label="Duration" value={fmtMs(e.durationMs)} />
            <Metric label="Response size" value={bytes(e.bytes)} />
            <Metric label="Request size" value={e.requestBytes > 0 ? bytes(e.requestBytes) : '—'} />
            <Metric label="Protocol" value={e.proto || '—'} />
          </div>

          <Tabs<Tab>
            value={tab}
            onChange={setTab}
            tabs={[
              { value: 'overview', label: 'Overview' },
              { value: 'request', label: 'Request', count: Object.keys(e.requestHeaders ?? {}).length },
              { value: 'response', label: 'Response', count: Object.keys(e.responseHeaders ?? {}).length },
            ]}
          />

          {tab === 'overview' && (
            <dl className="grid grid-cols-[140px_minmax(0,1fr)] gap-x-4 gap-y-2.5 text-[13px] [&_dd]:break-all [&_dt]:text-dim">
              <dt>Time</dt>
              <dd className="num">{new Date(e.time).toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'medium' })}</dd>
              <dt>Route</dt>
              <dd className="num">{e.route || '—'}</dd>
              <dt>Query</dt>
              <dd className="num">{e.query ? <QueryParams query={e.query} /> : '—'}</dd>
              <dt>User</dt>
              <dd>
                {e.actor ? (
                  <span className="inline-flex items-center gap-2">
                    {e.actor} {e.actorRole && <StatusPill status={e.actorRole} />}
                  </span>
                ) : (
                  <span className="text-dim">anonymous</span>
                )}
              </dd>
              <dt>Client IP</dt>
              <dd className="num">{e.ip}</dd>
              <dt>User agent</dt>
              <dd className="text-muted">{e.userAgent || '—'}</dd>
              <dt>Request ID</dt>
              <dd className="num">{e.id}</dd>
            </dl>
          )}

          {tab === 'request' && (
            <>
              <Headers h={e.requestHeaders} />
              <Body title="Body" body={e.requestBody} truncated={e.bodyTruncated} empty="No request body" />
            </>
          )}

          {tab === 'response' && (
            <>
              <Headers h={e.responseHeaders} />
              <Body title="Body" body={e.responseBody} empty={e.status >= 400 ? 'Empty' : 'Bodies are only captured for error responses (4xx/5xx).'} />
            </>
          )}

          <p className="text-xs text-dim">Cookie, Authorization and password/token fields are redacted before they’re stored.</p>
        </>
      )}
    </Sheet>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="card px-3 py-2.5">
      <div className="eyebrow">{label}</div>
      <div className="num mt-0.5 text-[15px]">{value}</div>
    </div>
  )
}

function QueryParams({ query }: { query: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      {[...new URLSearchParams(query)].map(([k, v], i) => (
        <span key={i}>
          <span className="text-dim">{k}=</span>
          {v}
        </span>
      ))}
    </div>
  )
}

function Headers({ h }: { h: Record<string, string> | null }) {
  const entries = Object.entries(h ?? {}).sort(([a], [b]) => a.localeCompare(b))
  return (
    <Block title="Headers">
      {entries.length === 0 ? (
        <span className="text-dim">None</span>
      ) : (
        <div className="grid grid-cols-[minmax(120px,auto)_minmax(0,1fr)] gap-x-4 gap-y-1">
          {entries.map(([k, v]) => (
            <div key={k} className="contents">
              <span className="text-dim">{k}</span>
              <span className={cn('break-all', v === '[redacted]' && 'text-warn')}>{v}</span>
            </div>
          ))}
        </div>
      )}
    </Block>
  )
}

function Body({ title, body, truncated, empty }: { title: string; body: string; truncated?: boolean; empty: string }) {
  let pretty = body
  try {
    pretty = JSON.stringify(JSON.parse(body), null, 2)
  } catch {
    // not JSON — show as-is
  }
  return (
    <Block title={title + (truncated ? ' (truncated to 4 KB)' : '')}>
      {body ? <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-all">{pretty}</pre> : <span className="font-sans text-dim">{empty}</span>}
    </Block>
  )
}

function Block({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <div className="eyebrow mb-1.5">{title}</div>
      <div className="num rounded-xl border border-line bg-bg/60 p-3.5 text-[12px] leading-relaxed">{children}</div>
    </section>
  )
}

const bytes = (n: number) => (n < 1024 ? `${n} B` : n < 1 << 20 ? `${(n / 1024).toFixed(1)} KB` : `${(n / (1 << 20)).toFixed(1)} MB`)

function copyCurl(e: RequestEntry) {
  const parts = [`curl -X ${e.method} '${location.origin}${e.path}'`]
  for (const [k, v] of Object.entries(e.requestHeaders ?? {})) if (v !== '[redacted]' && k !== 'Content-Length') parts.push(`-H '${k}: ${v}'`)
  if (e.requestBody && !e.requestBody.startsWith('[')) parts.push(`--data '${e.requestBody}'`)
  navigator.clipboard?.writeText(parts.join(' \\\n  ')).then(() => toast('Copied as curl'))
}
