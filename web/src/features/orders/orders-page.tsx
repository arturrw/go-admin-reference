import { ChevronRight, Download, FileText, Inbox, Plus, Truck, Undo2 } from 'lucide-react'
import { useDeferredValue, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, TableCard } from '@/components/ui/card'
import { SearchInput } from '@/components/ui/input'
import { Avatar, EmptyState, PageHeader, Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Segmented } from '@/components/ui/segmented'
import { Sheet } from '@/components/ui/sheet'
import type { Order, OrderStatus } from '@/lib/api'
import { capitalize, money, timeAgo } from '@/lib/format'
import { useOrders, useUpdateOrderStatus } from '@/lib/queries'
import { cn } from '@/lib/utils'

const STATUSES: (OrderStatus | 'all')[] = ['all', 'pending', 'paid', 'shipped', 'delivered', 'refunded', 'failed']

export function OrdersPage() {
  const [status, setStatus] = useState<OrderStatus | 'all'>('all')
  const [q, setQ] = useState('')
  const [openId, setOpenId] = useState<number | null>(null)
  const { data, isPending } = useOrders({ status, q: useDeferredValue(q) })
  const counts = data?.counts ?? {}
  const total = Object.values(counts).reduce((a, b) => a + (b ?? 0), 0)
  const open = data?.items.find((o) => o.id === openId)

  return (
    <>
      <PageHeader title="Orders" description={`${total} orders in the last 7 days`}>
        <Button>
          <Download />
          Export
        </Button>
        <Button variant="primary" onClick={() => toast('Draft order created')}>
          <Plus />
          Create order
        </Button>
      </PageHeader>

      <div className="mb-3.5 flex flex-wrap items-center gap-2.5">
        <Segmented
          value={status}
          onChange={setStatus}
          options={STATUSES.map((s) => ({
            value: s,
            label: (
              <>
                {capitalize(s)}
                <span className="num text-[11px] opacity-55">{s === 'all' ? total : (counts[s] ?? 0)}</span>
              </>
            ),
          }))}
        />
        <SearchInput className="ml-auto" placeholder="Order # or customer…" value={q} onChange={(e) => setQ(e.target.value)} />
      </div>

      {isPending ? (
        <Skeleton className="h-125" />
      ) : !data?.items.length ? (
        <Card>
          <EmptyState icon={Inbox} title="No orders found." />
        </Card>
      ) : (
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
                <tr key={o.id} className="cursor-pointer" onClick={() => setOpenId(o.id)}>
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
                          <ProductThumb category={it.category} hue={it.hue} size={28} />
                        </div>
                      ))}
                      <span className="ml-2 text-[12.5px] text-muted">{o.items.reduce((s, i) => s + i.qty, 0)} items</span>
                    </div>
                  </td>
                  <td className="text-muted">{o.payment}</td>
                  <td>
                    <StatusPill status={o.status} />
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
      )}

      {open && <OrderSheet order={open} onClose={() => setOpenId(null)} />}
    </>
  )
}

const STEPS = ['Placed', 'Paid', 'Packed', 'Shipped', 'Delivered']
const STEPS_DONE: Record<OrderStatus, number> = { pending: 1, paid: 2, shipped: 4, delivered: 5, refunded: 2, failed: 1 }

function OrderSheet({ order: o, onClose }: { order: Order; onClose: () => void }) {
  const update = useUpdateOrderStatus()
  const subtotal = o.totalCents
  const shipping = subtotal > 10_000 ? 0 : 790
  const tax = Math.round(subtotal * 0.08)
  const done = STEPS_DONE[o.status]
  const placed = new Date(o.placedAt)
  const stepTime = (i: number) =>
    new Date(placed.getTime() + [0, 1, 300, 1400, 4500][i] * 60_000).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })

  return (
    <Sheet
      open
      onOpenChange={(v) => !v && onClose()}
      title={`Order #${o.id}`}
      description={`${timeAgo(o.placedAt)} · ${o.payment}`}
      footer={
        <>
          <Button variant="danger" className="mr-auto" disabled={update.isPending || o.status === 'refunded'} onClick={() => update.mutate({ id: o.id, status: 'refunded' })}>
            <Undo2 />
            Refund
          </Button>
          <Button onClick={() => toast('Invoice downloaded')}>
            <FileText />
            Invoice
          </Button>
          <Button variant="primary" disabled={update.isPending || ['shipped', 'delivered', 'refunded'].includes(o.status)} onClick={() => update.mutate({ id: o.id, status: 'shipped' })}>
            <Truck />
            Mark shipped
          </Button>
        </>
      }
    >
      <div>
        <StatusPill status={o.status} />
      </div>
      <Card className="p-3.5">
        <div className="flex items-center gap-2.5">
          <Avatar name={o.customer.name} size={36} />
          <div>
            <b className="block font-medium">{o.customer.name}</b>
            <small className="block text-xs text-dim">
              {o.customer.email} · {o.customer.country}
            </small>
          </div>
          <span className="ml-auto">
            <StatusPill status={o.customer.segment} />
          </span>
        </div>
      </Card>

      <div>
        <div className="eyebrow mb-1.5">Items</div>
        {o.items.map((it, i) => (
          <div key={i} className="flex items-center gap-2.5 border-b border-dashed border-line py-2 last:border-0">
            <ProductThumb category={it.category} hue={it.hue} size={36} />
            <div className="min-w-0 flex-1">
              <b className="block truncate font-medium">{it.name}</b>
              <div className="num text-[11.5px] text-dim">
                {it.sku} × {it.qty}
              </div>
            </div>
            <b className="num font-medium">{money(it.priceCents * it.qty, 2)}</b>
          </div>
        ))}
      </div>

      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-[13px] [&_dd]:num [&_dd]:text-right [&_dt]:text-dim">
        <dt>Subtotal</dt>
        <dd>{money(subtotal, 2)}</dd>
        <dt>Shipping</dt>
        <dd>{shipping ? money(shipping, 2) : 'Free'}</dd>
        <dt>Tax (8%)</dt>
        <dd>{money(tax, 2)}</dd>
        <dt className="text-fg!">Total</dt>
        <dd className="text-base text-accent">{money(subtotal + shipping + tax, 2)}</dd>
      </dl>

      <div>
        <div className="eyebrow mb-2.5">Fulfillment</div>
        {STEPS.map((s, i) => (
          <div key={s} className="relative flex gap-3 pb-3.5 not-last:after:absolute not-last:after:top-4 not-last:after:bottom-0 not-last:after:left-1.5 not-last:after:w-px not-last:after:bg-line-2">
            <i className={cn('mt-[3px] size-[13px] shrink-0 rounded-full border-2', i < done ? 'border-accent bg-accent' : 'border-line-2 bg-panel')} />
            <div>
              <b className="block text-[13px] font-medium">{s}</b>
              <small className="num text-[11px] text-dim">{i < done ? stepTime(i) : 'waiting'}</small>
            </div>
          </div>
        ))}
      </div>
    </Sheet>
  )
}
