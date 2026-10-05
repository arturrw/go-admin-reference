import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

/** Underline tabs for switching sections inside a sheet. */
export function Tabs<T extends string>({
  value,
  onChange,
  tabs,
  className,
}: {
  value: T
  onChange: (v: T) => void
  tabs: { value: T; label: ReactNode; count?: number }[]
  className?: string
}) {
  return (
    <div role="tablist" className={cn('scrollbar-none -mx-4.5 flex shrink-0 gap-1 overflow-x-auto border-b border-line px-4.5', className)}>
      {tabs.map((t) => {
        const on = t.value === value
        return (
          <button
            key={t.value}
            role="tab"
            aria-selected={on}
            onClick={() => onChange(t.value)}
            className={cn(
              'relative flex shrink-0 items-center gap-1.5 px-2.5 pt-1 pb-2.5 text-[13px] font-medium transition-colors',
              on ? 'text-fg after:absolute after:inset-x-1.5 after:-bottom-px after:h-0.5 after:rounded-full after:bg-accent' : 'text-muted hover:text-fg',
            )}
          >
            {t.label}
            {t.count !== undefined && <span className="num rounded-full bg-panel-3 px-1.5 text-[10.5px] text-muted">{t.count}</span>}
          </button>
        )
      })}
    </div>
  )
}
