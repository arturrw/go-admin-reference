import { BellOff, BellRing, ChevronRight, Copy, Mail, MapPin, MessageSquarePlus, Phone, Trash2, Undo2, X } from 'lucide-react'
import { type FormEvent, type ReactNode, useState } from 'react'
import { toast } from 'sonner'
import { Donut, SERIES_COLORS } from '@/components/charts/donut'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/ui/dialog'
import { Textarea } from '@/components/ui/input'
import { Avatar, Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Sheet } from '@/components/ui/sheet'
import { Tabs } from '@/components/ui/tabs'
import type { Category, CustomerDetail, CustomerNote, OrderStatus } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { compact, int, money, monthYear, shortDate, timeAgo } from '@/lib/format'
import { PeekButton } from '@/lib/peek'
import { useAddCustomerNote, useCustomer, useDeleteCustomerNote } from '@/lib/queries'
import { cn } from '@/lib/utils'

type Tab = 'overview' | 'orders' | 'products' | 'notes'

/** What the overview widgets drill into: orders of a month, a category or a status. */
interface Filter {
  month?: string // YYYY-MM
  category?: Category
  status?: OrderStatus
}

const monthName = (m: string) => new Date(m + '-01').toLocaleDateString('en-US', { month: 'long', year: 'numeric' })

/** Everything about one customer: profile, metrics, purchase history and notes. */
export function CustomerSheet({ customerId, onClose }: { customerId: number; onClose: () => void }) {
  const { data, isPending, isError } = useCustomer(customerId)
  const [tab, setTab] = useState<Tab>('overview')
  const [filter, setFilter] = useState<Filter>({})
  const go = (t: Tab, f: Filter = {}) => {
    setFilter(f)
    setTab(t)
  }
  if (isError) return null
  const c = data?.customer

  return (
    <Sheet open onOpenChange={(v) => !v && onClose()} size="lg" title={c?.name ?? 'Customer'} description={c ? `Customer #${c.id} · since ${monthYear(c.createdAt)}` : 'Loading…'}>
      {isPending || !data || !c ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-28" />
          <Skeleton className="h-60" />
        </div>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-3.5">
            <Avatar name={c.name} size={52} />
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <b className="text-lg font-semibold tracking-[-0.01em]">{c.name}</b>
                <StatusPill status={c.segment} />
              </div>
              <div className="mt-1 flex flex-wrap gap-1.5">
                {c.tags.map((t) => (
                  <span key={t} className="rounded-md bg-panel-3 px-2 py-0.5 text-[11.5px] text-muted">
                    {t}
                  </span>
                ))}
              </div>
            </div>
            <Button size="sm" onClick={() => (window.location.href = `mailto:${c.email}`)}>
              <Mail />
              Email
            </Button>
          </div>

          <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
            <Metric label="Total spent" value={money(data.stats.totalSpentCents)} accent />
            <Metric label="Orders" value={int(data.stats.orders)} />
            <Metric label="Avg. order" value={money(data.stats.avgOrderCents)} />
            <Metric label="Items bought" value={int(data.stats.itemsBought)} />
          </div>

          <Tabs<Tab>
            value={tab}
            onChange={(t) => go(t)}
            tabs={[
              { value: 'overview', label: 'Overview' },
              { value: 'orders', label: 'Orders', count: data.orders.length },
              { value: 'products', label: 'Products', count: data.products.length },
              { value: 'notes', label: 'Notes', count: c.notes.length },
            ]}
          />

          {tab === 'overview' && <Overview data={data} onDrill={go} />}
          {tab === 'orders' && <OrdersTab data={data} filter={filter} onClear={() => setFilter({})} />}
          {tab === 'products' && <ProductsTab data={data} category={filter.category} onClear={() => setFilter({})} />}
          {tab === 'notes' && <NotesTab data={data} />}
        </>
      )}
    </Sheet>
  )
}

function Metric({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div className="card px-3 py-2.5">
      <div className="eyebrow">{label}</div>
      <div className={cn('num mt-0.5 text-[17px] font-semibold tracking-[-0.02em]', accent && 'text-accent')}>{value}</div>
    </div>
  )
}

