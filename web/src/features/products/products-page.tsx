import { useNavigate, useSearch } from '@tanstack/react-router'
import {
  Archive,
  CircleCheck,
  CircleX,
  Eye,
  LayoutGrid,
  Package,
  PackageSearch,
  Pencil,
  Plus,
  Rows3,
  Trash2,
  TriangleAlert,
  Upload,
  Warehouse,
  X,
} from 'lucide-react'
import { type ReactNode, useDeferredValue, useState } from 'react'
import { Sparkline } from '@/components/charts/sparkline'
import { Button } from '@/components/ui/button'
import { TableCard } from '@/components/ui/card'
import { SearchInput, Select } from '@/components/ui/input'
import { Checkbox, EmptyState, PageHeader, Skeleton, StatStrip } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { CATEGORY_ICON, ProductThumb } from '@/components/ui/product-thumb'
import { Segmented } from '@/components/ui/segmented'
import { CATEGORIES, type Category, type Product, type ProductStats } from '@/lib/api'
import { int, money } from '@/lib/format'
import { useCan } from '@/lib/auth'
import { useBulkProducts, useProducts } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { ProductSheet } from './product-sheet'

const LOW_STOCK = 15
const SORTS = [
  ['revenue', 'Sort: Revenue'],
  ['sales', 'Sort: Units sold'],
  ['price', 'Sort: Price'],
  ['stock', 'Sort: Stock'],
  ['name', 'Sort: Name'],
] as const

export function ProductsPage() {
  const { edit } = useSearch({ from: '/app/products' })
  const navigate = useNavigate({ from: '/products' })
  const [q, setQ] = useState('')
  const [category, setCategory] = useState<Category | 'all'>('all')
  const [status, setStatus] = useState('all')
  const [sort, setSort] = useState('revenue')
  const [view, setView] = useState<'grid' | 'table'>('grid')
  const [selected, setSelected] = useState<Set<number>>(new Set())

  const canWrite = useCan('products:write')
  const filters = { q: useDeferredValue(q), category, status, sort }
  const { data, isPending } = useProducts(filters)
  const items = data?.items ?? []

  // The sheet is URL-driven (?edit=new | ?edit=<id>) so it can be deep-linked
  // and opened from the command menu.
  const sheetOpen = edit === 'new' ? canWrite : edit !== undefined
  const openSheet = (id: number | 'new') => navigate({ search: { edit: id } })
  const closeSheet = () => navigate({ search: {} })

  const toggle = (id: number) =>
    setSelected((s) => {
      const n = new Set(s)
      if (n.has(id)) n.delete(id)
      else n.add(id)
      return n
    })

  return (
    <>
      <PageHeader title="Products" description={data ? `${data.stats.total} products across ${CATEGORIES.length} categories` : ' '}>
        {canWrite && (
          <>
            <Button>
              <Upload />
              Import CSV
            </Button>
            <Button variant="primary" onClick={() => openSheet('new')}>
              <Plus />
              Add product
            </Button>
          </>
        )}
      </PageHeader>

      {data ? <Stats stats={data.stats} /> : <Skeleton className="mb-4 h-19" />}

      <div className="mb-3.5 flex flex-wrap items-center gap-2.5">
        <SearchInput placeholder="Search name or SKU…" value={q} onChange={(e) => setQ(e.target.value)} />
        <Select className="w-37.5" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="all">All statuses</option>
          <option value="active">Active</option>
          <option value="draft">Draft</option>
          <option value="archived">Archived</option>
        </Select>
        <Select className="w-42.5" value={sort} onChange={(e) => setSort(e.target.value)}>
          {SORTS.map(([v, l]) => (
            <option key={v} value={v}>
              {l}
            </option>
          ))}
        </Select>
        <Segmented
          className="ml-auto"
          value={view}
          onChange={setView}
          options={[
            { value: 'grid', label: <><LayoutGrid />Grid</> },
            { value: 'table', label: <><Rows3 />Table</> },
          ]}
        />
      </div>

      <div className="mb-3.5 flex flex-wrap gap-1.5">
        <Chip on={category === 'all'} onClick={() => setCategory('all')} count={data?.stats.total}>
          All
        </Chip>
        {CATEGORIES.map((c) => {
          const Icon = CATEGORY_ICON[c]
          return (
            <Chip key={c} on={category === c} onClick={() => setCategory(c)} count={data?.stats.byCategory[c] ?? 0}>
              <Icon className="size-3.5" />
              {c}
            </Chip>
          )
        })}
      </div>

      {isPending ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(232px,1fr))] gap-3.5">
          {Array.from({ length: 8 }, (_, i) => (
            <Skeleton key={i} className="h-72" />
          ))}
        </div>
      ) : items.length === 0 ? (
        <div className="card">
          <EmptyState icon={PackageSearch} title="No products match these filters." />
        </div>
      ) : view === 'grid' ? (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(232px,1fr))] gap-3.5">
          {items.map((p) => (
            <ProductCard key={p.id} product={p} onOpen={() => openSheet(p.id)} />
          ))}
        </div>
      ) : (
        <ProductTable items={items} selectable={canWrite} selected={selected} onToggle={toggle} onToggleAll={setSelected} onOpen={openSheet} />
      )}

      <BulkBar selected={selected} onClear={() => setSelected(new Set())} />

      <ProductSheet
        key={String(edit ?? 'closed')}
        productId={typeof edit === 'number' ? edit : undefined}
        open={sheetOpen}
        onClose={closeSheet}
        onCreated={(p) => navigate({ search: { edit: p.id }, replace: true })}
      />
    </>
  )
}

