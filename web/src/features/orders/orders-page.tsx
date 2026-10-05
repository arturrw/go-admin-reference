import { useNavigate, useSearch } from '@tanstack/react-router'
import { ChevronLeft, ChevronRight, Download, Inbox, Undo2 } from 'lucide-react'
import { useDeferredValue, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, TableCard } from '@/components/ui/card'
import { SearchInput } from '@/components/ui/input'
import { Avatar, EmptyState, PageHeader, Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Segmented } from '@/components/ui/segmented'
import { exportUrl, type OrderStatus } from '@/lib/api'
import { downloadUrl } from '@/lib/download'
import { capitalize, money, timeAgo } from '@/lib/format'
import { useOrders } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { OrderSheet } from './order-sheet'

const STATUSES: (OrderStatus | 'all')[] = ['all', 'pending', 'paid', 'shipped', 'delivered', 'refunded', 'failed']
const PAGE = 25

const units = (n: number) => `${n} ${n === 1 ? 'item' : 'items'}`

export function OrdersPage() {
  const { view } = useSearch({ from: '/app/orders' })
  const navigate = useNavigate({ from: '/orders' })
  const [status, setStatus] = useState<OrderStatus | 'all'>('all')
  const [q, setQ] = useState('')
  const [page, setPage] = useState(0)
  const dq = useDeferredValue(q)
  const { data, isPending } = useOrders({ status, q: dq, limit: PAGE, offset: page * PAGE })
  const counts = data?.counts ?? {}
  const all = Object.values(counts).reduce((a, b) => a + (b ?? 0), 0)
  const total = data?.total ?? 0
  const pages = Math.max(1, Math.ceil(total / PAGE))

  const filter = (fn: () => void) => {
    fn()
    setPage(0)
  }

  return (
    <>
      <PageHeader title="Orders" description={`${all} orders · newest first`}>
        <Button onClick={() => downloadUrl(exportUrl('orders', { status, q: dq }))}>
          <Download />
          Export CSV
        </Button>
      </PageHeader>

      <div className="mb-3.5 flex flex-wrap items-center gap-2.5">
        <Segmented
          value={status}
          onChange={(s) => filter(() => setStatus(s))}
          options={STATUSES.map((s) => ({
            value: s,
            label: (
              <>
                {capitalize(s)}
                <span className="num text-[11px] opacity-55">{s === 'all' ? all : (counts[s] ?? 0)}</span>
              </>
            ),
          }))}
        />
        <SearchInput className="ml-auto" placeholder="Order #, customer or email…" value={q} onChange={(e) => filter(() => setQ(e.target.value))} />
      </div>

      {isPending ? (
        <Skeleton className="h-125" />
      ) : !data?.items.length ? (
        <Card>
          <EmptyState icon={Inbox} title="No orders found." />
        </Card>
      ) : (
        <>
          <TableCard>
            <table className="data-table">
              <thead>
                <tr>
                  <th>Order</th>
                  <th>Customer</th>
                  <th>Items</th>
                  <th>Payment</th>
                  <th>Status</th>
                  <th className="num">Total</th>
                  <th className="num">Placed</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {data.items.map((o) => (
                  <tr key={o.id} className="cursor-pointer" onClick={() => navigate({ search: { view: o.id } })}>
                    <td className="num">#{o.id}</td>
                    <td>
                      <div className="flex items-center gap-2.5">
                        <Avatar name={o.customer.name} size={28} />
                        <div>
                          <b className="block font-medium">{o.customer.name}</b>
                          <small className="block text-xs text-dim">{o.customer.email}</small>
                        </div>
                      </div>
                    </td>
                    <td>
                      <div className="flex items-center">
                        {o.items.slice(0, 3).map((it, k) => (
                          <div key={k} className={cn('rounded-[10px] ring-2 ring-panel', k > 0 && '-ml-2')}>
                            <ProductThumb category={it.category} hue={it.hue} src={it.imageUrl} size={28} />
                          </div>
                        ))}
                        <span className="ml-2 text-[12.5px] text-muted">{units(o.items.reduce((s, i) => s + i.qty, 0))}</span>
                      </div>
                    </td>
                    <td className="text-muted">{o.payment}</td>
                    <td>
                      <StatusPill status={o.status} />
                      {o.status === 'refunded' && (
                        <small
                          className="mt-1 flex max-w-56 items-center gap-1 text-[11.5px] text-violet"
                          title={o.refund ? `${o.refund.reason} · refunded by ${o.refund.by}` : undefined}
                          data-testid="refund-reason"
                        >
                          <Undo2 className="size-3 shrink-0" />
                          <span className="truncate">{o.refund?.reason ?? 'No reason recorded'}</span>
                        </small>
                      )}
                    </td>
                    <td className="num">{money(o.totalCents, 2)}</td>
                    <td className="num text-muted">{timeAgo(o.placedAt)}</td>
                    <td className="num">
                      <ChevronRight className="inline size-4 text-dim" />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableCard>
          <div className="mt-3 flex items-center justify-between gap-3 text-[12.5px] text-muted">
            <span className="num">
              {page * PAGE + 1}–{Math.min(total, (page + 1) * PAGE)} of {total}
            </span>
            <div className="flex items-center gap-1.5">
              <Button size="sm" disabled={page === 0} onClick={() => setPage(page - 1)} aria-label="Previous page">
                <ChevronLeft />
              </Button>
              <span className="num px-1.5">
                {page + 1} / {pages}
              </span>
              <Button size="sm" disabled={page + 1 >= pages} onClick={() => setPage(page + 1)} aria-label="Next page">
                <ChevronRight />
              </Button>
            </div>
          </div>
        </>
      )}

      {view !== undefined && <OrderSheet orderId={view} onClose={() => navigate({ search: {} })} />}
    </>
  )
}
