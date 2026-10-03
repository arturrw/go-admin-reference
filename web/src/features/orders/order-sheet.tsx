import { Link } from '@tanstack/react-router'
import { ChevronRight, FileText, Truck, Undo2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Avatar, Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Sheet } from '@/components/ui/sheet'
import type { OrderStatus } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { money, timeAgo } from '@/lib/format'
import { useOrder, useUpdateOrderStatus } from '@/lib/queries'
import { cn } from '@/lib/utils'

const STEPS = ['Placed', 'Paid', 'Packed', 'Shipped', 'Delivered']
const STEPS_DONE: Record<OrderStatus, number> = { pending: 1, paid: 2, shipped: 4, delivered: 5, refunded: 2, failed: 1 }

/** Order detail sheet, fetched by id so it works as a deep link (/orders?view=10480). */
export function OrderSheet({ orderId, onClose }: { orderId: number; onClose: () => void }) {
  const { data: o, isPending, isError } = useOrder(orderId)
  const update = useUpdateOrderStatus()
  const canWrite = useCan('orders:write')
  const canCustomers = useCan('customers:read')

  if (isError) return null

  const subtotal = o?.totalCents ?? 0
  const shipping = subtotal > 10_000 ? 0 : 790
  const tax = Math.round(subtotal * 0.08)
  const done = o ? STEPS_DONE[o.status] : 0
  const stepTime = (i: number) =>
    new Date(new Date(o!.placedAt).getTime() + [0, 1, 300, 1400, 4500][i] * 60_000).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })

  return (
    <Sheet
      open
      onOpenChange={(v) => !v && onClose()}
      title={`Order #${orderId}`}
      description={o ? `${timeAgo(o.placedAt)} · ${o.payment}` : 'Loading…'}
      footer={
        o && (
          <>
            <Button onClick={() => toast('Invoice downloaded')} className={cn(!canWrite && 'mr-auto')}>
              <FileText />
              Invoice
            </Button>
            {canWrite && (
              <>
                <Button variant="danger" className="order-first mr-auto" disabled={update.isPending || o.status === 'refunded'} onClick={() => update.mutate({ id: o.id, status: 'refunded' })}>
                  <Undo2 />
                  Refund
                </Button>
                <Button
                  variant="primary"
                  disabled={update.isPending || ['shipped', 'delivered', 'refunded', 'failed'].includes(o.status)}
                  onClick={() => update.mutate({ id: o.id, status: 'shipped' })}
                >
                  <Truck />
                  Mark shipped
                </Button>
              </>
            )}
          </>
        )
      }
    >
      {isPending || !o ? (
        <Skeleton className="h-96" />
      ) : (
        <>
          <div>
            <StatusPill status={o.status} />
          </div>
          {canCustomers ? (
            <Link
              to="/customers"
              search={{ view: o.customer.id }}
              className="card flex items-center gap-2.5 p-3.5 transition-colors hover:border-accent/35"
              aria-label={`Open customer ${o.customer.name}`}
            >
              <CustomerRow o={o} />
              <ChevronRight className="size-4 text-dim" />
            </Link>
          ) : (
            <div className="card flex items-center gap-2.5 p-3.5">
              <CustomerRow o={o} />
            </div>
          )}

          <div>
            <div className="eyebrow mb-1.5">Items</div>
            {o.items.map((it, i) => (
              <div key={i} className="flex items-center gap-2.5 border-b border-dashed border-line py-2 last:border-0">
                <ProductThumb category={it.category} hue={it.hue} src={it.imageUrl} size={40} />
                <div className="min-w-0 flex-1">
                  <b className="block truncate font-medium">{it.name}</b>
                  <div className="num text-[11.5px] text-dim">
                    {it.sku} · {money(it.priceCents, 2)} × {it.qty}
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
              <div
                key={s}
                className="relative flex gap-3 pb-3.5 not-last:after:absolute not-last:after:top-4 not-last:after:bottom-0 not-last:after:left-1.5 not-last:after:w-px not-last:after:bg-line-2"
              >
                <i className={cn('mt-[3px] size-[13px] shrink-0 rounded-full border-2', i < done ? 'border-accent bg-accent' : 'border-line-2 bg-panel')} />
                <div>
                  <b className="block text-[13px] font-medium">{s}</b>
                  <small className="num text-[11px] text-dim">{i < done ? stepTime(i) : o.status === 'failed' ? 'payment failed' : 'waiting'}</small>
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </Sheet>
  )
}

function CustomerRow({ o }: { o: { customer: { name: string; email: string; country: string; segment: string } } }) {
  return (
    <>
      <Avatar name={o.customer.name} size={36} />
      <div className="min-w-0 flex-1">
        <b className="block font-medium">{o.customer.name}</b>
        <small className="block truncate text-xs text-dim">
          {o.customer.email} · {o.customer.country}
        </small>
      </div>
      <StatusPill status={o.customer.segment} />
    </>
  )
}