function Stats({ stats }: { stats: ProductStats }) {
  return (
    <StatStrip
      items={[
        { label: 'Total', value: stats.total, icon: Package },
        { label: 'Active', value: stats.active, icon: CircleCheck, tone: 'var(--color-accent)' },
        { label: 'Low stock', value: stats.lowStock, icon: TriangleAlert, tone: 'var(--color-warn)' },
        { label: 'Out of stock', value: stats.outOfStock, icon: CircleX, tone: 'var(--color-danger)' },
        { label: 'Inventory value', value: money(stats.inventoryValueCents), icon: Warehouse },
      ]}
    />
  )
}

function Chip({ on, onClick, count, children }: { on: boolean; onClick: () => void; count?: number; children: ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'inline-flex h-7.5 items-center gap-1.5 rounded-full border px-3 text-[12.5px] font-medium transition-colors',
        on ? 'border-accent/35 bg-accent/14 text-accent' : 'border-line-2 bg-panel text-muted hover:text-fg',
      )}
    >
      {children}
      {count !== undefined && <span className="num text-[11px] opacity-70">{count}</span>}
    </button>
  )
}

function stockLabel(p: Product) {
  if (p.stock === 0) return <span className="text-danger">Out of stock</span>
  if (p.stock < LOW_STOCK) return <span className="text-warn">{p.stock} left</span>
  return `${int(p.stock)} in stock`
}

function StockMeter({ product: p, className }: { product: Product; className?: string }) {
  const color = p.stock === 0 ? 'bg-danger' : p.stock < LOW_STOCK ? 'bg-warn' : 'bg-accent'
  return (
    <div className={cn('h-1.5 overflow-hidden rounded-md bg-panel-3', className)}>
      <i className={cn('block h-full rounded-md', color)} style={{ width: `${Math.max(3, Math.min(100, p.stock / 6.4))}%` }} />
    </div>
  )
}

function ProductCard({ product: p, onOpen }: { product: Product; onOpen: () => void }) {
  return (
    <article
      onClick={onOpen}
      className="card group cursor-pointer overflow-hidden p-0 transition hover:-translate-y-0.5 hover:border-accent/30"
    >
      <ProductThumb
        category={p.category}
        hue={p.hue}
        src={p.images[0]?.url}
        alt={p.images[0]?.alt}
        size={null}
        className="h-37.5 rounded-none border-0 border-b border-b-line"
        iconClassName="relative size-11.5 stroke-[1.25] drop-shadow-[0_8px_18px_currentColor]"
      >
        {/* faint grid texture */}
        <div className="absolute inset-0 bg-[linear-gradient(rgb(255_255_255/.04)_1px,transparent_1px),linear-gradient(90deg,rgb(255_255_255/.04)_1px,transparent_1px)] mask-radial-from-0% mask-radial-to-75% bg-size-[18px_18px]" />
        <span className="absolute top-2.5 left-2.5 z-10">
          <StatusPill status={p.status} />
        </span>
        <span className="num absolute top-2.5 right-2.5 z-10 rounded-md bg-black/45 px-1.5 py-0.5 text-[11px] text-fg/80 backdrop-blur-sm">
          ★ {p.rating.toFixed(1)}
          {p.images.length > 1 && ` · ${p.images.length} photos`}
        </span>
      </ProductThumb>
      <div className="p-3.5">
        <div className="truncate font-medium">{p.name}</div>
        <div className="num mt-0.5 text-[11px] text-dim">
          {p.sku} · {p.category}
        </div>
        <div className="mt-3 flex items-end justify-between">
          <div className="flex items-baseline gap-1.5">
            <span className="num text-lg font-semibold tracking-[-0.02em]">{money(p.priceCents, 2)}</span>
            {p.compareAtCents > 0 && <s className="num text-xs text-dim">{money(p.compareAtCents, 2)}</s>}
          </div>
          <Sparkline data={p.trend} color={`hsl(${p.hue} 85% 68%)`} />
        </div>
        <div className="mt-3 flex justify-between text-[11.5px] text-muted">
          <span>{stockLabel(p)}</span>
          <span className="num">{int(p.sold30d)} sold</span>
        </div>
        <StockMeter product={p} className="mt-1.5" />
      </div>
    </article>
  )
}

