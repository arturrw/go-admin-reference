import { Check, Copy, KeyRound, Plus, TriangleAlert } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { ConfirmDialog, Dialog } from '@/components/ui/dialog'
import { Field, Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/misc'
import { Pill } from '@/components/ui/pill'
import { Segmented } from '@/components/ui/segmented'
import { ApiError, type ApiKey, type ApiKeyScope } from '@/lib/api'
import { shortDate, timeAgo } from '@/lib/format'
import { useApiKeys, useCreateApiKey, useRevokeApiKey } from '@/lib/queries'

const SCOPES: { value: ApiKeyScope; label: string; hint: string }[] = [
  { value: 'read', label: 'Read', hint: 'Like a viewer: dashboard, products, orders and customers, read-only.' },
  { value: 'write', label: 'Read & write', hint: 'Like an editor: also edits products and orders. Never settings or team.' },
]

/** Keys other servers use as `Authorization: Bearer <key>`. Only members who can edit settings see them. */
export function ApiKeys({ enabled }: { enabled: boolean }) {
  const { data, isPending } = useApiKeys(enabled)
  const [creating, setCreating] = useState(false)
  const [secret, setSecret] = useState<{ name: string; value: string } | null>(null)
  const [revoking, setRevoking] = useState<ApiKey | null>(null)
  const revoke = useRevokeApiKey()

  return (
    <>
      {enabled && isPending && <Skeleton className="h-20" />}
      {data?.items.length === 0 && <p className="rounded-xl border border-dashed border-line-2 px-3.5 py-5 text-center text-[13px] text-dim">No API keys yet.</p>}
      {data?.items.map((k) => (
        <div key={k.id} data-testid="api-key" className="flex items-center gap-3 rounded-xl border border-line bg-panel px-3.5 py-3">
          <KeyRound className="size-4 shrink-0 text-accent" />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <b className="font-medium">{k.name}</b>
              <Pill tone={k.scope === 'write' ? 'amber' : 'gray'} dot={false}>
                {k.scope === 'write' ? 'read & write' : 'read'}
              </Pill>
            </div>
            <code className="num block text-[12.5px] text-muted">{k.masked}</code>
            <small className="text-xs text-dim">
              Created {shortDate(k.createdAt)} by {k.createdBy} · {k.lastUsedAt ? `used ${timeAgo(k.lastUsedAt)}` : 'never used'}
            </small>
          </div>
          <Button size="sm" variant="danger" aria-label={`Revoke ${k.name}`} onClick={() => setRevoking(k)}>
            Revoke
          </Button>
        </div>
      ))}
      <div>
        <Button onClick={() => setCreating(true)} disabled={!enabled}>
          <Plus />
          Create key
        </Button>
      </div>

      <CreateKeyDialog
        open={creating}
        onOpenChange={setCreating}
        onCreated={(name, value) => {
          setCreating(false)
          setSecret({ name, value })
        }}
      />
      {secret && <SecretDialog name={secret.name} value={secret.value} onClose={() => setSecret(null)} />}
      <ConfirmDialog
        open={!!revoking}
        onOpenChange={(v) => !v && setRevoking(null)}
        title={`Revoke “${revoking?.name ?? ''}”?`}
        description="Anything using this key stops working immediately. This can't be undone; create a new key to replace it."
        confirmLabel="Revoke key"
        pending={revoke.isPending}
        onConfirm={() => revoking && revoke.mutate(revoking.id, { onSuccess: () => setRevoking(null) })}
      />
    </>
  )
}

function CreateKeyDialog({ open, onOpenChange, onCreated }: { open: boolean; onOpenChange: (v: boolean) => void; onCreated: (name: string, secret: string) => void }) {
  // Remounted on each open, so the form starts empty.
  return (
    <Dialog open={open} onOpenChange={onOpenChange} title="Create API key" description="Give it a name that says what uses it. You'll see the key once.">
      {open && <CreateKeyForm onCancel={() => onOpenChange(false)} onCreated={onCreated} />}
    </Dialog>
  )
}

function CreateKeyForm({ onCancel, onCreated }: { onCancel: () => void; onCreated: (name: string, secret: string) => void }) {
  const [name, setName] = useState('')
  const [scope, setScope] = useState<ApiKeyScope>('read')
  const create = useCreateApiKey()
  const errors = create.error instanceof ApiError ? create.error.fields : undefined
  const submit = (e: FormEvent) => {
    e.preventDefault()
    create.mutate({ name, scope }, { onSuccess: (r) => onCreated(r.key.name, r.secret) })
  }
  return (
    <form onSubmit={submit} className="flex flex-col gap-3.5">
      <Field label="Name" error={errors?.name}>
        <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Reporting dashboard" autoFocus maxLength={60} />
      </Field>
      <Field label="Access" error={errors?.scope} hint={SCOPES.find((s) => s.value === scope)?.hint}>
        <Segmented className="self-start" value={scope} onChange={setScope} options={SCOPES.map(({ value, label }) => ({ value, label }))} />
      </Field>
      <div className="flex justify-end gap-2 pt-1">
        <Button onClick={onCancel}>Cancel</Button>
        <Button variant="primary" type="submit" disabled={!name.trim() || create.isPending}>
          <Plus />
          {create.isPending ? 'Creating…' : 'Create key'}
        </Button>
      </div>
    </form>
  )
}

/** The only time the full key is visible. */
function SecretDialog({ name, value, onClose }: { name: string; value: string; onClose: () => void }) {
  const [copied, setCopied] = useState(false)
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
    } catch {
      toast.error('Could not copy. Select the key and copy it by hand.')
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(v) => !v && onClose()}
      title={`Key “${name}” created`}
      className="w-[min(520px,calc(100vw-24px))]"
      footer={
        <Button variant="primary" onClick={onClose}>
          Done
        </Button>
      }
    >
      <div className="flex items-start gap-2 rounded-xl border border-warn/30 bg-warn/8 px-3 py-2.5 text-[12.5px] text-warn">
        <TriangleAlert className="mt-0.5 size-4 shrink-0" />
        Copy it now. For security only a hash is stored, so this key can't be shown again.
      </div>
      <div className="flex items-center gap-2">
        <code data-testid="api-key-secret" className="num min-w-0 flex-1 rounded-[10px] border border-line-2 bg-panel-2 px-3 py-2.5 text-[12.5px] break-all select-all">
          {value}
        </code>
        <Button size="icon" aria-label="Copy key" onClick={copy}>
          {copied ? <Check className="text-accent" /> : <Copy />}
        </Button>
      </div>
      <pre className="num overflow-x-auto rounded-[10px] border border-line bg-panel-2/60 px-3 py-2.5 text-[11.5px] text-muted">
        {`curl -H "Authorization: Bearer ${value.slice(0, 12)}…" \\\n  ${location.origin}/api/v1/orders?limit=5`}
      </pre>
    </Dialog>
  )
}