function Overview({ data, onDrill }: { data: CustomerDetail; onDrill: (t: Tab, f?: Filter) => void }) {
  const c = data.customer
  const maxMonth = Math.max(1, ...data.monthly.map((m) => m.cents))
  const catTotal = data.categories.reduce((s, x) => s + x.salesCents, 0) || 1
  const copy = (v: string) => navigator.clipboard?.writeText(v).then(() => toast('Copied'))
  const [month, setMonth] = useState<number | null>(null)
  const [cat, setCat] = useState<number | null>(null)
  // Orders are newest first.
  const lastOrder = data.orders[0]
  const firstOrder = data.orders[data.orders.length - 1]
  const ordersIn = (m: string) => data.orders.filter((o) => o.placedAt.startsWith(m)).length
  const share = (cents: number) => `${((cents / catTotal) * 100).toFixed(0)}%`

  return (
    <div className="grid gap-4 md:grid-cols-2">
      <Panel title="Contact">
        <Row icon={<Mail />} onCopy={() => copy(c.email)}>
          {c.email}
        </Row>
        <Row icon={<Phone />} onCopy={() => copy(c.phone)}>
          <span className="num">{c.phone}</span>
        </Row>
        <Row icon={<MapPin />}>
          {c.address.line1}, {c.address.postalCode} {c.address.city}, {c.address.country}
        </Row>
        <Row icon={c.acceptsMarketing ? <BellRing /> : <BellOff />}>{c.acceptsMarketing ? 'Subscribed to marketing' : 'Not subscribed to marketing'}</Row>
      </Panel>

      <Panel title="Activity">
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-[13px] [&_dd]:text-right [&_dt]:text-dim">
          <dt>Customer since</dt>
          <dd>{shortDate(c.createdAt)}</dd>
          <dt>First order</dt>
          <dd>{firstOrder ? <OrderLink id={firstOrder.id}>{shortDate(firstOrder.placedAt)}</OrderLink> : '—'}</dd>
          <dt>Last order</dt>
          <dd>{lastOrder ? <OrderLink id={lastOrder.id}>{timeAgo(lastOrder.placedAt)}</OrderLink> : '—'}</dd>
          <dt>Last seen</dt>
          <dd>{timeAgo(c.lastSeenAt)}</dd>
          <dt>Acquired via</dt>
          <dd>{c.source}</dd>
          <dt>Refunds</dt>
          <dd className="num">
            {data.stats.refunds ? (
              <button className="underline-offset-2 hover:text-accent hover:underline" onClick={() => onDrill('orders', { status: 'refunded' })}>
                {data.stats.refunds}
              </button>
            ) : (
              0
            )}
          </dd>
        </dl>
      </Panel>

      <Panel title="Spend · last 6 months" hint={month !== null ? `${monthName(data.monthly[month].month)}` : 'click a month'}>
        <div className="relative flex h-28 items-end gap-2" onMouseLeave={() => setMonth(null)}>
          {data.monthly.map((m, i) => {
            const n = ordersIn(m.month)
            return (
              <button
                key={m.month}
                disabled={!n}
                onMouseEnter={() => setMonth(i)}
                onFocus={() => setMonth(i)}
                onClick={() => onDrill('orders', { month: m.month })}
                aria-label={`${monthName(m.month)}: ${money(m.cents)}, ${n} orders`}
                className={cn(
                  'flex h-full flex-1 flex-col items-center justify-end gap-1.5 rounded-md transition-opacity enabled:cursor-pointer',
                  month !== null && month !== i && 'opacity-45',
                )}
              >
                <span className={cn('num text-[10px]', month === i ? 'text-fg' : 'text-dim')}>{m.cents ? compact(m.cents / 100) : ''}</span>
                <div
                  className={cn('w-full rounded-t-[4px] bg-linear-to-b from-accent to-accent/25 transition-[filter]', month === i && 'brightness-125')}
                  style={{ height: `${Math.max(3, (m.cents / maxMonth) * 72)}px` }}
                />
                <span className="num text-[10px] text-dim">{new Date(m.month + '-01').toLocaleDateString('en-US', { month: 'short' })}</span>
              </button>
            )
          })}
          {month !== null && (
            <div className="pointer-events-none absolute -top-1 left-1/2 z-10 -translate-x-1/2 rounded-lg border border-line-2 bg-panel-2/95 px-2.5 py-1.5 text-xs whitespace-nowrap shadow-float">
              <b className="num font-medium">{money(data.monthly[month].cents, 2)}</b>
              <span className="text-dim"> · {ordersIn(data.monthly[month].month)} orders</span>
            </div>
          )}
        </div>
      </Panel>

      <Panel title="Favourite categories" hint="click a category">
        {data.categories.length === 0 ? (
          <p className="text-[13px] text-dim">No purchases yet.</p>
        ) : (
          <div className="flex items-center gap-4">
            <div
              role="button"
              tabIndex={-1}
              className={cn(cat !== null && 'cursor-pointer')}
              onClick={() => cat !== null && onDrill('products', { category: data.categories[cat].category })}
            >
              <Donut
                values={data.categories.map((x) => x.salesCents)}
                labels={data.categories.map((x) => `${x.category}: ${share(x.salesCents)}`)}
                size={96}
                active={cat}
                onActive={setCat}
              >
                {cat !== null && (
                  <>
                    <b className="num text-sm font-semibold" style={{ color: SERIES_COLORS[cat] }}>
                      {share(data.categories[cat].salesCents)}
                    </b>
                    <small className="num text-[10px] text-dim">{compact(data.categories[cat].salesCents / 100)}</small>
                  </>
                )}
              </Donut>
            </div>
            <div className="flex flex-1 flex-col gap-0.5 text-[12.5px]" onMouseLeave={() => setCat(null)}>
              {data.categories.map((x, i) => (
                <button
                  key={x.category}
                  onMouseEnter={() => setCat(i)}
                  onFocus={() => setCat(i)}
                  onClick={() => onDrill('products', { category: x.category })}
                  className={cn(
                    '-mx-1.5 flex items-center gap-2 rounded-md px-1.5 py-[3px] text-left transition-[background-color,opacity]',
                    cat === i && 'bg-panel-2',
                    cat !== null && cat !== i && 'opacity-50',
                  )}
                >
                  <i className="size-2 rounded-[2px]" style={{ background: SERIES_COLORS[i] }} />
                  <span className="flex-1 text-muted">{x.category}</span>
                  <b className="num text-xs font-medium">{share(x.salesCents)}</b>
                  <ChevronRight className={cn('size-3 text-dim', cat === i ? 'opacity-100' : 'opacity-0')} />
                </button>
              ))}
            </div>
          </div>
        )}
      </Panel>
    </div>
  )
}

