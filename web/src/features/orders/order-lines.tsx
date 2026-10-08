import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input, Select } from '@/components/ui/input'
import type { Product } from '@/lib/api'
import { money } from '@/lib/format'
import { cn } from '@/lib/utils'

export interface Line {
  key: number
  productId: number | ''
  qty: number
}

let nextKey = 1
export const newLine = (productId: number | '' = '', qty = 1): Line => ({ key: nextKey++, productId, qty })

/** Total of the lines at the given unit prices. */
export const linesTotal = (lines: Line[], price: (id: number | '') => number) => lines.reduce((s, l) => s + price(l.productId) * l.qty, 0)

/** The product rows of an order form: pick a product, a quantity, add or remove lines. */
export function OrderLines({
  lines,
  setLines,
  products,
  error,
}: {
  lines: Line[]
  setLines: (fn: (ls: Line[]) => Line[]) => void
  products: Product[]
  error?: string
}) {
  const setLine = (key: number, patch: Partial<Line>) => setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...patch } : l)))
  return (
    <div className="flex flex-col gap-2.5">
      <div className="eyebrow">Products</div>
      {lines.map((l, i) => (
        <div key={l.key} className="grid grid-cols-[1fr_72px_auto] items-start gap-2">
          <Select aria-label={`Product ${i + 1}`} value={l.productId} onChange={(e) => setLine(l.key, { productId: e.target.value ? Number(e.target.value) : '' })}>
            <option value="">Choose a product…</option>
            {products.map((p) => (
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
      {error && <span className="text-xs text-danger">{error}</span>}
      <Button className="self-start" onClick={() => setLines((ls) => [...ls, newLine()])}>
        <Plus />
        Add product
      </Button>
    </div>
  )
}
