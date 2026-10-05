import * as Dialog from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { Button } from './button'

const WIDTHS = {
  md: 'w-[min(460px,calc(100vw-20px))]',
  lg: 'w-[min(720px,calc(100vw-20px))]',
}

/** Right-hand slide-over panel used for create/edit/detail views. */
export function Sheet({
  open,
  onOpenChange,
  title,
  description,
  footer,
  size = 'md',
  children,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  footer?: ReactNode
  size?: keyof typeof WIDTHS
  children: ReactNode
}) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-60 animate-fade-in bg-[rgb(4_5_6/0.6)] backdrop-blur-[3px]" />
        <Dialog.Content
          // Radix warns unless a Description exists or aria-describedby is explicitly unset.
          {...(description ? {} : { 'aria-describedby': undefined })}
          className={cn(
            'fixed top-2.5 right-2.5 bottom-2.5 z-61 flex animate-sheet-in flex-col rounded-[18px] border border-line-2 bg-panel shadow-float outline-none',
            WIDTHS[size],
          )}
        >
          <div className="flex items-center gap-2.5 border-b border-line px-4.5 py-4">
            <div className="min-w-0">
              <Dialog.Title className="text-base font-semibold tracking-[-0.01em]">{title}</Dialog.Title>
              {description && <Dialog.Description className="mt-0.5 text-[12.5px] text-dim">{description}</Dialog.Description>}
            </div>
            <Dialog.Close asChild>
              <Button variant="ghost" size="icon" className="ml-auto" aria-label="Close">
                <X />
              </Button>
            </Dialog.Close>
          </div>
          {/* Children never shrink: blocks with overflow set would otherwise be squashed and clipped. */}
          <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4.5 *:shrink-0">{children}</div>
          {footer && <div className="flex flex-wrap justify-end gap-2 border-t border-line px-4.5 py-3.5">{footer}</div>}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
