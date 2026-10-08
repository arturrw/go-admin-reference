import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/misc'
import { Pill } from '@/components/ui/pill'

export function ToggleRow({
  title,
  description,
  checked,
  onChange,
  disabled,
  planned,
}: {
  title: string
  description: ReactNode
  checked: boolean
  onChange: (v: boolean) => void
  disabled?: boolean
  /** Marks a setting that isn't implemented in this reference build. */
  planned?: boolean
}) {
  return (
    <div className="flex items-center justify-between gap-3.5 rounded-xl border border-line bg-panel px-3.5 py-3">
      <div className="min-w-0">
        <b className="flex items-center gap-2 font-medium">
          {title}
          {planned && (
            <Pill tone="gray" dot={false}>
              planned
            </Pill>
          )}
        </b>
        <small className="text-xs text-dim">{description}</small>
      </div>
      <Switch checked={checked} onChange={onChange} label={title} disabled={disabled || planned} />
    </div>
  )
}

export function DangerRow({
  title,
  description,
  action,
  onClick,
  disabled,
}: {
  title: string
  description: ReactNode
  action: string
  onClick: () => void
  disabled?: boolean
}) {
  return (
    <div className="flex items-center justify-between gap-3.5 rounded-xl border border-danger/30 bg-danger/5 px-3.5 py-3">
      <div className="min-w-0">
        <b className="block font-medium">{title}</b>
        <small className="text-xs text-dim">{description}</small>
      </div>
      <Button size="sm" variant="danger" onClick={onClick} disabled={disabled}>
        {action}
      </Button>
    </div>
  )
}
