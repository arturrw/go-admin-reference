import { Check, Copy } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Dialog } from '@/components/ui/dialog'
import type { Invite } from '@/lib/api'

/**
 * The invitation link, shown right after inviting someone. The server can't
 * send email, so the person who invited delivers it; it works once, for a week.
 */
export function InviteLinkDialog({ name, invite, onClose }: { name: string; invite: Invite; onClose: () => void }) {
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(invite.url)
      setCopied(true)
    } catch {
      toast.error('Could not copy. Select the link and copy it by hand.')
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(v) => !v && onClose()}
      title={`Invitation for ${name}`}
      description={`Send them this link. It works once and expires on ${new Date(invite.expiresAt).toLocaleDateString('en-US', { month: 'long', day: 'numeric' })}.`}
      className="w-[min(520px,calc(100vw-24px))]"
      footer={
        <Button variant="primary" onClick={onClose}>
          Done
        </Button>
      }
    >
      <div className="flex items-center gap-2">
        <code data-testid="invite-url" className="num min-w-0 flex-1 rounded-[10px] border border-line-2 bg-panel-2 px-3 py-2.5 text-[12.5px] break-all select-all">
          {invite.url}
        </code>
        <Button size="icon" aria-label="Copy invitation link" onClick={copy}>
          {copied ? <Check className="text-accent" /> : <Copy />}
        </Button>
      </div>
      <p className="text-[12.5px] text-dim">This server doesn't send email. Paste the link into a message. A new link replaces this one.</p>
    </Dialog>
  )
}