function OrdersTab({ data, filter, onClear }: { data: CustomerDetail; filter: Filter; onClear: () => void }) {
  const canOrders = useCan('orders:read')
  if (!data.orders.length) return <p className="py-8 text-center text-muted">No orders yet.</p>
  const orders = data.orders.filter(
    (o) =>
      (!filter.month || o.placedAt.startsWith(filter.month)) &&
      (!filter.status || o.status === filter.status) &&
      (!filter.category || o.items.some((i) => i.category === filter.category)),
  )
  const label = [filter.month && monthName(filter.month), filter.status, filter.category].filter(Boolean).join(' · ')
  return (
    <div className="flex flex-col gap-2">
      {label && <FilterChip label={label} count={orders.length} noun="orders" onClear={onClear} />}
      {orders.map((o) => (
        <PeekButton
          key={o.id}
          kind="order"
          id={o.id}
          disabled={!canOrders}
          label={`Open order #${o.id}`}
          className="card flex items-center gap-3 px-3.5 py-3 text-left transition-colors enabled:hover:border-accent/35"
        >
          <div className="flex items-center">
            {o.items.slice(0, 3).map((it, k) => (
              <div key={k} className={cn('rounded-[10px] ring-2 ring-panel', k > 0 && '-ml-2.5')}>
                <ProductThumb category={it.category} hue={it.hue} src={it.imageUrl} size={34} />
              </div>
            ))}
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <b className="num font-medium">#{o.id}</b>
              <StatusPill status={o.status} />
            </div>
            <small className="block truncate text-xs text-dim">
              {o.items.map((i) => `${i.name}${i.qty > 1 ? ` ×${i.qty}` : ''}`).join(', ')}
            </small>
            {o.refund && (
              <small className="mt-0.5 flex items-center gap-1 text-xs text-violet" data-testid="refund-reason">
                <Undo2 className="size-3 shrink-0" />
                <span className="truncate">
                  {o.refund.reason} · {o.refund.by}, {shortDate(o.refund.at)}
                </span>
              </small>
            )}
          </div>
          <div className="text-right">
            <b className="num block font-medium">{money(o.totalCents, 2)}</b>
            <small className="text-xs text-dim">{shortDate(o.placedAt)}</small>
          </div>
        </PeekButton>
      ))}
    </div>
  )
}

function ProductsTab({ data, category, onClear }: { data: CustomerDetail; category?: Category; onClear: () => void }) {
  const canOpen = useCan('products:read')
  if (!data.products.length) return <p className="py-8 text-center text-muted">No purchases yet.</p>
  const products = data.products.filter((p) => !category || p.category === category)
  return (
    <div className="grid gap-2.5 sm:grid-cols-2">
      {category && <FilterChip label={category} count={products.length} noun="products" onClear={onClear} className="sm:col-span-2" />}
      {products.map((p) => (
        <PeekButton
          key={p.productId}
          kind="product"
          id={p.productId}
          disabled={!canOpen}
          label={`Open ${p.name}`}
          className="card flex items-center gap-3 p-3 text-left transition-colors enabled:hover:border-accent/35"
        >
          <ProductThumb category={p.category} hue={p.hue} src={p.imageUrl} size={48} />
          <div className="min-w-0 flex-1">
            <b className="block truncate font-medium">{p.name}</b>
            <small className="text-xs text-dim">
              {p.category} · {p.qty} bought
            </small>
          </div>
          <b className="num text-[13px] font-medium">{money(p.spentCents)}</b>
        </PeekButton>
      ))}
    </div>
  )
}

