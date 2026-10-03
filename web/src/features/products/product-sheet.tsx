import { Check, Lock, Trash2 } from 'lucide-react'
import { type FormEvent, type ReactNode, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Field, Input, Select, Textarea } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/misc'
import { Segmented } from '@/components/ui/segmented'
import { Sheet } from '@/components/ui/sheet'
import { TagInput } from '@/components/ui/tag-input'
import { ApiError, CATEGORIES, type Category, type Product, type ProductStatus } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { int, money, timeAgo } from '@/lib/format'
import { useDeleteProduct, useProduct, useSaveProduct } from '@/lib/queries'
import { ImageGallery } from './image-gallery'

interface FormState {
  name: string
  sku: string
  category: Category
  vendor: string
  tags: string[]
  price: string
  compareAt: string
  cost: string
  stock: string
  weight: string
  status: ProductStatus
  description: string
}

const emptyForm = (): FormState => ({
  name: '',
  sku: `NEW-${Math.floor(1000 + Math.random() * 9000)}`,
  category: 'Audio',
  vendor: '',
  tags: [],
  price: '49.99',
  compareAt: '',
  cost: '',
  stock: '100',
  weight: '',
  status: 'draft',
  description: '',
})

const dollars = (cents: number) => (cents ? (cents / 100).toFixed(2) : '')
const toCents = (s: string) => Math.round(parseFloat(s || '0') * 100) || 0

const toForm = (p: Product): FormState => ({
  name: p.name,
  sku: p.sku,
  category: p.category,
  vendor: p.vendor,
  tags: p.tags,
  price: dollars(p.priceCents),
  compareAt: dollars(p.compareAtCents),
  cost: dollars(p.costCents),
  stock: String(p.stock),
  weight: p.weightGrams ? String(p.weightGrams) : '',
  status: p.status,
  description: p.description,
})

/**
 * Create/edit sheet. `productId` undefined = create. Data is fetched by id so
 * image changes show up immediately.
 */
export function ProductSheet({ productId, open, onClose, onCreated }: { productId?: number; open: boolean; onClose: () => void; onCreated: (p: Product) => void }) {
  const { data: product, isPending } = useProduct(productId)
  return (
    <Sheet
      open={open}
      onOpenChange={(o) => !o && onClose()}
      size="lg"
      title={productId ? (product?.name ?? 'Product') : 'New product'}
      description={product ? `${product.sku} · ${product.category} · updated ${timeAgo(product.updatedAt)}` : 'Draft products are hidden from the storefront'}
    >
      {productId && isPending ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="aspect-[4/3] w-full" />
          <Skeleton className="h-60" />
        </div>
      ) : (
        <ProductForm key={product?.id ?? 'new'} product={product} onClose={onClose} onCreated={onCreated} />
      )}
    </Sheet>
  )
}

