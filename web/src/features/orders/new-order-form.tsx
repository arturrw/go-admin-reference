import { Plus, ShoppingCart, X } from 'lucide-react'
import { type FormEvent, useDeferredValue, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Field, Input, SearchInput, Select } from '@/components/ui/input'
import { Avatar } from '@/components/ui/misc'
import { Sheet } from '@/components/ui/sheet'
import { ApiError, type Order, PAYMENT_METHODS } from '@/lib/api'
import { money } from '@/lib/format'
import { useCreateOrder, useCustomers, useProducts } from '@/lib/queries'
import { cn } from '@/lib/utils'

interface Line {
  key: number
  productId: number | ''
  qty: number
}

/** Enters a sale by hand, e.g. a phone order: prices and stock come from the catalogue. */
export function NewOrderForm({ onClose, onCreated }: { onClose: () => void; onCreated: (o: Order) => void }) {
  const [customerId, setCustomerId] = useState<number | null>(null)
  const [q, setQ] = useState('')
  const dq = useDeferredValue(q)
  const [payment, setPayment] = useState(PAYMENT_METHODS[0])
  const [lines, setLines] = useState<Line[]>([{ key: 1, productId: '', qty: 1 }])
  const create = useCreateOrder()
  const customers = useCustomers({ q: dq })
  const products = useProducts({ status: 'active', sort: 'name' })
  const errors = create.error instanceof ApiError ? create.error.fields : undefined

  const sellable = (products.data?.items ?? []).filter((p) => p.stock > 0)
  const price = (id: number | '') => sellable.find((p) => p.id === id)?.priceCents ?? 0
  const total = lines.reduce((s, l) => s + price(l.productId) * l.qty, 0)
  const picked = customers.data?.items.find((c) => c.id === customerId)
  const ready = customerId !== null && lines.length > 0 && lines.every((l) => l.productId !== '' && l.qty >= 1)

  const setLine = (key: number, patch: Partial<Line>) => setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...patch } : l)))

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (!ready) return
    create.mutate(
      { customerId, payment, items: lines.map((l) => ({ productId: l.productId as number, qty: l.qty })) },
      { onSuccess: onCreated },
    )
  }

  return (
    <Sheet
      open
      onOpenChange={(v) => !v && onClose()}
      title="New order"
      description="For orders taken by phone or in person. Prices and stock come from the catalogue; the order starts as pending."
      footer={
        <>
          <span className="num mr-auto text-[13px] text-muted max-sm:w-full">
            Total <b className="text-base font-semibold text-fg">{money(total, 2)}</b>
          </span>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" className="max-sm:w-full" type="submit" form="new-order-form" disabled={create.isPending || !ready}>
            <ShoppingCart />
            {create.isPending ? 'Creating…' : 'Create order'}
          </Button>
        </>
      }
    >
      <form id="new-order-form" onSubmit={submit} className="flex flex-col gap-4">
        <Field label="Customer" error={errors?.customerId}>
          {picked ? (
            <div className="card flex items-center gap-2.5 p-2.5">
              <Avatar name={picked.name} size={32} />
              <div className="min-w-0 flex-1">
                <b className="block truncate font-medium">{picked.name}</b>
                <small className="block truncate text-xs text-dim">{picked.email}</small>
              </div>
              <Button size="icon" variant="ghost" aria-label="Choose another customer" onClick={() => setCustomerId(null)}>
                <X />
              </Button>
            </div>
          ) : (
            <>
              <SearchInput className="max-w-none" placeholder="Search customers by name or email" value={q} onChange={(e) => setQ(e.target.value)} />
              <div role="listbox" aria-label="Customers" className="card max-h-52 overflow-y-auto">
                {(customers.data?.items ?? []).slice(0, 30).map((c) => (
                  <button
                    key={c.id}
                    type="button"
                    role="option"
                    aria-selected={false}
                    onClick={() => setCustomerId(c.id)}
                    className="flex w-full items-center gap-2.5 border-b border-line px-3 py-2 text-left last:border-0 hover:bg-panel-2"
                  >
                    <Avatar name={c.name} size={28} />
                    <span className="min-w-0 flex-1 truncate">{c.name}</span>
                    <small className="truncate text-xs text-dim max-sm:hidden">{c.email}</small>
                  </button>
                ))}
                {customers.data?.items.length === 0 && <p className="p-3 text-[13px] text-dim">No customer matches “{dq}”.</p>}
              </div>
            </>
          )}
        </Field>

        <div className="flex flex-col gap-2.5">
          <div className="eyebrow">Products</div>
          {lines.map((l, i) => (
            <div key={l.key} className="grid grid-cols-[1fr_72px_auto] items-start gap-2">
              <Select aria-label={`Product ${i + 1}`} value={l.productId} onChange={(e) => setLine(l.key, { productId: e.target.value ? Number(e.target.value) : '' })}>
                <option value="">Choose a product…</option>
                {sellable.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} · {money(p.priceCents, 2)} · {p.stock} in stock
                  </option>
                ))}
              </Select>
              <Input
                aria-label={`Quantity ${i + 1}`}
                className="num"
                type="number"
                min={1}
                max={99}
                value={l.qty}
                onChange={(e) => setLine(l.key, { qty: Math.max(1, Math.min(99, Number(e.target.value) || 1)) })}
              />
              <Button
                size="icon"
                variant="ghost"
                aria-label={`Remove product ${i + 1}`}
                className={cn(lines.length === 1 && 'invisible')}
                onClick={() => setLines((ls) => ls.filter((x) => x.key !== l.key))}
              >
                <X />
              </Button>
            </div>
          ))}
          {errors?.items && <span className="text-xs text-danger">{errors.items}</span>}
          <Button className="self-start" onClick={() => setLines((ls) => [...ls, { key: Math.max(...ls.map((l) => l.key)) + 1, productId: '', qty: 1 }])}>
            <Plus />
            Add product
          </Button>
        </div>

        <Field label="Payment" error={errors?.payment}>
          <Select value={payment} onChange={(e) => setPayment(e.target.value)}>
            {PAYMENT_METHODS.map((m) => (
              <option key={m}>{m}</option>
            ))}
          </Select>
        </Field>
      </form>
    </Sheet>
  )
}
