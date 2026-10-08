import { Field, Select } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/misc'
import { usePatchSettings, useSettings } from '@/lib/queries'
import { ToggleRow } from './rows'

// Lifetimes the API accepts (domain.SessionLifetimes).
const LIFETIMES = [3600, 8 * 3600, 12 * 3600, 24 * 3600, 7 * 24 * 3600]

const label = (s: number) => (s >= 86400 && s % 86400 === 0 ? `${s / 86400} ${s === 86400 ? 'day' : 'days'}` : s % 3600 === 0 ? `${s / 3600} ${s === 3600 ? 'hour' : 'hours'}` : `${Math.round(s / 60)} minutes`)

export function SecuritySettings({ enabled }: { enabled: boolean }) {
  const { data: s, isPending } = useSettings(enabled)
  const patch = usePatchSettings()
  if (enabled && isPending) return <Skeleton className="h-60" />

  // SESSION_TTL may be a value that isn't one of the choices; show it too.
  const ttl = s?.sessionTtlSeconds ?? 0
  const options = ttl && !LIFETIMES.includes(ttl) ? [ttl, ...LIFETIMES].sort((a, b) => a - b) : LIFETIMES

  return (
    <>
      <ToggleRow title="Require 2FA" description="All members must enroll a TOTP or passkey. Needs two-factor sign-in, which this build doesn't have." checked={false} onChange={() => {}} planned />
      <ToggleRow
        title="Audit log"
        description="Record changes and sign-ins in the activity log. When off, nothing new is recorded, except switching this."
        checked={s?.auditLog ?? true}
        onChange={(v) => patch.mutate({ auditLog: v })}
      />
      <ToggleRow
        title="Login alerts"
        description={
          s && !s.auditLog
            ? 'Needs the audit log: new devices are found in its sign-ins.'
            : 'Alert a member in their notifications when their account signs in from a new device.'
        }
        checked={(s?.loginAlerts ?? true) && (s?.auditLog ?? true)}
        disabled={!!s && !s.auditLog}
        onChange={(v) => patch.mutate({ loginAlerts: v })}
      />
      <Field label="Session lifetime" hint="How long a session lasts without use. It applies from each member's next request.">
        <Select className="max-w-55" aria-label="Session lifetime" value={ttl || ''} disabled={!s || patch.isPending} onChange={(e) => patch.mutate({ sessionTtlSeconds: Number(e.target.value) })}>
          {options.map((v) => (
            <option key={v} value={v}>
              {label(v)}
            </option>
          ))}
        </Select>
      </Field>
    </>
  )
}
