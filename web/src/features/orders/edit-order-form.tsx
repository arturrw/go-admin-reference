import { Save } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Sheet } from '@/components/ui/sheet'
import { ApiError, type Order } from '@/lib/api'
import { money } from '@/lib/format'
import { useEditOrderItems, useProducts } from '@/lib/queries'
import { type Line, linesTotal, newLine, OrderLines } from './order-lines'

/** Changes what a pending order contains. Stock moves by the difference; a product already in the order keeps its price. */
export function EditOrderForm({ order, onClose }: { order: Order; onClose: () => void }) {
  const [lines, setLines] = useState<Line[]>(() => order.items.map((it) => newLine(it.productId, it.qty)))
  const edit = useEditOrderItems()
  const products = useProducts({ status: 'active', sort: 'name' })
  const error = edit.error instanceof ApiError ? edit.error.fields?.items : undefined

  // Sellable now, plus whatever is already in the order (it may have sold out since).
  const inOrder = new Set(order.items.map((it) => it.productId))
  const options = (products.data?.items ?? []).filter((p) => p.stock > 0 || inOrder.has(p.id))
  // A product already in the order keeps the price it was sold at.
  const price = (id: number | '') => order.items.find((it) => it.productId === id)?.priceCents ?? options.find((p) => p.id === id)?.priceCents ?? 0
  const total = linesTotal(lines, price)
  const ready = lines.every((l) => l.productId !== '' && l.qty >= 1)

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (!ready) return
    edit.mutate({ id: order.id, items: lines.map((l) => ({ productId: l.productId as number, qty: l.qty })) }, { onSuccess: onClose })
  }

  return (
    <Sheet
      open
      onOpenChange={(v) => !v && onClose()}
      title={`Edit order #${order.id}`}
      description="Only a pending order can change. Stock moves by the difference; products already in the order keep their price."
      footer={
        <>
          <span className="num mr-auto text-[13px] text-muted max-sm:w-full">
            Total <b className="text-base font-semibold text-fg">{money(total, 2)}</b>
          </span>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" className="max-sm:w-full" type="submit" form="edit-order-form" disabled={edit.isPending || !ready}>
            <Save />
            {edit.isPending ? 'Saving…' : 'Save changes'}
          </Button>
        </>
      }
    >
      <form id="edit-order-form" onSubmit={submit} className="flex flex-col gap-4">
        <OrderLines lines={lines} setLines={setLines} products={options} error={error} />
      </form>
    </Sheet>
  )
}
