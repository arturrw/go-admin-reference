import { useNavigate, useSearch } from '@tanstack/react-router'
import { ChevronRight, Crown, Download, type LucideIcon, Mail, Sparkles, TriangleAlert, User, UserPlus, UserX, X } from 'lucide-react'
import { useDeferredValue, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, TableCard } from '@/components/ui/card'
import { SearchInput } from '@/components/ui/input'
import { Avatar, EmptyState, IconTile, PageHeader, Skeleton } from '@/components/ui/misc'
import { STATUS_TONE, StatusPill, toneColor } from '@/components/ui/pill'
import { exportUrl, type Segment } from '@/lib/api'
import { downloadUrl } from '@/lib/download'
import { int, money, monthYear, timeAgo } from '@/lib/format'
import { useCustomers } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { CustomerSheet } from './customer-sheet'

const SEGMENTS: [Segment, LucideIcon][] = [
  ['VIP', Crown],
  ['Regular', User],
  ['New', Sparkles],
  ['At risk', TriangleAlert],
]

export function CustomersPage() {
  const { view } = useSearch({ from: '/app/customers' })
  const navigate = useNavigate({ from: '/customers' })
  const [q, setQ] = useState('')
  const [segment, setSegment] = useState<Segment | ''>('')
  const dq = useDeferredValue(q)
  const { data, isPending } = useCustomers({ q: dq, segment })
  const segs = data?.segments ?? {}
  const totalLtv = Object.values(segs).reduce((s, x) => s + (x?.ltvCents ?? 0), 0)
  const totalCount = Object.values(segs).reduce((s, x) => s + (x?.count ?? 0), 0)
  const maxLtv = Math.max(1, ...(data?.items.map((c) => c.ltvCents) ?? []))

  return (
    <>
      <PageHeader title="Customers" description={`${totalCount} customers · ${money(totalLtv)} lifetime value`}>
        <Button onClick={() => downloadUrl(exportUrl('customers', { q: dq, segment }))}>
          <Download />
          Export CSV
        </Button>
        <Button>
          <Mail />
          Email segment
        </Button>
        <Button variant="primary" onClick={() => toast('Customer invited')}>
          <UserPlus />
          Add customer
        </Button>
      </PageHeader>

      <div className="mb-3.5 grid grid-cols-2 gap-3.5 lg:grid-cols-4">
        {SEGMENTS.map(([s, Icon]) => {
          const c = toneColor(STATUS_TONE[s])
          const on = segment === s
          return (
            <button
              key={s}
              onClick={() => setSegment(on ? '' : s)}
              className={cn('group card text-left transition-colors hover:border-line-2')}
              style={on ? { borderColor: `color-mix(in srgb, ${c} 45%, transparent)` } : undefined}
            >
              <div className="flex items-center gap-2 text-[12.5px] text-muted">
                <IconTile icon={Icon} color={c} />
                {s}
              </div>
              <div className="num mt-2.5 text-2xl font-semibold tracking-[-0.03em]">{segs[s]?.count ?? '—'}</div>
              <div className="text-xs text-dim">{money(segs[s]?.ltvCents ?? 0)} LTV</div>
            </button>
          )
        })}
      </div>

      <div className="mb-3.5 flex flex-wrap items-center gap-2.5">
        <SearchInput placeholder="Search customers…" value={q} onChange={(e) => setQ(e.target.value)} />
        {segment && (
          <button onClick={() => setSegment('')} className="inline-flex h-7.5 items-center gap-1.5 rounded-full border border-accent/35 bg-accent/14 px-3 text-[12.5px] font-medium text-accent">
            {segment}
            <X className="size-3.5" />
          </button>
        )}
      </div>

      {isPending ? (
        <Skeleton className="h-125" />
      ) : !data?.items.length ? (
        <Card>
          <EmptyState icon={UserX} title="No customers found." />
        </Card>
      ) : (
        <TableCard>
          <table className="data-table">
            <thead>
              <tr>
                <th>Customer</th>
                <th>Country</th>
                <th>Segment</th>
                <th className="num">Orders</th>
                <th>Lifetime value</th>
                <th>Customer since</th>
                <th className="num">Last seen</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {data.items.map((c) => (
                <tr key={c.id} className="cursor-pointer" onClick={() => navigate({ search: { view: c.id } })}>
                  <td>
                    <div className="flex items-center gap-2.5">
                      <Avatar name={c.name} />
                      <div>
                        <b className="block font-medium">{c.name}</b>
                        <small className="block text-xs text-dim">{c.email}</small>
                      </div>
                    </div>
                  </td>
                  <td>
                    <span className="num rounded-[5px] border border-line-2 px-1.5 py-0.5 text-[10.5px] font-semibold text-muted">{c.country}</span>
                  </td>
                  <td>
                    <StatusPill status={c.segment} />
                  </td>
                  <td className="num">{int(c.orders)}</td>
                  <td>
                    <div className="flex items-center gap-2.5">
                      <span className="num min-w-16">{money(c.ltvCents)}</span>
                      <div className="h-1 w-22 overflow-hidden rounded bg-panel-3">
                        <i className="block h-full rounded bg-accent" style={{ width: `${(c.ltvCents / maxLtv) * 100}%` }} />
                      </div>
                    </div>
                  </td>
                  <td className="text-muted">{monthYear(c.createdAt)}</td>
                  <td className="num text-muted">{timeAgo(c.lastSeenAt)}</td>
                  <td className="num">
                    <ChevronRight className="inline size-4 text-dim" />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableCard>
      )}
      {view !== undefined && <CustomerSheet key={view} customerId={view} onClose={() => navigate({ search: {} })} />}
    </>
  )
}
