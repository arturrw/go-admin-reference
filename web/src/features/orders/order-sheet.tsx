import { CheckCheck, ChevronRight, CreditCard, FileText, PackageCheck, Truck, Undo2 } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { Button, buttonVariants } from '@/components/ui/button'
import { Dialog } from '@/components/ui/dialog'
import { Textarea } from '@/components/ui/input'
import { Avatar, Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Sheet } from '@/components/ui/sheet'
import { ApiError, type Order, type OrderStatus, REFUND_REASONS } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { money, timeAgo } from '@/lib/format'
import { PeekButton } from '@/lib/peek'
import { useOrder, useUpdateOrderStatus } from '@/lib/queries'
import { cn } from '@/lib/utils'

const EVENT_LABEL: Record<OrderStatus, string> = { pending: 'Placed', paid: 'Paid', shipped: 'Shipped', delivered: 'Delivered', refunded: 'Refunded', failed: 'Payment failed' }
/** What is still to come, by where the order is now. */
const NEXT_STEP: Partial<Record<OrderStatus, string>> = { pending: 'Awaiting payment', paid: 'Awaiting shipment', shipped: 'Awaiting delivery' }
/** The forward move each status offers (mirrors domain.CanTransition). */
const ADVANCE: Partial<Record<OrderStatus, { to: OrderStatus; label: string; icon: typeof Truck }>> = {
  pending: { to: 'paid', label: 'Mark paid', icon: CreditCard },
  paid: { to: 'shipped', label: 'Mark shipped', icon: Truck },
  shipped: { to: 'delivered', label: 'Mark delivered', icon: PackageCheck },
}
const when = (at: string) => new Date(at).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })

/** Order detail sheet, fetched by id so it works as a deep link (/orders?view=10480). */
export function OrderSheet({ orderId, onClose }: { orderId: number; onClose: () => void }) {
  const { data: o, isPending, isError } = useOrder(orderId)
  const update = useUpdateOrderStatus()
  const canWrite = useCan('orders:write')
  const canCustomers = useCan('customers:read')
  const canProducts = useCan('products:read')
  const [refunding, setRefunding] = useState(false)

  if (isError) return null

  const advance = o && ADVANCE[o.status]

  return (
    <Sheet
      open
      onOpenChange={(v) => !v && onClose()}
      title={`Order #${orderId}`}
      description={o ? `${timeAgo(o.placedAt)} · ${o.payment}` : 'Loading…'}
      footer={
        o && (
          <>
            <a
              href={`/api/v1/orders/${o.id}/invoice`}
              target="_blank"
              rel="noopener"
              className={cn(buttonVariants(), !canWrite && 'mr-auto')}
              title="Opens a printable invoice; use Print → Save as PDF"
            >
              <FileText />
              Invoice
            </a>
            {canWrite && (
              <>
                <Button variant="danger" className="order-first mr-auto" disabled={update.isPending || o.status === 'refunded' || o.status === 'failed'} onClick={() => setRefunding(true)}>
                  <Undo2 />
                  Refund
                </Button>
                {advance ? (
                  <Button variant="primary" className="max-sm:w-full" disabled={update.isPending} onClick={() => update.mutate({ id: o.id, status: advance.to })}>
                    <advance.icon />
                    {advance.label}
                  </Button>
                ) : (
                  <Button variant="primary" className="max-sm:w-full" disabled>
                    <CheckCheck />
                    {o.status === 'delivered' ? 'Delivered' : 'Closed'}
                  </Button>
                )}
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
          {o.refund ? (
            <RefundNote refund={o.refund} />
          ) : (
            o.status === 'refunded' && <p className="text-[12.5px] text-dim">Refunded before reasons were recorded.</p>
          )}
          <RefundDialog key={String(refunding)} order={o} open={refunding} onOpenChange={setRefunding} />
          {canCustomers ? (
            <PeekButton
              kind="customer"
              id={o.customer.id}
              className="card flex items-center gap-2.5 p-3.5 text-left transition-colors hover:border-accent/35"
              label={`Open customer ${o.customer.name}`}
            >
              <CustomerRow o={o} />
              <ChevronRight className="size-4 text-dim" />
            </PeekButton>
          ) : (
            <div className="card flex items-center gap-2.5 p-3.5">
              <CustomerRow o={o} />
            </div>
          )}

          <div>
            <div className="eyebrow mb-1.5">Items</div>
            {o.items.map((it, i) => (
              <PeekButton
                key={i}
                kind="product"
                id={it.productId}
                disabled={!canProducts || !it.productId}
                label={`Open ${it.name}`}
                className="-mx-2 flex w-[calc(100%+16px)] items-center gap-2.5 rounded-lg border-b border-dashed border-line px-2 py-2 text-left last:border-0 enabled:hover:bg-panel-2"
              >
                <ProductThumb category={it.category} hue={it.hue} src={it.imageUrl} size={40} />
                <div className="min-w-0 flex-1">
                  <b className="block truncate font-medium">{it.name}</b>
                  <div className="num text-[11.5px] text-dim">
                    {it.sku} · {money(it.priceCents, 2)} × {it.qty}
                  </div>
                </div>
                <b className="num font-medium">{money(it.priceCents * it.qty, 2)}</b>
              </PeekButton>
            ))}
          </div>

          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-[13px] [&_dd]:num [&_dd]:text-right [&_dt]:text-dim">
            <dt>Subtotal</dt>
            <dd>{money(o.totalCents, 2)}</dd>
            <dt>Shipping</dt>
            <dd>{o.shippingCents ? money(o.shippingCents, 2) : 'Free'}</dd>
            <dt>Tax (8%)</dt>
            <dd>{money(o.taxCents ?? 0, 2)}</dd>
            <dt className="text-fg!">Total</dt>
            <dd className="text-base text-accent">{money(o.grandTotalCents ?? o.totalCents, 2)}</dd>
          </dl>

          <div>
            <div className="eyebrow mb-2.5">Timeline</div>
            <ol data-testid="order-timeline">
              {(o.events ?? []).map((e, i) => (
                <TimelineStep key={i} done label={EVENT_LABEL[e.status]} note={`${when(e.at)} · ${e.by}`} last={i === (o.events?.length ?? 0) - 1 && !NEXT_STEP[o.status]} />
              ))}
              {NEXT_STEP[o.status] && <TimelineStep label={NEXT_STEP[o.status]!} note="waiting" last />}
            </ol>
          </div>
        </>
      )}
    </Sheet>
  )
}

function TimelineStep({ label, note, done, last }: { label: string; note: string; done?: boolean; last?: boolean }) {
  return (
    <li className={cn('relative flex gap-3 pb-3.5', !last && 'after:absolute after:top-4 after:bottom-0 after:left-1.5 after:w-px after:bg-line-2')}>
      <i className={cn('mt-[3px] size-[13px] shrink-0 rounded-full border-2', done ? 'border-accent bg-accent' : 'border-line-2 bg-panel')} />
      <div>
        <b className={cn('block text-[13px] font-medium', !done && 'text-dim')}>{label}</b>
        <small className="num text-[11px] text-dim">{note}</small>
      </div>
    </li>
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

/** Why the order was refunded; stays in the order and the customer's history. */
export function RefundNote({ refund, compact }: { refund: NonNullable<Order['refund']>; compact?: boolean }) {
  return (
    <div className={cn('flex gap-2.5 rounded-xl border border-violet/25 bg-violet/6 text-[13px]', compact ? 'px-2.5 py-1.5' : 'px-3.5 py-3')}>
      <Undo2 className={cn('shrink-0 text-violet', compact ? 'mt-0.5 size-3.5' : 'mt-0.5 size-4')} />
      <div className="min-w-0">
        <b className="font-medium">{refund.reason}</b>
        <small className="block text-xs text-dim">
          Refunded by {refund.by} · {when(refund.at)}
        </small>
      </div>
    </div>
  )
}

function RefundDialog({ order, open, onOpenChange }: { order: Order; open: boolean; onOpenChange: (v: boolean) => void }) {
  const update = useUpdateOrderStatus()
  const [preset, setPreset] = useState<string>(REFUND_REASONS[0])
  const [details, setDetails] = useState('')
  const reason = preset === 'Other' ? details.trim() : details.trim() ? `${preset}: ${details.trim()}` : preset
  const error = update.error instanceof ApiError ? update.error.fields?.reason : undefined

  const submit = (e: FormEvent) => {
    e.preventDefault()
    update.mutate({ id: order.id, status: 'refunded', reason }, { onSuccess: () => onOpenChange(false) })
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={`Refund order #${order.id}`}
      description={`Refunds ${money(order.totalCents, 2)} to ${order.customer.name}. The reason is kept in the order and the customer's purchase history, and the items go back to stock.`}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button variant="danger" type="submit" form="refund-form" disabled={update.isPending || !reason}>
            <Undo2 />
            {update.isPending ? 'Refunding…' : `Refund ${money(order.totalCents, 2)}`}
          </Button>
        </>
      }
    >
      <form id="refund-form" onSubmit={submit} className="flex flex-col gap-3">
        <div role="radiogroup" aria-label="Reason" className="flex flex-wrap gap-1.5">
          {[...REFUND_REASONS, 'Other'].map((r) => (
            <button
              key={r}
              type="button"
              role="radio"
              aria-checked={preset === r}
              onClick={() => setPreset(r)}
              className={cn(
                'rounded-lg border px-2.5 py-1 text-[12.5px] transition-colors',
                preset === r ? 'border-accent/50 bg-accent/10 text-fg' : 'border-line-2 text-muted hover:text-fg',
              )}
            >
              {r}
            </button>
          ))}
        </div>
        <Textarea
          rows={3}
          aria-label="Refund details"
          value={details}
          onChange={(e) => setDetails(e.target.value)}
          placeholder={preset === 'Other' ? 'Describe the reason (required)' : 'Add details (optional)'}
        />
        {error && <span className="text-xs text-danger">{error}</span>}
      </form>
    </Dialog>
  )
}