import { ArrowDownRight, ArrowUpRight, Check, type LucideIcon, Minus } from 'lucide-react'
import type { CSSProperties, ReactNode } from 'react'
import { hueOf, initials } from '@/lib/format'
import { cn } from '@/lib/utils'

export function Delta({ value, className }: { value: number; className?: string }) {
  const up = value >= 0
  const Icon = up ? ArrowUpRight : ArrowDownRight
  return (
    <span
      className={cn(
        'num inline-flex items-center gap-0.5 rounded-md px-1.5 py-0.5 text-xs font-medium',
        up ? 'bg-accent/14 text-accent' : 'bg-danger/12 text-danger',
        className,
      )}
    >
      <Icon className="size-3" />
      {Math.abs(value).toFixed(1)}%
    </span>
  )
}

/** Tinted square holding an icon; outlines itself on hover (see .icon-tile). */
export function IconTile({ icon: Icon, color, size = 26, className }: { icon: LucideIcon; color: string; size?: number; className?: string }) {
  const icon = Math.round(size * 0.52)
  return (
    <span className={cn('icon-tile rounded-lg', className)} style={{ '--tile': color, width: size, height: size } as CSSProperties}>
      <Icon style={{ width: icon, height: icon }} />
    </span>
  )
}

export function Avatar({ name, size = 30 }: { name: string; size?: number }) {
  const h = hueOf(name)
  const style: CSSProperties = {
    width: size,
    height: size,
    fontSize: Math.round(size * 0.38),
    background: `linear-gradient(135deg, hsl(${h} 85% 72%), hsl(${h + 40} 80% 58%))`,
  }
  return (
    <div className="grid shrink-0 place-items-center rounded-full font-semibold text-accent-ink" style={style}>
      {initials(name)}
    </div>
  )
}

export function Switch({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      onClick={() => onChange(!checked)}
      className={cn(
        'relative h-[21px] w-9 shrink-0 rounded-full border transition-colors',
        checked ? 'border-transparent bg-accent' : 'border-line-2 bg-panel-3',
      )}
    >
      <span
        className={cn(
          'absolute top-0.5 left-0.5 size-[15px] rounded-full transition-transform duration-200',
          checked ? 'translate-x-[15px] bg-accent-ink' : 'bg-[#c9ced6]',
        )}
      />
    </button>
  )
}

export function Checkbox({
  checked,
  indeterminate,
  onChange,
  label,
}: {
  checked: boolean
  indeterminate?: boolean
  onChange: () => void
  label: string
}) {
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={indeterminate ? 'mixed' : checked}
      aria-label={label}
      onClick={onChange}
      className={cn(
        'inline-grid size-4 place-items-center rounded-[5px] border-[1.5px] align-middle',
        checked || indeterminate ? 'border-accent bg-accent text-accent-ink' : 'border-line-2',
      )}
    >
      {indeterminate ? <Minus className="size-[11px] stroke-3" /> : checked && <Check className="size-[11px] stroke-3" />}
    </button>
  )
}

export function EmptyState({ icon: Icon, title }: { icon: LucideIcon; title: string }) {
  return (
    <div className="flex flex-col items-center gap-2 px-5 py-14 text-center text-muted">
      <Icon className="size-7 text-dim" />
      <div>{title}</div>
    </div>
  )
}

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn('animate-pulse rounded-lg bg-panel-2', className)} />
}

export function PageHeader({ title, description, children }: { title: ReactNode; description?: ReactNode; children?: ReactNode }) {
  return (
    <div className="mb-5.5 flex flex-wrap items-end gap-4">
      <div>
        <h1 className="text-[26px] font-semibold tracking-[-0.025em]">{title}</h1>
        {description && <p className="mt-1 text-muted">{description}</p>}
      </div>
      {children && <div className="ml-auto flex flex-wrap items-center gap-2">{children}</div>}
    </div>
  )
}

/** Horizontal strip of headline numbers (Products, Request log). */
export function StatStrip({ items }: { items: { label: ReactNode; value: ReactNode; icon?: LucideIcon; tone?: string }[] }) {
  return (
    <div className="mb-4 grid grid-cols-1 overflow-hidden rounded-[14px] border border-line bg-panel sm:grid-cols-2 lg:grid-cols-5">
      {items.map(({ label, value, icon: Icon, tone }, i) => (
        <div key={i} className="border-line px-4.5 py-3.5 max-lg:border-b lg:border-r lg:last:border-r-0">
          <div className="flex items-center gap-1.5 text-xs text-muted">
            {Icon && <Icon className="size-3.5" style={tone ? { color: tone } : undefined} />}
            {label}
          </div>
          <div className="num mt-1 text-[22px] font-semibold tracking-[-0.03em]" style={tone && !Icon ? { color: tone } : undefined}>
            {value}
          </div>
        </div>
      ))}
    </div>
  )
}
