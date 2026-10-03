import { Check, Copy, KeyRound, Lock, OctagonAlert, Palette, Plus, SlidersHorizontal } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Field, Input, Select } from '@/components/ui/input'
import { PageHeader, Switch } from '@/components/ui/misc'
import { Segmented } from '@/components/ui/segmented'
import { ACCENTS, getAccent, setAccent } from '@/lib/theme'
import { cn } from '@/lib/utils'

// Settings are local-only in this reference build; wire them to an API
// endpoint (e.g. PUT /api/v1/settings) when you need persistence.

const SECTIONS = [
  ['general', 'General', SlidersHorizontal],
  ['appearance', 'Appearance', Palette],
  ['security', 'Security', Lock],
  ['api', 'API keys', KeyRound],
  ['danger', 'Danger zone', OctagonAlert],
] as const

export function SettingsPage() {
  const [accent, setAccentState] = useState(getAccent)
  const [logLevel, setLogLevel] = useState('info')
  const [density, setDensity] = useState('comfortable')
  const [toggles, setToggles] = useState({ maint: false, mfa: true, audit: true, alerts: true, webhooks: false })
  const flip = (k: keyof typeof toggles, label: string) => (v: boolean) => {
    setToggles((t) => ({ ...t, [k]: v }))
    toast(`${label}: ${v ? 'on' : 'off'}`)
  }

  return (
    <>
      <PageHeader title="Settings" description="Workspace, service and security configuration." />
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
          <Section id="general" title="General" description="Basic service configuration loaded from env on boot.">
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
            <Field label="Log level">
              <Segmented className="self-start" value={logLevel} onChange={setLogLevel} options={['debug', 'info', 'warn', 'error'].map((l) => ({ value: l, label: l }))} />
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
                  className={cn('grid size-8.5 place-items-center rounded-[10px] border-2 text-accent-ink', accent === c ? 'border-fg' : 'border-transparent')}
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

          <Section id="security" title="Security" description="Authentication and audit policies for team members.">
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

          <Section id="api" title="API keys" description="Keys for server-to-server access to /api/v1.">
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

          <Section id="danger" title="Danger zone" description="Irreversible actions. Be careful.">
            <DangerRow title="Flush cache" description="Clears cached keys under goadmin:*" action="Flush" onClick={() => toast('Cache flushed')} />
            <DangerRow title="Delete workspace" description="Removes all data. This cannot be undone." action="Delete" onClick={() => toast.error('Disabled in the reference build')} />
          </Section>
        </div>
      </div>
    </>
  )
}

function Section({ id, title, description, children }: { id: string; title: string; description: string; children: ReactNode }) {
  return (
    <section id={`s-${id}`} className="grid scroll-mt-20 gap-3 border-b border-line py-6 first:pt-0 md:grid-cols-[260px_minmax(0,1fr)] md:gap-6">
      <div>
        <h3 className="text-sm font-medium">{title}</h3>
        <p className="mt-1 text-[12.5px] text-dim">{description}</p>
      </div>
      <div className="flex max-w-140 flex-col gap-3.5">{children}</div>
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
