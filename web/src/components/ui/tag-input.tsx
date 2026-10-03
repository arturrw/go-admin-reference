import { X } from 'lucide-react'
import { type KeyboardEvent, useState } from 'react'
import { cn } from '@/lib/utils'

/** Free-form tags: Enter or comma adds, Backspace on empty input removes the last one. */
export function TagInput({ value, onChange, disabled, placeholder = 'Add tag…' }: { value: string[]; onChange: (v: string[]) => void; disabled?: boolean; placeholder?: string }) {
  const [draft, setDraft] = useState('')

  const add = () => {
    const t = draft.trim().toLowerCase().replace(/,$/, '')
    if (t && !value.includes(t) && value.length < 10) onChange([...value, t])
    setDraft('')
  }

  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      add()
    } else if (e.key === 'Backspace' && !draft && value.length) {
      onChange(value.slice(0, -1))
    }
  }

  return (
    <div
      className={cn(
        'flex min-h-9 flex-wrap items-center gap-1.5 rounded-[10px] border border-line-2 bg-panel px-2 py-1.5 transition focus-within:border-accent/55 focus-within:ring-3 focus-within:ring-accent/14',
        disabled && 'opacity-60',
      )}
    >
      {value.map((t) => (
        <span key={t} className="inline-flex items-center gap-1 rounded-md bg-panel-3 py-0.5 pr-1 pl-2 text-xs text-fg">
          {t}
          {!disabled && (
            <button type="button" aria-label={`Remove ${t}`} onClick={() => onChange(value.filter((x) => x !== t))} className="rounded text-dim hover:text-fg">
              <X className="size-3" />
            </button>
          )}
        </span>
      ))}
      {!disabled && (
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={onKey}
          onBlur={add}
          aria-label="Add tag"
          placeholder={value.length ? '' : placeholder}
          className="min-w-20 flex-1 bg-transparent px-1 text-sm outline-none placeholder:text-dim"
        />
      )}
    </div>
  )
}
