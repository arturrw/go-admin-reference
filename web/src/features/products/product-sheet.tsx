import { Check, Trash2 } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Field, Input, Select, Textarea } from '@/components/ui/input'
import { ProductThumb } from '@/components/ui/product-thumb'
import { Segmented } from '@/components/ui/segmented'
import { Sheet } from '@/components/ui/sheet'
import { ApiError, CATEGORIES, type Category, type Product, type ProductStatus } from '@/lib/api'
import { int, money, timeAgo } from '@/lib/format'
import { useDeleteProduct, useSaveProduct } from '@/lib/queries'

interface FormState {
  name: string
  sku: string
  category: Category
  price: string
  stock: string
  status: ProductStatus
  description: string
}

const emptyForm = (): FormState => ({
  name: '',
  sku: `NEW-${Math.floor(1000 + Math.random() * 9000)}`,
  category: 'Audio',
  price: '49.99',
  stock: '100',
  status: 'draft',
  description: '',
})

const toForm = (p: Product): FormState => ({
  name: p.name,
  sku: p.sku,
  category: p.category,
  price: (p.priceCents / 100).toFixed(2),
  stock: String(p.stock),
  status: p.status,
  description: p.description,
})

/** Create/edit form. Mount with a `key` so the form resets per product. */
export function ProductSheet({ product, open, onClose }: { product?: Product; open: boolean; onClose: () => void }) {
  const [form, setForm] = useState<FormState>(() => (product ? toForm(product) : emptyForm()))
  const save = useSaveProduct()
  const remove = useDeleteProduct()
  const fieldErrors = save.error instanceof ApiError ? save.error.fields : undefined
  const set = <K extends keyof FormState>(k: K, v: FormState[K]) => setForm((f) => ({ ...f, [k]: v }))

  const submit = (e: FormEvent) => {
    e.preventDefault()
    save.mutate(
      {
        id: product?.id,
        input: {
          name: form.name,
          sku: form.sku,
          category: form.category,
          priceCents: Math.round(parseFloat(form.price || '0') * 100),
          stock: parseInt(form.stock || '0', 10),
          status: form.status,
          description: form.description,
        },
      },
      { onSuccess: onClose },
    )
  }

  return (
    <Sheet
      open={open}
      onOpenChange={(o) => !o && onClose()}
      title={product ? 'Edit product' : 'New product'}
      description={product ? `${product.sku} · ${product.category}` : 'Draft products are hidden from the storefront'}
      footer={
        <>
          {product && (
            <Button variant="danger" className="mr-auto" disabled={remove.isPending} onClick={() => remove.mutate(product.id, { onSuccess: onClose })}>
              <Trash2 />
              Delete
            </Button>
          )}
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" type="submit" form="product-form" disabled={save.isPending}>
            <Check />
            {save.isPending ? 'Saving…' : 'Save'}
          </Button>
        </>
      }
    >
      {product && (
        <>
          <ProductThumb category={form.category} hue={product.hue} size={null} className="h-42 rounded-[14px]" iconClassName="size-13 stroke-[1.25]" />
          <div className="grid grid-cols-2 gap-2.5">
            {[
              ['Sold (30d)', int(product.sold30d)],
              ['Revenue', money(product.sold30d * product.priceCents)],
              ['Rating', `★ ${product.rating.toFixed(1)}`],
              ['Updated', timeAgo(product.updatedAt)],
            ].map(([k, v]) => (
              <div key={k} className="card px-3 py-2.5">
                <div className="eyebrow">{k}</div>
                <div className="num mt-0.5 text-[15px]">{v}</div>
              </div>
            ))}
          </div>
        </>
      )}

      <form id="product-form" onSubmit={submit} className="flex flex-col gap-4">
        <Field label="Name" error={fieldErrors?.name}>
          <Input value={form.name} onChange={(e) => set('name', e.target.value)} placeholder="e.g. Aero Buds Pro" autoFocus={!product} />
        </Field>
        <div className="grid grid-cols-1 gap-3.5 sm:grid-cols-2">
          <Field label="SKU" error={fieldErrors?.sku}>
            <Input className="num" value={form.sku} onChange={(e) => set('sku', e.target.value)} />
          </Field>
          <Field label="Category" error={fieldErrors?.category}>
            <Select value={form.category} onChange={(e) => set('category', e.target.value as Category)}>
              {CATEGORIES.map((c) => (
                <option key={c}>{c}</option>
              ))}
            </Select>
          </Field>
          <Field label="Price, USD" error={fieldErrors?.priceCents}>
            <Input className="num" type="number" step="0.01" min="0" value={form.price} onChange={(e) => set('price', e.target.value)} />
          </Field>
          <Field label="Stock" error={fieldErrors?.stock}>
            <Input className="num" type="number" min="0" value={form.stock} onChange={(e) => set('stock', e.target.value)} />
          </Field>
        </div>
        <Field label="Status">
          <Segmented<ProductStatus>
            value={form.status}
            onChange={(v) => set('status', v)}
            options={(['active', 'draft', 'archived'] as const).map((s) => ({ value: s, label: s }))}
            className="self-start"
          />
        </Field>
        <Field label="Description" hint="Shown on the storefront product page.">
          <Textarea value={form.description} onChange={(e) => set('description', e.target.value)} />
        </Field>
      </form>
    </Sheet>
  )
}
