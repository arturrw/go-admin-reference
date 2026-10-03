import { Link } from '@tanstack/react-router'
import { BellOff, BellRing, Copy, Mail, MapPin, MessageSquarePlus, Phone } from 'lucide-react'
import { type FormEvent, type ReactNode, useState } from 'react'
import { toast } from 'sonner'
import { Donut, SERIES_COLORS } from '@/components/charts/donut'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/input'
import { Avatar, Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Sheet } from '@/components/ui/sheet'
import { Tabs } from '@/components/ui/tabs'
import type { CustomerDetail } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { compact, int, money, monthYear, shortDate, timeAgo } from '@/lib/format'
import { useAddCustomerNote, useCustomer } from '@/lib/queries'
import { cn } from '@/lib/utils'

type Tab = 'overview' | 'orders' | 'products' | 'notes'

/** Everything about one customer: profile, metrics, purchase history and notes. */
export function CustomerSheet({ customerId, onClose }: { customerId: number; onClose: () => void }) {
  const { data, isPending, isError } = useCustomer(customerId)
  const [tab, setTab] = useState<Tab>('overview')
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
            onChange={setTab}
            tabs={[
              { value: 'overview', label: 'Overview' },
              { value: 'orders', label: 'Orders', count: data.orders.length },
              { value: 'products', label: 'Products', count: data.products.length },
              { value: 'notes', label: 'Notes', count: c.notes.length },
            ]}
          />

          {tab === 'overview' && <Overview data={data} />}
          {tab === 'orders' && <OrdersTab data={data} />}
          {tab === 'products' && <ProductsTab data={data} />}
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

function Overview({ data }: { data: CustomerDetail }) {
  const c = data.customer
  const maxMonth = Math.max(1, ...data.monthly.map((m) => m.cents))
  const catTotal = data.categories.reduce((s, x) => s + x.salesCents, 0) || 1
  const copy = (v: string) => navigator.clipboard?.writeText(v).then(() => toast('Copied'))

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
          <dd>{data.stats.firstOrderAt ? shortDate(data.stats.firstOrderAt) : '—'}</dd>
          <dt>Last order</dt>
          <dd>{data.stats.lastOrderAt ? timeAgo(data.stats.lastOrderAt) : '—'}</dd>
          <dt>Last seen</dt>
          <dd>{timeAgo(c.lastSeenAt)}</dd>
          <dt>Acquired via</dt>
          <dd>{c.source}</dd>
          <dt>Refunds</dt>
          <dd className="num">{data.stats.refunds}</dd>
        </dl>
      </Panel>

      <Panel title="Spend · last 6 months">
        <div className="flex h-28 items-end gap-2">
          {data.monthly.map((m) => (
            <div key={m.month} className="flex flex-1 flex-col items-center gap-1.5">
              <span className="num text-[10px] text-dim">{m.cents ? compact(m.cents / 100) : ''}</span>
              <div className="w-full rounded-t-[4px] bg-linear-to-b from-accent to-accent/25" style={{ height: `${Math.max(3, (m.cents / maxMonth) * 72)}px` }} />
              <span className="num text-[10px] text-dim">{new Date(m.month + '-01').toLocaleDateString('en-US', { month: 'short' })}</span>
            </div>
          ))}
        </div>
      </Panel>

      <Panel title="Favourite categories">
        {data.categories.length === 0 ? (
          <p className="text-[13px] text-dim">No purchases yet.</p>
        ) : (
          <div className="flex items-center gap-4">
            <Donut values={data.categories.map((x) => x.salesCents)} size={96} />
            <div className="flex flex-1 flex-col gap-1.5 text-[12.5px]">
              {data.categories.map((x, i) => (
                <div key={x.category} className="flex items-center gap-2">
                  <i className="size-2 rounded-[2px]" style={{ background: SERIES_COLORS[i] }} />
                  <span className="flex-1 text-muted">{x.category}</span>
                  <b className="num text-xs font-medium">{((x.salesCents / catTotal) * 100).toFixed(0)}%</b>
                </div>
              ))}
            </div>
          </div>
        )}
      </Panel>
    </div>
  )
}

function OrdersTab({ data }: { data: CustomerDetail }) {
  if (!data.orders.length) return <p className="py-8 text-center text-muted">No orders yet.</p>
  return (
    <div className="flex flex-col gap-2">
      {data.orders.map((o) => (
        <Link
          key={o.id}
          to="/orders"
          search={{ view: o.id }}
          className="card flex items-center gap-3 px-3.5 py-3 transition-colors hover:border-accent/35"
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
          </div>
          <div className="text-right">
            <b className="num block font-medium">{money(o.totalCents, 2)}</b>
            <small className="text-xs text-dim">{shortDate(o.placedAt)}</small>
          </div>
        </Link>
      ))}
    </div>
  )
}

function ProductsTab({ data }: { data: CustomerDetail }) {
  if (!data.products.length) return <p className="py-8 text-center text-muted">No purchases yet.</p>
  return (
    <div className="grid gap-2.5 sm:grid-cols-2">
      {data.products.map((p) => (
        <div key={p.productId} className="card flex items-center gap-3 p-3">
          <ProductThumb category={p.category} hue={p.hue} src={p.imageUrl} size={48} />
          <div className="min-w-0 flex-1">
            <b className="block truncate font-medium">{p.name}</b>
            <small className="text-xs text-dim">
              {p.category} · {p.qty} bought
            </small>
          </div>
          <b className="num text-[13px] font-medium">{money(p.spentCents)}</b>
        </div>
      ))}
    </div>
  )
}

function NotesTab({ data }: { data: CustomerDetail }) {
  const canWrite = useCan('customers:write')
  const add = useAddCustomerNote()
  const [text, setText] = useState('')
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
        <div key={n.id} className="card p-3.5">
          <div className="mb-1.5 flex items-center gap-2 text-xs text-dim">
            <Avatar name={n.author} size={20} />
            <b className="font-medium text-muted">{n.author}</b>· {timeAgo(n.at)}
          </div>
          <p className="text-[13px] whitespace-pre-wrap">{n.text}</p>
        </div>
      ))}
    </div>
  )
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="card p-4">
      <div className="eyebrow mb-3">{title}</div>
      {children}
    </section>
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