function ProductTable({
  items,
  selectable,
  selected,
  onToggle,
  onToggleAll,
  onOpen,
}: {
  items: Product[]
  selectable: boolean
  selected: Set<number>
  onToggle: (id: number) => void
  onToggleAll: (s: Set<number>) => void
  onOpen: (id: number) => void
}) {
  const all = items.length > 0 && items.every((p) => selected.has(p.id))
  const some = !all && items.some((p) => selected.has(p.id))
  return (
    <TableCard>
      <table className="data-table">
        <thead>
          <tr>
            {selectable && (
              <th className="w-8">
                <Checkbox
                  label="Select all"
                  checked={all}
                  indeterminate={some}
                  onChange={() => onToggleAll(all ? new Set() : new Set(items.map((p) => p.id)))}
                />
              </th>
            )}
            <th>Product</th>
            <th>Category</th>
            <th className="num">Price</th>
            <th>Inventory</th>
            <th className="num">Sold (30d)</th>
            <th className="num">Revenue</th>
            <th>Trend</th>
            <th>Status</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {items.map((p) => (
            <tr key={p.id} className={cn(selected.has(p.id) && 'bg-accent/5')}>
              {selectable && (
                <td>
                  <Checkbox label={`Select ${p.name}`} checked={selected.has(p.id)} onChange={() => onToggle(p.id)} />
                </td>
              )}
              <td>
                <div className="flex items-center gap-2.5">
                  <ProductThumb category={p.category} hue={p.hue} src={p.images[0]?.url} size={36} />
                  <div>
                    <b className="block font-medium">{p.name}</b>
                    <small className="num block text-xs text-dim">{p.sku}</small>
                  </div>
                </div>
              </td>
              <td className="text-muted">{p.category}</td>
              <td className="num">{money(p.priceCents, 2)}</td>
              <td>
                <div className="min-w-30 text-[12.5px]">
                  {stockLabel(p)}
                  <StockMeter product={p} className="mt-1 h-1" />
                </div>
              </td>
              <td className="num">{int(p.sold30d)}</td>
              <td className="num">{money(p.sold30d * p.priceCents)}</td>
              <td>
                <Sparkline data={p.trend} color={`hsl(${p.hue} 85% 68%)`} />
              </td>
              <td>
                <StatusPill status={p.status} />
              </td>
              <td className="num">
                <Button variant="ghost" size="icon-sm" aria-label={`${selectable ? 'Edit' : 'View'} ${p.name}`} onClick={() => onOpen(p.id)}>
                  {selectable ? <Pencil /> : <Eye />}
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </TableCard>
  )
}

function BulkBar({ selected, onClear }: { selected: Set<number>; onClear: () => void }) {
  const bulk = useBulkProducts()
  const n = selected.size
  const run = (action: 'publish' | 'archive' | 'delete') => bulk.mutate({ ids: [...selected], action }, { onSuccess: onClear })
  return (
    <div
      className={cn(
        'fixed bottom-6 left-1/2 z-35 flex -translate-x-1/2 items-center gap-2.5 rounded-[14px] border border-line-2 bg-panel-2 py-2 pr-2 pl-4 shadow-float transition-all duration-250',
        n ? 'translate-y-0 opacity-100' : 'pointer-events-none translate-y-[120%] opacity-0',
      )}
    >
      <b className="num text-[13px] font-medium text-accent">{n}</b>
      <span className="mr-2 text-[13px] text-muted">selected</span>
      <Button size="sm" disabled={bulk.isPending} onClick={() => run('publish')}>
        <Eye />
        Publish
      </Button>
      <Button size="sm" disabled={bulk.isPending} onClick={() => run('archive')}>
        <Archive />
        Archive
      </Button>
      <Button size="sm" variant="danger" disabled={bulk.isPending} onClick={() => run('delete')}>
        <Trash2 />
        Delete
      </Button>
      <Button variant="ghost" size="icon-sm" aria-label="Clear selection" onClick={onClear}>
        <X />
      </Button>
    </div>
  )
}
