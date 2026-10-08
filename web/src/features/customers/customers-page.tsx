import { useNavigate, useSearch } from '@tanstack/react-router'
import { ChevronRight, Crown, Download, type LucideIcon, Mail, Sparkles, TriangleAlert, User, UserPlus, UserX, X } from 'lucide-react'
import { useDeferredValue, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, TableCard } from '@/components/ui/card'
import { SearchInput } from '@/components/ui/input'
import { Avatar, EmptyState, IconTile, PageHeader, Skeleton } from '@/components/ui/misc'
import { STATUS_TONE, StatusPill, toneColor } from '@/components/ui/pill'
import { exportUrl, type Segment } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { downloadUrl } from '@/lib/download'
import { int, money, monthYear, timeAgo } from '@/lib/format'
import { useCustomers } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { CustomerForm } from './customer-form'
import { CustomerSheet } from './customer-sheet'
import { EmailSegment } from './email-segment'

const SEGMENTS: [Segment, LucideIcon][] = [
  ['VIP', Crown],
  ['Regular', User],
  ['New', Sparkles],
  ['At risk', TriangleAlert],
]

export function CustomersPage() {
  const { view } = useSearch({ from: '/app/customers' })
  const navigate = useNavigate({ from: '/customers' })
  const canAdd = useCan('customers:write')
  const [adding, setAdding] = useState(false)
  const [emailing, setEmailing] = useState(false)
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
        <Button disabled={!data?.items.length} onClick={() => setEmailing(true)}>
          <Mail />
          Email {segment ? segment.toLowerCase() : 'customers'}
        </Button>
        {canAdd && (
          <Button variant="primary" onClick={() => setAdding(true)}>
            <UserPlus />
            Add customer
          </Button>
        )}
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
                <th className="max-sm:hidden">Country</th>
                <th className="max-sm:hidden">Segment</th>
                <th className="num max-sm:hidden">Orders</th>
                <th className="max-sm:text-right">Lifetime value</th>
                <th className="max-md:hidden">Customer since</th>
                <th className="num max-md:hidden">Last seen</th>
                <th className="max-sm:hidden" />
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
                        <small className="block max-w-44 truncate text-xs text-dim">{c.email}</small>
                        <span className="mt-1 block sm:hidden">
                          <StatusPill status={c.segment} />
                        </span>
                      </div>
                    </div>
                  </td>
                  <td className="max-sm:hidden">
                    <span className="num rounded-[5px] border border-line-2 px-1.5 py-0.5 text-[10.5px] font-semibold text-muted">{c.country}</span>
                  </td>
                  <td className="max-sm:hidden">
                    <StatusPill status={c.segment} />
                  </td>
                  <td className="num max-sm:hidden">{int(c.orders)}</td>
                  <td>
                    <div className="flex items-center gap-2.5 max-sm:justify-end">
                      <span className="num min-w-16 max-sm:text-right">{money(c.ltvCents)}</span>
                      <div className="h-1 w-22 overflow-hidden rounded bg-panel-3 max-sm:hidden">
                        <i className="block h-full rounded bg-accent" style={{ width: `${(c.ltvCents / maxLtv) * 100}%` }} />
                      </div>
                    </div>
                  </td>
                  <td className="text-muted max-md:hidden">{monthYear(c.createdAt)}</td>
                  <td className="num text-muted max-md:hidden">{timeAgo(c.lastSeenAt)}</td>
                  <td className="num max-sm:hidden">
                    <ChevronRight className="inline size-4 text-dim" />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableCard>
      )}
      {emailing && data && <EmailSegment customers={data.items} label={segment ? `the ${segment} segment` : q ? 'these customers' : 'all customers'} onClose={() => setEmailing(false)} />}
      {adding && (
        <CustomerForm
          onClose={() => setAdding(false)}
          onCreated={(c) => {
            setAdding(false)
            navigate({ search: { view: c.id } })
          }}
        />
      )}
      {view !== undefined && <CustomerSheet key={view} customerId={view} onClose={() => navigate({ search: {} })} />}
    </>
  )
}
