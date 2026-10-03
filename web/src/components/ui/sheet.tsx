import * as Dialog from '@radix-ui/react-dialog'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { Button } from './button'

/** Right-hand slide-over panel used for create/edit/detail views. */
export function Sheet({
  open,
  onOpenChange,
  title,
  description,
  footer,
  children,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  footer?: ReactNode
  children: ReactNode
}) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-60 animate-fade-in bg-[rgb(4_5_6/0.6)] backdrop-blur-[3px]" />
        <Dialog.Content
          // Radix warns unless a Description exists or aria-describedby is explicitly unset.
          {...(description ? {} : { 'aria-describedby': undefined })}
          className="fixed top-2.5 right-2.5 bottom-2.5 z-61 flex w-[min(460px,calc(100vw-20px))] animate-sheet-in flex-col rounded-[18px] border border-line-2 bg-panel shadow-float outline-none"
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
          <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4.5">{children}</div>
          {footer && <div className="flex justify-end gap-2 border-t border-line px-4.5 py-3.5">{footer}</div>}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
