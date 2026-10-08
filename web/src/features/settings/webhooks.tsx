import { Check, Copy, Eye, EyeOff, RefreshCw, Send } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/ui/dialog'
import { Field, Input } from '@/components/ui/input'
import { Pill } from '@/components/ui/pill'
import { ApiError, type WebhookDelivery, type WorkspaceSettings } from '@/lib/api'
import { timeAgo } from '@/lib/format'
import { useDeliveries, usePatchSettings, useRotateSecret, useSettings, useTestWebhook } from '@/lib/queries'
import { ToggleRow } from './rows'

/** Where change events are sent, how they are signed, and how the last ones went. */
export function Webhooks({ enabled }: { enabled: boolean }) {
  const { data: s } = useSettings(enabled)
  if (!s) return <ToggleRow title="Signed webhooks" description="Only owners and admins can configure webhooks." checked={false} onChange={() => {}} disabled />
  return <WebhookForm key={s.webhookUrl} s={s} />
}

function WebhookForm({ s }: { s: WorkspaceSettings }) {
  const patch = usePatchSettings()
  const [url, setUrl] = useState(s.webhookUrl)
  const errors = patch.error instanceof ApiError ? patch.error.fields : undefined
  const submit = (e: FormEvent) => {
    e.preventDefault()
    patch.mutate({ webhookUrl: url })
  }

  return (
    <>
      <form onSubmit={submit} className="flex flex-col gap-2">
        <Field label="Webhook URL" error={errors?.webhookUrl} hint="Receives a JSON event for every change in the activity log: orders, refunds, products, notes, team, settings. Empty = off.">
          <div className="flex gap-2">
            <Input className="num min-w-0 flex-1" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://example.com/hooks/goadmin" />
            <Button type="submit" variant="primary" disabled={url.trim() === s.webhookUrl || patch.isPending}>
              Save
            </Button>
          </div>
        </Field>
      </form>
      <ToggleRow
        title="Signed webhooks"
        description="HMAC-SHA256 of the timestamp and body in X-GoAdmin-Signature, so receivers can verify the sender."
        checked={s.webhooksSigned}
        onChange={(v) => patch.mutate({ webhooksSigned: v })}
      />
      {s.webhooksSigned && <Secret secret={s.webhookSecret} />}
      {s.webhookUrl && <Deliveries />}
    </>
  )
}

function Secret({ secret }: { secret: string }) {
  const [shown, setShown] = useState(false)
  const [asking, setAsking] = useState(false)
  const rotate = useRotateSecret()
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(secret)
      toast('Secret copied')
    } catch {
      toast.error('Could not copy. Reveal it and copy it by hand.')
    }
  }
  return (
    <div className="flex flex-col gap-1.5 rounded-xl border border-line bg-panel px-3.5 py-3">
      <small className="text-xs text-dim">Signing secret</small>
      <div className="flex items-center gap-2">
        <code data-testid="webhook-secret" className="num min-w-0 flex-1 truncate text-[12.5px] text-muted">
          {shown ? secret : 'whsec_' + '•'.repeat(24)}
        </code>
        <Button size="icon-sm" variant="ghost" aria-label={shown ? 'Hide secret' : 'Reveal secret'} onClick={() => setShown(!shown)}>
          {shown ? <EyeOff /> : <Eye />}
        </Button>
        <Button size="icon-sm" variant="ghost" aria-label="Copy secret" onClick={copy}>
          <Copy />
        </Button>
        <Button size="sm" onClick={() => setAsking(true)}>
          <RefreshCw />
          Rotate
        </Button>
      </div>
      <ConfirmDialog
        open={asking}
        onOpenChange={setAsking}
        title="Rotate the signing secret?"
        description="The old secret stops matching at once. Update your receiver with the new one, or its checks will fail."
        confirmLabel="Rotate"
        pending={rotate.isPending}
        onConfirm={() => rotate.mutate(undefined, { onSuccess: () => setAsking(false) })}
      />
    </div>
  )
}

function Deliveries() {
  const { data } = useDeliveries(true)
  const test = useTestWebhook()
  const last = test.data
  return (
    <div className="flex flex-col gap-2 rounded-xl border border-line bg-panel px-3.5 py-3">
      <div className="flex items-center gap-2">
        <small className="text-xs text-dim">Recent deliveries</small>
        <Button size="sm" className="ml-auto" disabled={test.isPending} onClick={() => test.mutate()}>
          <Send />
          {test.isPending ? 'Sending…' : 'Send test event'}
        </Button>
      </div>
      {last && (
        <p role="status" data-testid="webhook-test-result" className={last.ok ? 'text-[12.5px] text-accent' : 'text-[12.5px] text-danger'}>
          {last.ok ? <Check className="mr-1 inline size-3.5" /> : null}
          {last.ok ? `Receiver answered ${last.status} in ${Math.round(last.durationMs)} ms` : `Failed after ${last.attempts} attempts: ${last.error}`}
        </p>
      )}
      {data?.items.length === 0 && <p className="text-[12.5px] text-dim">Nothing sent yet. Make a change, or send a test event.</p>}
      {data?.items.map((d) => <DeliveryRow key={d.id} d={d} />)}
    </div>
  )
}

function DeliveryRow({ d }: { d: WebhookDelivery }) {
  return (
    <div data-testid="webhook-delivery" className="flex items-center gap-2 text-[12.5px]">
      <Pill tone={d.ok ? 'lime' : 'red'}>{d.ok ? d.status : (d.status || 'error')}</Pill>
      <code className="num min-w-0 flex-1 truncate">{d.type}</code>
      {d.attempts > 1 && <small className="text-dim">{d.attempts} attempts</small>}
      <small className="num text-dim">{timeAgo(d.at)}</small>
    </div>
  )
}
