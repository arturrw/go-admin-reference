import * as RadixDialog from '@radix-ui/react-dialog'
import { TriangleAlert } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { Button } from './button'

/** Centered modal for short decisions; stacks above sheets. */
export function Dialog({
  open,
  onOpenChange,
  title,
  description,
  footer,
  className,
  children,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  footer?: ReactNode
  className?: string
  children?: ReactNode
}) {
  return (
    <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
      <RadixDialog.Portal>
        <RadixDialog.Overlay className="fixed inset-0 z-70 animate-fade-in bg-[rgb(4_5_6/0.55)] backdrop-blur-[2px]" />
        <RadixDialog.Content
          {...(description ? {} : { 'aria-describedby': undefined })}
          className={cn(
            'fixed top-1/2 left-1/2 z-71 flex w-[min(440px,calc(100vw-24px))] -translate-1/2 animate-fade-in flex-col gap-4 rounded-2xl border border-line-2 bg-panel p-5 shadow-float outline-none',
            className,
          )}
        >
          <div>
            <RadixDialog.Title className="text-[15px] font-semibold tracking-[-0.01em]">{title}</RadixDialog.Title>
            {description && <RadixDialog.Description className="mt-1 text-[13px] text-muted">{description}</RadixDialog.Description>}
          </div>
          {children}
          {footer && <div className="flex flex-wrap justify-end gap-2">{footer}</div>}
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  )
}

/** "Are you sure?" for destructive actions. */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = 'Delete',
  pending,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  confirmLabel?: string
  pending?: boolean
  onConfirm: () => void
}) {
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={
        <span className="flex items-center gap-2.5">
          <span className="grid size-8 place-items-center rounded-[10px] bg-danger/12 text-danger">
            <TriangleAlert className="size-4" />
          </span>
          {title}
        </span>
      }
      description={description}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)} autoFocus>
            Cancel
          </Button>
          <Button variant="danger" disabled={pending} onClick={onConfirm}>
            {pending ? 'Working…' : confirmLabel}
          </Button>
        </>
      }
    />
  )
}
