import { Check, Copy, KeyRound, Lock, OctagonAlert, ShieldAlert, Palette, Plus, SlidersHorizontal } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Field, Input, Select } from '@/components/ui/input'
import { PageHeader, Switch } from '@/components/ui/misc'
import { Segmented } from '@/components/ui/segmented'
import type { LogLevel } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { useLogLevel, useSetLogLevel } from '@/lib/queries'
import { ACCENTS, getAccent, setAccent } from '@/lib/theme'
import { cn } from '@/lib/utils'

// Settings are local-only in this reference build except the log level,
// which is applied to the running server (PUT /api/v1/settings/log-level).
// Wire the rest to an API endpoint when you need persistence.

const LOG_LEVELS: { value: LogLevel; hint: string }[] = [
  { value: 'debug', hint: 'Everything, including static files and the UI’s polling requests' },
  { value: 'info', hint: 'Every API request, sign-ins, imports and startup messages' },
  { value: 'warn', hint: 'Only client errors (4xx), settings changes and warnings' },
  { value: 'error', hint: 'Only server errors (5xx) and failures' },
]

const SECTIONS = [
  ['general', 'General', SlidersHorizontal],
  ['appearance', 'Appearance', Palette],
  ['security', 'Security', Lock],
  ['api', 'API keys', KeyRound],
  ['danger', 'Danger zone', OctagonAlert],
] as const

export function SettingsPage() {
  const canEdit = useCan('settings:write')
  const canDanger = useCan('workspace:manage')
  const [accent, setAccentState] = useState(getAccent)
  const logLevel = useLogLevel()
  const setLogLevel = useSetLogLevel()
  const level = setLogLevel.isPending ? setLogLevel.variables : logLevel.data?.level
  const [density, setDensity] = useState('comfortable')
  const [toggles, setToggles] = useState({ maint: false, mfa: true, audit: true, alerts: true, webhooks: false })
  const flip = (k: keyof typeof toggles, label: string) => (v: boolean) => {
    setToggles((t) => ({ ...t, [k]: v }))
    toast(`${label}: ${v ? 'on' : 'off'}`)
  }

  return (
    <>
      <PageHeader title="Settings" description="Workspace, service and security configuration." />
      {!canEdit && (
        <div className="mb-5 flex items-center gap-2 rounded-xl border border-line bg-panel-2 px-3.5 py-2.5 text-[12.5px] text-muted">
          <ShieldAlert className="size-4 text-warn" />
          Read-only — only owners and admins can change workspace settings. Appearance is personal and always editable.
        </div>
      )}
      <div className="grid items-start gap-7 lg:grid-cols-[200px_minmax(0,1fr)]">
        <nav className="sticky top-20 hidden flex-col gap-0.5 lg:flex">
          {SECTIONS.map(([id, label, Icon]) => (
            <a key={id} href={`#s-${id}`} className="flex items-center gap-2 rounded-lg px-2.5 py-1.5 text-[13px] text-muted hover:bg-panel-2 hover:text-fg">
              <Icon className="size-4" />
              {label}
            </a>
          ))}
        </nav>

        <div>
          <Section id="general" title="General" description="Basic service configuration loaded from env on boot." locked={!canEdit}>
            <div className="grid gap-3.5 sm:grid-cols-2">
              <Field label="Service name">
                <Input defaultValue="goadmin-api" />
              </Field>
              <Field label="Listen address">
                <Input className="num" defaultValue=":8080" />
              </Field>
            </div>
            <Field label="Public base URL">
              <Input className="num" defaultValue="https://admin.acme.io" />
            </Field>
            <Field label="Log level" hint={
                <>
                  {LOG_LEVELS.find((l) => l.value === level)?.hint ?? 'Loading…'}. Applied to the running server immediately; after a restart LOG_LEVEL decides
                  again.
                </>
              }>
              <Segmented
                className="self-start"
                value={level ?? ('' as LogLevel)}
                onChange={(l) => l !== level && setLogLevel.mutate(l)}
                options={LOG_LEVELS.map((l) => ({ value: l.value, label: l.value }))}
              />
            </Field>
            <ToggleRow title="Maintenance mode" description="Storefront returns 503 with a friendly page" checked={toggles.maint} onChange={flip('maint', 'Maintenance mode')} />
            <div>
              <Button variant="primary" onClick={() => toast.success('Settings saved')}>
                Save changes
              </Button>
            </div>
          </Section>

          <Section id="appearance" title="Appearance" description="Accent color for this workspace. Applied instantly and remembered in this browser.">
            <div className="flex flex-wrap gap-2.5">
              {ACCENTS.map((c) => (
                <button
                  key={c}
                  aria-label={`Accent ${c}`}
                  onClick={() => {
                    setAccent(c)
                    setAccentState(c)
                  }}
                  className={cn('grid size-8.5 place-items-center rounded-[10px] text-accent-ink ring-offset-2 ring-offset-panel', accent === c && 'ring-2 ring-fg/80')}
                  style={{ background: c }}
                >
                  {accent === c && <Check className="size-4" />}
                </button>
              ))}
            </div>
            <Field label="Density">
              <Segmented
                className="self-start"
                value={density}
                onChange={setDensity}
                options={[
                  { value: 'comfortable', label: 'Comfortable' },
                  { value: 'compact', label: 'Compact' },
                ]}
              />
            </Field>
          </Section>

          <Section id="security" title="Security" description="Authentication and audit policies for team members." locked={!canEdit}>
            <ToggleRow title="Require 2FA" description="All members must enroll a TOTP or passkey" checked={toggles.mfa} onChange={flip('mfa', 'Require 2FA')} />
            <ToggleRow title="Audit log" description="Record every write action with actor & diff" checked={toggles.audit} onChange={flip('audit', 'Audit log')} />
            <ToggleRow title="Login alerts" description="Email members on sign-in from a new device" checked={toggles.alerts} onChange={flip('alerts', 'Login alerts')} />
            <Field label="Session lifetime">
              <Select className="max-w-55" defaultValue="24h">
                <option value="8h">8 hours</option>
                <option value="24h">24 hours</option>
                <option value="7d">7 days</option>
              </Select>
            </Field>
          </Section>

          <Section id="api" title="API keys" description="Keys for server-to-server access to /api/v1." locked={!canEdit}>
            {[
              ['Storefront (read)', 'ga_live_••••••••3f9a', 'Created Aug 12 · used 2m ago'],
              ['Warehouse sync', 'ga_live_••••••••a71c', 'Created Jun 2 · used 1h ago'],
              ['CI smoke tests', 'ga_test_••••••••0b2e', 'Created Sep 30 · never used'],
            ].map(([name, key, meta]) => (
              <div key={name} className="flex items-center gap-3 rounded-xl border border-line bg-panel px-3.5 py-3">
                <KeyRound className="size-4 text-accent" />
                <div className="min-w-0 flex-1">
                  <b className="font-medium">{name}</b>
                  <code className="num block text-[12.5px] text-muted">{key}</code>
                  <small className="text-xs text-dim">{meta}</small>
                </div>
                <Button size="icon-sm" aria-label="Copy key" onClick={() => toast('Key copied')}>
                  <Copy />
                </Button>
                <Button size="sm" variant="danger" onClick={() => toast('Key revoked')}>
                  Revoke
                </Button>
              </div>
            ))}
            <ToggleRow title="Signed webhooks" description="HMAC-SHA256 signature in X-GoAdmin-Signature" checked={toggles.webhooks} onChange={flip('webhooks', 'Signed webhooks')} />
            <div>
              <Button>
                <Plus />
                Create key
              </Button>
            </div>
          </Section>

          <Section id="danger" title="Danger zone" description={canDanger ? 'Irreversible actions. Be careful.' : 'Owner only.'} locked={!canDanger}>
            <DangerRow title="Flush cache" description="Clears cached keys under goadmin:*" action="Flush" onClick={() => toast('Cache flushed')} />
            <DangerRow title="Delete workspace" description="Removes all data. This cannot be undone." action="Delete" onClick={() => toast.error('Disabled in the reference build')} />
          </Section>
        </div>
      </div>
    </>
  )
}

