import { BookOpen, Check, ExternalLink, Keyboard, Minus, ShieldCheck } from 'lucide-react'
import type { ReactNode } from 'react'
import { Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { Sheet } from '@/components/ui/sheet'
import { ROLE_INFO } from '@/features/team/member-detail-sheet'
import { useMe } from '@/lib/auth'
import { useMeta, useRoles } from '@/lib/queries'

const REPO = 'https://github.com/arturrw/go-admin-reference'
const DOCS = [
  ['Architecture and diagrams', `${REPO}/blob/main/docs/ARCHITECTURE.md`],
  ['API reference', `${REPO}/blob/main/docs/API.md`],
  ['Contributing', `${REPO}/blob/main/CONTRIBUTING.md`],
  ['Report a problem', `${REPO}/issues/new`],
] as const

const mod = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘' : 'Ctrl'

const SHORTCUTS: [string[], string][] = [
  [[mod, 'K'], 'Search and jump anywhere'],
  [['?'], 'Open this help'],
  [['Esc'], 'Close a panel, sheet or dialog'],
  [['Enter'], 'Open the focused card or row'],
]

/** What you can do here, how to get around, and where the documentation is. */
export function HelpSheet({ onClose }: { onClose: () => void }) {
  const me = useMe()
  const { data: roles } = useRoles()
  const { data: meta } = useMeta()
  let group = ''

  return (
    <Sheet open onOpenChange={(v) => !v && onClose()} title="Help" description="Your access, shortcuts and documentation">
      <Section icon={<ShieldCheck />} title="Your access">
        <div className="flex flex-wrap items-center gap-2 text-[13px]">
          <StatusPill status={me.user.role} />
          <span className="text-muted">{ROLE_INFO[me.user.role]}</span>
        </div>
        {me.user.granted.length + me.user.revoked.length > 0 && (
          <p className="text-[12.5px] text-warn">The owner made {me.user.granted.length + me.user.revoked.length} exception(s) to your role.</p>
        )}
        {!roles ? (
          <Skeleton className="h-60" />
        ) : (
          <div className="overflow-hidden rounded-xl border border-line" data-testid="help-access">
            {roles.permissions.map((p) => {
              const allowed = me.permissions.includes(p.key)
              const header = p.group !== group
              group = p.group
              return (
                <div key={p.key}>
                  {header && <div className="eyebrow border-b border-line bg-white/[.015] px-3.5 py-1.5">{p.group}</div>}
                  <div className="flex items-center gap-3 border-b border-line px-3.5 py-2 last:border-0">
                    <span className={allowed ? 'text-accent' : 'text-dim/60'} aria-label={allowed ? 'allowed' : 'not allowed'}>
                      {allowed ? <Check className="size-4" /> : <Minus className="size-4" />}
                    </span>
                    <div className="min-w-0 flex-1">
                      <b className={allowed ? 'block text-[13px] font-medium' : 'block text-[13px] font-medium text-muted'}>{p.label}</b>
                      <small className="block text-xs text-dim">{p.description}</small>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
        <p className="text-[12.5px] text-dim">Missing something? Ask an owner or admin; they manage roles in Team.</p>
      </Section>

      <Section icon={<Keyboard />} title="Shortcuts">
        <div className="overflow-hidden rounded-xl border border-line">
          {SHORTCUTS.map(([keys, what]) => (
            <div key={what} className="flex items-center gap-3 border-b border-line px-3.5 py-2.5 text-[13px] last:border-0">
              <span className="flex gap-1">
                {keys.map((k) => (
                  <kbd key={k} className="num min-w-6 rounded-md border border-line-2 bg-panel-2 px-1.5 py-0.5 text-center text-[11.5px] text-muted">
                    {k}
                  </kbd>
                ))}
              </span>
              <span className="text-muted">{what}</span>
            </div>
          ))}
        </div>
        <p className="text-[12.5px] text-dim">
          Anything linked to a customer, order, product or member opens on top of the page you are on. Close it to carry on where you were.
        </p>
      </Section>

      <Section icon={<BookOpen />} title="Documentation">
        <div className="overflow-hidden rounded-xl border border-line">
          {DOCS.map(([label, href]) => (
            <a key={label} href={href} target="_blank" rel="noreferrer" className="flex items-center gap-2 border-b border-line px-3.5 py-2.5 text-[13px] last:border-0 hover:bg-panel-2">
              {label}
              <ExternalLink className="ml-auto size-3.5 text-dim" />
            </a>
          ))}
        </div>
        {meta && (
          <p className="num text-[11.5px] text-dim" data-testid="help-version">
            {meta.serviceName} · api {meta.version} · {meta.goVersion} · {meta.env}
          </p>
        )}
      </Section>
    </Sheet>
  )
}

function Section({ icon, title, children }: { icon: ReactNode; title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2.5">
      <h3 className="flex items-center gap-2 text-[13.5px] font-medium [&>svg]:size-4 [&>svg]:text-dim">
        {icon}
        {title}
      </h3>
      {children}
    </section>
  )
}
