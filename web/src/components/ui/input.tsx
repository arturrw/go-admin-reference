import { ChevronDown, Search } from 'lucide-react'
import { type ComponentProps, createContext, type ReactNode, useContext, useId } from 'react'
import { cn } from '@/lib/utils'

const base =
  'w-full rounded-[10px] border border-line-2 bg-panel px-3 text-sm outline-none transition placeholder:text-dim focus:border-accent/55 focus:ring-3 focus:ring-accent/14 aria-invalid:border-danger/60 disabled:cursor-not-allowed disabled:opacity-60'

// Field provides an id so the control inside is labelled without wiring htmlFor by hand.
const FieldContext = createContext<{ id: string; invalid: boolean } | null>(null)

function useFieldProps(props: { id?: string }) {
  const field = useContext(FieldContext)
  return { id: props.id ?? field?.id, 'aria-invalid': field?.invalid || undefined }
}

export function Input({ className, ...props }: ComponentProps<'input'>) {
  return <input {...useFieldProps(props)} className={cn(base, 'h-9', className)} {...props} />
}

export function Textarea({ className, ...props }: ComponentProps<'textarea'>) {
  return <textarea {...useFieldProps(props)} className={cn(base, 'min-h-21 resize-y py-2.5', className)} {...props} />
}

export function Select({ className, ...props }: ComponentProps<'select'>) {
  const field = useFieldProps(props)
  return (
    <div className={cn('relative', className)}>
      <select {...field} className={cn(base, 'h-9 appearance-none pr-8 [&_option]:bg-panel')} {...props} />
      <ChevronDown className="pointer-events-none absolute top-2.5 right-3 size-4 text-muted" />
    </div>
  )
}

export function SearchInput({ className, ...props }: ComponentProps<'input'>) {
  return (
    <label className={cn('relative block min-w-55 flex-1 sm:max-w-95', className)}>
      <Search className="pointer-events-none absolute top-2.5 left-3 size-4 text-dim" />
      <Input className="pl-9" aria-label={props.placeholder} {...props} />
    </label>
  )
}

export function Field({ label, hint, error, children }: { label: string; hint?: ReactNode; error?: string; children: ReactNode }) {
  const id = useId()
  return (
    <FieldContext value={{ id, invalid: !!error }}>
      <div className="flex flex-col gap-1.5">
        <label htmlFor={id} className="text-[12.5px] font-medium text-muted">
          {label}
        </label>
        {children}
        {error ? <span className="text-xs text-danger">{error}</span> : hint && <span className="text-xs text-dim">{hint}</span>}
      </div>
    </FieldContext>
  )
}