function Section({ id, title, description, locked, children }: { id: string; title: string; description: string; locked?: boolean; children: ReactNode }) {
  return (
    <section id={`s-${id}`} className="grid scroll-mt-20 gap-3 border-b border-line py-6 first:pt-0 md:grid-cols-[260px_minmax(0,1fr)] md:gap-6">
      <div>
        <h3 className="text-sm font-medium">{title}</h3>
        <p className="mt-1 text-[12.5px] text-dim">{description}</p>
      </div>
      <fieldset disabled={locked} className={cn('flex max-w-140 flex-col gap-3.5', locked && 'pointer-events-none opacity-55')}>
        {children}
      </fieldset>
    </section>
  )
}

function ToggleRow({ title, description, checked, onChange }: { title: string; description: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-center justify-between gap-3.5 rounded-xl border border-line bg-panel px-3.5 py-3">
      <div>
        <b className="block font-medium">{title}</b>
        <small className="text-xs text-dim">{description}</small>
      </div>
      <Switch checked={checked} onChange={onChange} label={title} />
    </div>
  )
}

function DangerRow({ title, description, action, onClick }: { title: string; description: string; action: string; onClick: () => void }) {
  return (
    <div className="flex items-center justify-between gap-3.5 rounded-xl border border-danger/30 bg-danger/5 px-3.5 py-3">
      <div>
        <b className="block font-medium">{title}</b>
        <small className="text-xs text-dim">{description}</small>
      </div>
      <Button size="sm" variant="danger" onClick={onClick}>
        {action}
      </Button>
    </div>
  )
}
