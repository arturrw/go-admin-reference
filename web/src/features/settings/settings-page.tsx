import { Check, KeyRound, Lock, OctagonAlert, Palette, ShieldAlert, SlidersHorizontal } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { toast } from 'sonner'
import { Field } from '@/components/ui/input'
import { PageHeader } from '@/components/ui/misc'
import { Segmented } from '@/components/ui/segmented'
import type { LogLevel } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { useLogLevel, usePatchSettings, useSetLogLevel, useSettings } from '@/lib/queries'
import { ACCENTS, getAccent, getDensity, setAccent, setDensity } from '@/lib/theme'
import { cn } from '@/lib/utils'
import { ApiKeys } from './api-keys'
import { GeneralSettings } from './general'
import { DangerZone } from './danger'
import { ToggleRow } from './rows'
import { SecuritySettings } from './security'

// Accent and density are personal and stay in this browser. Everything else is
// saved on the server (/api/v1/settings, the log level, API keys, danger zone).

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
  const [density, setDensityState] = useState(getDensity)
  const settings = useSettings(canEdit)
  const patch = usePatchSettings()
  const [toggles, setToggles] = useState({ webhooks: false })
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
            <GeneralSettings enabled={canEdit} />
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
            <ToggleRow
              title="Maintenance mode"
              description="Locks out everyone except owners and admins: they see a holding page and API keys get 503"
              checked={settings.data?.maintenance ?? false}
              onChange={(v) => patch.mutate({ maintenance: v })}
            />
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
                onChange={(d) => {
                  setDensity(d)
                  setDensityState(d)
                }}
                options={[
                  { value: 'comfortable', label: 'Comfortable' },
                  { value: 'compact', label: 'Compact' },
                ]}
              />
            </Field>
          </Section>

          <Section id="security" title="Security" description="Authentication and audit policies for team members." locked={!canEdit}>
            <SecuritySettings enabled={canEdit} />
          </Section>

          <Section id="api" title="API keys" description="Keys for server-to-server access to /api/v1." locked={!canEdit}>
            <ApiKeys enabled={canEdit} />
            <ToggleRow title="Signed webhooks" description="HMAC-SHA256 signature in X-GoAdmin-Signature" checked={toggles.webhooks} onChange={flip('webhooks', 'Signed webhooks')} />
          </Section>

          <Section id="danger" title="Danger zone" description={canDanger ? 'Irreversible actions. Be careful.' : 'Owner only.'} locked={!canDanger}>
            <DangerZone enabled={canDanger} />
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
