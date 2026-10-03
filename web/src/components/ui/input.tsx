import { ChevronDown, Search } from 'lucide-react'
import type { ComponentProps, ReactNode } from 'react'
import { cn } from '@/lib/utils'

const base =
  'w-full rounded-[10px] border border-line-2 bg-panel px-3 text-sm outline-none transition placeholder:text-dim focus:border-accent/55 focus:ring-3 focus:ring-accent/14 aria-invalid:border-danger/60'

export function Input({ className, ...props }: ComponentProps<'input'>) {
  return <input className={cn(base, 'h-9', className)} {...props} />
}

export function Textarea({ className, ...props }: ComponentProps<'textarea'>) {
  return <textarea className={cn(base, 'min-h-21 resize-y py-2.5', className)} {...props} />
}

export function Select({ className, ...props }: ComponentProps<'select'>) {
  return (
    <div className={cn('relative', className)}>
      <select className={cn(base, 'h-9 appearance-none pr-8 [&_option]:bg-panel')} {...props} />
      <ChevronDown className="pointer-events-none absolute top-2.5 right-3 size-4 text-muted" />
    </div>
  )
}

export function SearchInput({ className, ...props }: ComponentProps<'input'>) {
  return (
    <label className={cn('relative block min-w-55 flex-1 sm:max-w-95', className)}>
      <Search className="pointer-events-none absolute top-2.5 left-3 size-4 text-dim" />
      <Input className="pl-9" {...props} />
    </label>
  )
}

export function Field({ label, hint, error, children }: { label: string; hint?: ReactNode; error?: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[12.5px] font-medium text-muted">{label}</span>
      {children}
      {error ? <span className="text-xs text-danger">{error}</span> : hint && <span className="text-xs text-dim">{hint}</span>}
    </div>
  )
}
