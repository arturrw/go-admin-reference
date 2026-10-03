import type { ComponentProps, ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function Card({ className, ...props }: ComponentProps<'div'>) {
  return <div className={cn('card', className)} {...props} />
}

export function CardHeader({ title, sub, children }: { title: ReactNode; sub?: ReactNode; children?: ReactNode }) {
  return (
    <div className="mb-3.5 flex items-center gap-2.5">
      <h3 className="text-sm font-medium">{title}</h3>
      {sub && <span className="text-[12.5px] text-dim">· {sub}</span>}
      {children && <div className="ml-auto flex items-center gap-1.5">{children}</div>}
    </div>
  )
}

/** Card wrapping an edge-to-edge table. */
export function TableCard({ className, children }: { className?: string; children: ReactNode }) {
  return <div className={cn('card overflow-x-auto p-0', className)}>{children}</div>
}
