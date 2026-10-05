import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export interface SegmentOption<T> {
  value: T
  label: ReactNode
}

export function Segmented<T extends string | number>({
  value,
  onChange,
  options,
  className,
}: {
  value: T
  onChange: (v: T) => void
  options: SegmentOption<T>[]
  className?: string
}) {
  return (
    <div
      role="radiogroup"
      className={cn('scrollbar-none inline-flex max-w-full gap-0.5 overflow-x-auto rounded-[10px] border border-line bg-panel p-[3px]', className)}
    >
      {options.map((o) => {
        const on = o.value === value
        return (
          <button
            key={String(o.value)}
            type="button"
            role="radio"
            aria-checked={on}
            onClick={() => onChange(o.value)}
            className={cn(
              'inline-flex h-6.5 shrink-0 items-center gap-1.5 rounded-[7px] px-2.5 text-[12.5px] font-medium text-muted transition-colors hover:text-fg [&_svg]:size-3.5',
              on && 'bg-panel-3 text-fg shadow-[inset_0_1px_0_rgb(255_255_255/0.06)] [&_svg]:text-accent',
            )}
          >
            {o.label}
          </button>
        )
      })}
    </div>
  )
}
