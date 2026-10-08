import { Copy, Mail } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button, buttonVariants } from '@/components/ui/button'
import { Dialog } from '@/components/ui/dialog'
import { Field, Input, Textarea } from '@/components/ui/input'
import type { Customer } from '@/lib/api'
import { buildMailto } from '@/lib/mailto'
import { cn } from '@/lib/utils'

/**
 * Writes to a segment through the person's own mail app: the server has no
 * mail transport, so the addresses go into Bcc of a mailto: link. Only
 * customers who accept marketing are included.
 */
export function EmailSegment({ customers, label, onClose }: { customers: Customer[]; label: string; onClose: () => void }) {
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const recipients = customers.filter((c) => c.acceptsMarketing)
  const skipped = customers.length - recipients.length
  const emails = recipients.map((c) => c.email)
  const mail = buildMailto(emails, subject, body)

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(emails.join(', '))
      toast.success(`Copied ${emails.length} ${emails.length === 1 ? 'address' : 'addresses'}`)
    } catch {
      toast.error('Could not copy to the clipboard')
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(v) => !v && onClose()}
      title={`Email ${label}`}
      description="Opens a draft in your mail app with everyone in Bcc, so nobody sees the other addresses."
      className="w-[min(520px,calc(100vw-24px))]"
      footer={
        <>
          <Button onClick={copy} disabled={!emails.length}>
            <Copy />
            Copy addresses
          </Button>
          <a
            href={emails.length ? mail.href : undefined}
            aria-disabled={!emails.length}
            data-testid="mailto"
            onClick={emails.length ? onClose : (e) => e.preventDefault()}
            className={cn(buttonVariants({ variant: 'primary' }), !emails.length && 'pointer-events-none opacity-45')}
          >
            <Mail />
            Open in mail app
          </a>
        </>
      }
    >
      <p className="text-[13px]" data-testid="recipients">
        <b className="num font-medium">{recipients.length}</b> {recipients.length === 1 ? 'customer' : 'customers'} will get it
        {skipped > 0 && <span className="text-dim"> · {skipped} left out (no marketing consent)</span>}
      </p>
      {recipients.length === 0 && <p className="text-[12.5px] text-warn">Nobody in this view accepts marketing email.</p>}
      <Field label="Subject">
        <Input value={subject} onChange={(e) => setSubject(e.target.value)} placeholder="e.g. New arrivals for you" autoFocus />
      </Field>
      <Field label="Message">
        <Textarea rows={5} value={body} onChange={(e) => setBody(e.target.value)} />
      </Field>
      {mail.included < emails.length && (
        <p className="text-[12.5px] text-warn" data-testid="mailto-limit">
          A mail link only fits {mail.included} of the {emails.length} addresses. Open it for those, then copy the addresses and add the rest by hand.
        </p>
      )}
    </Dialog>
  )
}