function ProductForm({ product, onClose, onCreated }: { product?: Product; onClose: () => void; onCreated: (p: Product) => void }) {
  const canWrite = useCan('products:write')
  const [form, setForm] = useState<FormState>(() => (product ? toForm(product) : emptyForm()))
  const save = useSaveProduct()
  const remove = useDeleteProduct()
  const errors = save.error instanceof ApiError ? save.error.fields : undefined
  const set = <K extends keyof FormState>(k: K, v: FormState[K]) => setForm((f) => ({ ...f, [k]: v }))

  const price = toCents(form.price)
  const cost = toCents(form.cost)
  const margin = price > 0 && cost > 0 ? ((price - cost) / price) * 100 : null

  const submit = (e: FormEvent) => {
    e.preventDefault()
    save.mutate(
      {
        id: product?.id,
        input: {
          name: form.name,
          sku: form.sku,
          category: form.category,
          vendor: form.vendor,
          tags: form.tags,
          priceCents: price,
          compareAtCents: toCents(form.compareAt),
          costCents: cost,
          stock: parseInt(form.stock || '0', 10),
          weightGrams: parseInt(form.weight || '0', 10),
          status: form.status,
          description: form.description,
        },
      },
      { onSuccess: (p) => (product ? onClose() : onCreated(p)) },
    )
  }

  return (
    <>
      {!canWrite && (
        <div className="flex items-center gap-2 rounded-xl border border-line bg-panel-2 px-3.5 py-2.5 text-[12.5px] text-muted">
          <Lock className="size-4 text-dim" />
          Read-only — your role can view products but not change them.
        </div>
      )}

      {product ? (
        <>
          <ImageGallery product={product} editable={canWrite} />
          <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
            {[
              ['Sold (30d)', int(product.sold30d)],
              ['Revenue', money(product.sold30d * product.priceCents)],
              ['Rating', `★ ${product.rating.toFixed(1)}`],
              ['Created', timeAgo(product.createdAt)],
            ].map(([k, v]) => (
              <div key={k} className="card px-3 py-2.5">
                <div className="eyebrow">{k}</div>
                <div className="num mt-0.5 text-[15px]">{v}</div>
              </div>
            ))}
          </div>
        </>
      ) : (
        <p className="rounded-xl border border-dashed border-line-2 px-3.5 py-3 text-[12.5px] text-dim">Save the product first — then you can upload images.</p>
      )}

      <form id="product-form" onSubmit={submit}>
        <fieldset disabled={!canWrite || save.isPending} className="flex flex-col gap-5">
          <Section title="Details">
            <Field label="Name" error={errors?.name}>
              <Input value={form.name} onChange={(e) => set('name', e.target.value)} placeholder="e.g. Aero Buds Pro" autoFocus={!product} />
            </Field>
            <div className="grid gap-3.5 sm:grid-cols-2">
              <Field label="SKU" error={errors?.sku}>
                <Input className="num" value={form.sku} onChange={(e) => set('sku', e.target.value)} />
              </Field>
              <Field label="Status">
                <Segmented<ProductStatus>
                  value={form.status}
                  onChange={(v) => set('status', v)}
                  options={(['active', 'draft', 'archived'] as const).map((s) => ({ value: s, label: s }))}
                  className="self-start"
                />
              </Field>
            </div>
            <Field label="Description" error={errors?.description} hint="Shown on the storefront product page.">
              <Textarea rows={4} value={form.description} onChange={(e) => set('description', e.target.value)} />
            </Field>
          </Section>

          <Section title="Pricing">
            <div className="grid gap-3.5 sm:grid-cols-3">
              <Field label="Price, USD" error={errors?.priceCents}>
                <Input className="num" type="number" step="0.01" min="0" value={form.price} onChange={(e) => set('price', e.target.value)} />
              </Field>
              <Field label="Compare-at price" error={errors?.compareAtCents} hint="Shown struck through">
                <Input className="num" type="number" step="0.01" min="0" value={form.compareAt} onChange={(e) => set('compareAt', e.target.value)} placeholder="—" />
              </Field>
              <Field label="Cost per item" error={errors?.costCents} hint={margin !== null ? `Margin ${margin.toFixed(1)}%` : 'Not shown to customers'}>
                <Input className="num" type="number" step="0.01" min="0" value={form.cost} onChange={(e) => set('cost', e.target.value)} placeholder="—" />
              </Field>
            </div>
          </Section>

          <Section title="Inventory & shipping">
            <div className="grid gap-3.5 sm:grid-cols-2">
              <Field label="Stock" error={errors?.stock}>
                <Input className="num" type="number" min="0" value={form.stock} onChange={(e) => set('stock', e.target.value)} />
              </Field>
              <Field label="Weight, g" error={errors?.weightGrams}>
                <Input className="num" type="number" min="0" value={form.weight} onChange={(e) => set('weight', e.target.value)} placeholder="—" />
              </Field>
            </div>
          </Section>

          <Section title="Organization">
            <div className="grid gap-3.5 sm:grid-cols-2">
              <Field label="Category" error={errors?.category}>
                <Select value={form.category} onChange={(e) => set('category', e.target.value as Category)}>
                  {CATEGORIES.map((c) => (
                    <option key={c}>{c}</option>
                  ))}
                </Select>
              </Field>
              <Field label="Vendor" error={errors?.vendor}>
                <Input value={form.vendor} onChange={(e) => set('vendor', e.target.value)} placeholder="e.g. Aero Labs" />
              </Field>
            </div>
            <Field label="Tags" error={errors?.tags} hint="Enter or comma to add · up to 10">
              <TagInput value={form.tags} onChange={(v) => set('tags', v)} disabled={!canWrite} />
            </Field>
          </Section>
        </fieldset>
      </form>

      <FooterActions>
        {product && canWrite && (
          <Button variant="danger" className="mr-auto" disabled={remove.isPending} onClick={() => remove.mutate(product.id, { onSuccess: onClose })}>
            <Trash2 />
            Delete
          </Button>
        )}
        <Button onClick={onClose}>{canWrite ? 'Cancel' : 'Close'}</Button>
        {canWrite && (
          <Button variant="primary" type="submit" form="product-form" disabled={save.isPending}>
            <Check />
            {save.isPending ? 'Saving…' : product ? 'Save' : 'Create product'}
          </Button>
        )}
      </FooterActions>
    </>
  )
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3.5">
      <div className="eyebrow border-b border-line pb-1.5">{title}</div>
      {children}
    </section>
  )
}

/** Sticky footer inside the scroll area (keeps actions next to their form state). */
function FooterActions({ children }: { children: ReactNode }) {
  return <div className="sticky -bottom-4.5 -mx-4.5 mt-auto -mb-4.5 flex flex-wrap justify-end gap-2 border-t border-line bg-panel px-4.5 py-3.5">{children}</div>
}