function NotesTab({ data }: { data: CustomerDetail }) {
  const canWrite = useCan('customers:write')
  const add = useAddCustomerNote()
  const remove = useDeleteCustomerNote()
  const [text, setText] = useState('')
  const [deleting, setDeleting] = useState<CustomerNote | null>(null)
  const submit = (e: FormEvent) => {
    e.preventDefault()
    add.mutate({ id: data.customer.id, text }, { onSuccess: () => setText('') })
  }
  return (
    <div className="flex flex-col gap-3">
      {canWrite && (
        <form onSubmit={submit} className="flex flex-col gap-2">
          <Textarea rows={3} value={text} onChange={(e) => setText(e.target.value)} placeholder="Add an internal note — only staff can see it" />
          <Button type="submit" variant="primary" size="sm" className="self-end" disabled={!text.trim() || add.isPending}>
            <MessageSquarePlus />
            Add note
          </Button>
        </form>
      )}
      {data.customer.notes.length === 0 && <p className="py-6 text-center text-muted">No notes yet.</p>}
      {data.customer.notes.map((n) => (
        <div key={n.id} className="group card p-3.5" data-testid="note">
          <div className="mb-1.5 flex items-center gap-2 text-xs text-dim">
            <Avatar name={n.author} size={20} />
            <b className="font-medium text-muted">{n.author}</b>· {timeAgo(n.at)}
            {canWrite && (
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label="Delete note"
                className="-my-1 ml-auto opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100 hover:text-danger"
                onClick={() => setDeleting(n)}
              >
                <Trash2 />
              </Button>
            )}
          </div>
          <p className="text-[13px] whitespace-pre-wrap">{n.text}</p>
        </div>
      ))}
      <ConfirmDialog
        open={!!deleting}
        onOpenChange={(v) => !v && setDeleting(null)}
        title="Delete this note?"
        description={
          deleting && (
            <>
              {deleting.author}’s note “{deleting.text.length > 140 ? deleting.text.slice(0, 140) + '…' : deleting.text}” will be removed for everyone. The deletion is
              recorded in the activity log.
            </>
          )
        }
        confirmLabel="Delete note"
        pending={remove.isPending}
        onConfirm={() => deleting && remove.mutate({ id: data.customer.id, noteId: deleting.id }, { onSuccess: () => setDeleting(null) })}
      />
    </div>
  )
}

function Panel({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <section className="card p-4">
      <div className="mb-3 flex items-baseline gap-2">
        <div className="eyebrow shrink-0 whitespace-nowrap">{title}</div>
        {hint && <small className="ml-auto truncate text-[11px] text-dim">{hint}</small>}
      </div>
      {children}
    </section>
  )
}

function OrderLink({ id, children }: { id: number; children: ReactNode }) {
  const canOrders = useCan('orders:read')
  return (
    <PeekButton kind="order" id={id} disabled={!canOrders} label={`Open order #${id}`} className="underline-offset-2 hover:text-accent hover:underline">
      {children}
    </PeekButton>
  )
}

function FilterChip({ label, count, noun, onClear, className }: { label: string; count: number; noun: string; onClear: () => void; className?: string }) {
  return (
    <div className={cn('flex items-center gap-2 text-[12.5px] text-muted', className)}>
      <span className="inline-flex items-center gap-1.5 rounded-lg border border-line-2 bg-panel-2 py-1 pr-1 pl-2.5">
        <span className="capitalize">{label}</span>
        <button onClick={onClear} aria-label="Clear filter" className="grid size-5 place-items-center rounded-md text-dim hover:bg-panel-3 hover:text-fg">
          <X className="size-3" />
        </button>
      </span>
      <span className="num text-dim">
        {count} {noun}
      </span>
    </div>
  )
}

function Row({ icon, children, onCopy }: { icon: ReactNode; children: ReactNode; onCopy?: () => void }) {
  return (
    <div className="group flex items-start gap-2.5 py-1.5 text-[13px] [&>svg]:mt-0.5 [&>svg]:size-4 [&>svg]:shrink-0 [&>svg]:text-dim">
      {icon}
      <span className="min-w-0 flex-1 break-words">{children}</span>
      {onCopy && (
        <button onClick={onCopy} aria-label="Copy" className="text-dim opacity-0 transition group-hover:opacity-100 hover:text-fg">
          <Copy className="size-3.5" />
        </button>
      )}
    </div>
  )
}
