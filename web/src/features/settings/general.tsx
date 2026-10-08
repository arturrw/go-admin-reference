import { type FormEvent, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/misc'
import { ApiError, type WorkspaceSettings } from '@/lib/api'
import { useMeta, usePatchSettings, useSettings } from '@/lib/queries'

/** Service name, public URL and the (read-only) listen address. */
export function GeneralSettings({ enabled }: { enabled: boolean }) {
  const { data, isPending } = useSettings(enabled)
  const { data: meta } = useMeta()
  if (enabled && isPending) return <Skeleton className="h-52" />
  if (!data) {
    // Read-only members see the name but not the rest.
    return (
      <Field label="Service name">
        <Input value={meta?.serviceName ?? ''} disabled readOnly />
      </Field>
    )
  }
  return <GeneralForm key={`${data.serviceName}|${data.publicBaseUrl}`} settings={data} />
}

function GeneralForm({ settings }: { settings: WorkspaceSettings }) {
  const [name, setName] = useState(settings.serviceName)
  const [url, setUrl] = useState(settings.publicBaseUrl)
  const patch = usePatchSettings()
  const errors = patch.error instanceof ApiError ? patch.error.fields : undefined
  const dirty = name.trim() !== settings.serviceName || url.trim().replace(/\/+$/, '') !== settings.publicBaseUrl

  const submit = (e: FormEvent) => {
    e.preventDefault()
    patch.mutate({ serviceName: name, publicBaseUrl: url })
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-3.5">
      <div className="grid gap-3.5 sm:grid-cols-2">
        <Field label="Service name" error={errors?.serviceName} hint="Shown in the sidebar.">
          <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={60} />
        </Field>
        <Field label="Listen address" hint={`Set by ADDR (${settings.env}); changing it needs a restart.`}>
          <Input className="num" value={settings.listenAddr} disabled readOnly />
        </Field>
      </div>
      <Field label="Public base URL" error={errors?.publicBaseUrl} hint="Used in links the admin generates, such as invoices and API examples.">
        <Input className="num" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://admin.example.com" />
      </Field>
      <div>
        <Button variant="primary" type="submit" disabled={!dirty || patch.isPending}>
          {patch.isPending ? 'Saving…' : 'Save changes'}
        </Button>
      </div>
    </form>
  )
}
