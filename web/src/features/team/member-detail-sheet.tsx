import { Link } from '@tanstack/react-router'
import { ArrowRight, Ban, Check, Clock, History, KeyRound, Mail, RotateCcw, ShieldCheck, ShieldOff, UserCheck } from 'lucide-react'
import { Fragment, type ReactNode, useState } from 'react'
import { Button, buttonVariants } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/ui/dialog'
import { Avatar, Skeleton, Switch } from '@/components/ui/misc'
import { Pill, StatusPill } from '@/components/ui/pill'
import { Sheet } from '@/components/ui/sheet'
import { Tabs } from '@/components/ui/tabs'
import { ActivityItem } from '@/features/activity/activity-item'
import type { Member, MemberDetail, Permission, Role } from '@/lib/api'
import { useCan, useMe } from '@/lib/auth'
import { capitalize, timeAgo } from '@/lib/format'
import { useActivity, useMember, useRoles, useSaveMember, useSetMemberAccess, useSetMemberStatus } from '@/lib/queries'
import { cn } from '@/lib/utils'

type Tab = 'overview' | 'activity' | 'access'

export const ROLE_INFO: Record<Role, string> = {
  owner: 'Full access, billing, danger zone',
  admin: 'Manage everything except the danger zone',
  editor: 'Catalogue and order fulfilment',
  support: 'Orders, refunds and customer notes',
  viewer: 'Read-only dashboards and lists',
}

/** A team member in detail: profile, presence, what they did and what they may do. */
export function MemberDetailSheet({ memberId, onClose }: { memberId: number; onClose: () => void }) {
  const { data, isPending, isError } = useMember(memberId)
  const me = useMe()
  const canWrite = useCan('team:write')
  const [tab, setTab] = useState<Tab>('overview')
  const [confirmSuspend, setConfirmSuspend] = useState(false)
  const status = useSetMemberStatus()
  if (isError) return null
  const m = data?.member

  // Mirrors the API: nobody changes their own status or the owner's; only the owner manages admins.
  const canSuspend = !!m && canWrite && m.id !== me.user.id && m.role !== 'owner' && (m.role !== 'admin' || me.user.role === 'owner') && m.status !== 'invited'

  return (
    <Sheet
      open
      onOpenChange={(v) => !v && onClose()}
      size="lg"
      title={m?.name ?? 'Member'}
      description={m ? `${capitalize(m.role)} · ${m.email}` : 'Loading…'}
      footer={
        m && (
          <>
            <Link to="/activity" search={{ actor: m.id }} className={cn(buttonVariants(), 'mr-auto')}>
              <History />
              Full activity
            </Link>
            {canSuspend &&
              (m.status === 'suspended' ? (
                <Button variant="primary" disabled={status.isPending} onClick={() => status.mutate({ id: m.id, status: 'active' })}>
                  <UserCheck />
                  Reactivate
                </Button>
              ) : (
                <Button variant="danger" disabled={status.isPending} onClick={() => setConfirmSuspend(true)}>
                  <Ban />
                  Suspend
                </Button>
              ))}
          </>
        )
      }
    >
      {isPending || !data || !m ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-24" />
          <Skeleton className="h-72" />
        </div>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-3.5">
            <div className="relative">
              <Avatar name={m.name} size={52} />
              <span
                className={cn('absolute right-0 bottom-0 size-3.5 rounded-full border-[2.5px] border-panel', data.online ? 'bg-accent' : 'bg-dim')}
                aria-hidden
              />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <b className="text-lg font-semibold tracking-[-0.01em]">{m.name}</b>
                <StatusPill status={m.role} />
                {m.status !== 'active' && <StatusPill status={m.status} />}
              </div>
              <div className="mt-0.5 text-[12.5px] text-dim" data-testid="presence">
                {data.online ? (
                  <span className="text-accent">● Online now</span>
                ) : m.lastActiveAt ? (
                  `Last seen ${timeAgo(m.lastActiveAt)}`
                ) : (
                  'Never signed in'
                )}
              </div>
            </div>
          </div>

          <Tabs<Tab>
            value={tab}
            onChange={setTab}
            tabs={[
              { value: 'overview', label: 'Overview' },
              { value: 'activity', label: 'Activity' },
              { value: 'access', label: 'Access', count: data.permissions.length },
            ]}
          />

          {tab === 'overview' && <Overview detail={data} />}
          {tab === 'activity' && <RecentActivity member={m} />}
          {tab === 'access' && <Access key={`${m.role}-${m.granted.join()}-${m.revoked.join()}`} detail={data} />}

          <ConfirmDialog
            open={confirmSuspend}
            onOpenChange={setConfirmSuspend}
            title={`Suspend ${m.name}?`}
            description="They are signed out everywhere and can't sign in until reactivated. Their history stays."
            confirmLabel="Suspend"
            pending={status.isPending}
            onConfirm={() => status.mutate({ id: m.id, status: 'suspended' }, { onSuccess: () => setConfirmSuspend(false) })}
          />
        </>
      )}
    </Sheet>
  )
}

function Overview({ detail }: { detail: MemberDetail }) {
  const m = detail.member
  const { data: week } = useActivity({ actor: m.id, limit: 200 })
  const since = Date.now() - 7 * 86_400_000
  const recent = week?.items.filter((a) => new Date(a.at).getTime() > since) ?? []
  const changes = recent.filter((a) => a.kind !== 'auth').length
  const signIns = recent.length - changes
  const exceptions = m.granted.length + m.revoked.length

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <Fact icon={<Mail />} label="Email">
        <a href={`mailto:${m.email}`} className="hover:text-accent">
          {m.email}
        </a>
      </Fact>
      <Fact icon={<KeyRound />} label="Role">
        <span className="capitalize">{m.role}</span>
        <small className="block text-xs text-dim">{ROLE_INFO[m.role]}</small>
      </Fact>
      <Fact icon={<Clock />} label="Last active">
        {m.lastActiveAt ? (
          <>
            {timeAgo(m.lastActiveAt)}
            <small className="num block text-xs text-dim">{new Date(m.lastActiveAt).toLocaleString('en-US')}</small>
          </>
        ) : (
          '—'
        )}
      </Fact>
      <Fact icon={m.mfa ? <ShieldCheck /> : <ShieldOff />} label="Two-factor auth">
        {m.mfa ? 'Enabled' : <span className="text-warn">Not set up</span>}
      </Fact>
      <Fact icon={<History />} label="Last 7 days">
        <span className="num">{changes}</span> changes · <span className="num">{signIns}</span> sign-ins
      </Fact>
      <Fact icon={<KeyRound />} label="Permissions">
        <span className="num">{detail.permissions.length}</span> granted
        {exceptions > 0 && <small className="block text-xs text-warn">{exceptions} exception(s) to the role</small>}
      </Fact>
    </div>
  )
}

function Fact({ icon, label, children }: { icon: ReactNode; label: string; children: ReactNode }) {
  return (
    <div className="card flex gap-3 px-3.5 py-3 [&>svg]:mt-0.5 [&>svg]:size-4 [&>svg]:shrink-0 [&>svg]:text-dim">
      {icon}
      <div className="min-w-0 text-[13px]">
        <div className="eyebrow mb-0.5">{label}</div>
        {children}
      </div>
    </div>
  )
}

function RecentActivity({ member }: { member: Member }) {
  const { data, isPending } = useActivity({ actor: member.id, limit: 15 })
  if (isPending) return <Skeleton className="h-60" />
  if (!data?.items.length) return <p className="py-8 text-center text-muted">No activity yet.</p>
  return (
    <div className="flex flex-col">
      {data.items.map((a) => (
        <ActivityItem key={a.id} a={a} />
      ))}
      {data.total > data.items.length && (
        <Link to="/activity" search={{ actor: member.id }} className="mt-2 flex items-center gap-1.5 self-start text-[12.5px] text-muted hover:text-fg">
          All {data.total} entries <ArrowRight className="size-3.5" />
        </Link>
      )}
    </div>
  )
}

/**
 * The member's permissions. The owner can switch any of them on or off: a
 * difference from the role is stored as a grant or a revocation.
 */
function Access({ detail }: { detail: MemberDetail }) {
  const m = detail.member
  const me = useMe()
  const { data: roles } = useRoles()
  const save = useSetMemberAccess()
  const saveRole = useSaveMember()
  const [draft, setDraft] = useState<Set<Permission>>(() => new Set(detail.permissions))
  const canChangeRole = useCanChangeRole(m)
  if (!roles) return <Skeleton className="h-96" />

  const isOwner = me.user.role === 'owner'
  const editable = isOwner && m.role !== 'owner' && m.id !== me.user.id
  const fromRole = new Set(roles.matrix[m.role])
  const granted = [...draft].filter((p) => !fromRole.has(p))
  const revoked = [...fromRole].filter((p) => !draft.has(p))
  const dirty = draft.size !== detail.permissions.length || detail.permissions.some((p) => !draft.has(p))
  const toggle = (p: Permission, on: boolean) =>
    setDraft((d) => {
      const next = new Set(d)
      if (on) next.add(p)
      else next.delete(p)
      return next
    })
  let lastGroup = ''

  return (
    <div className="flex flex-col gap-4">
      {canChangeRole && (
        <section>
          <div className="eyebrow mb-2">Role</div>
          <div role="radiogroup" aria-label="Role" className="flex flex-wrap gap-1.5">
            {roles.roles
              .filter((r) => r !== 'owner' && (r !== 'admin' || isOwner))
              .map((r) => (
                <button
                  key={r}
                  type="button"
                  role="radio"
                  aria-checked={m.role === r}
                  disabled={saveRole.isPending}
                  title={ROLE_INFO[r]}
                  onClick={() => m.role !== r && saveRole.mutate({ id: m.id, input: { name: m.name, email: m.email, role: r } })}
                  className={cn(
                    'rounded-lg border px-3 py-1.5 text-[12.5px] capitalize transition-colors',
                    m.role === r ? 'border-accent/50 bg-accent/10 text-fg' : 'border-line-2 text-muted hover:text-fg',
                  )}
                >
                  {r}
                </button>
              ))}
          </div>
        </section>
      )}

      {!editable && (
        <p className="rounded-xl border border-line bg-panel-2 px-3.5 py-2.5 text-[12.5px] text-muted">
          {m.role === 'owner' ? 'The owner always has full access.' : 'Only the owner can change individual permissions.'}
        </p>
      )}

      <div className="overflow-hidden rounded-xl border border-line" data-testid="access">
        {roles.permissions.map((p) => {
          const header = p.group !== lastGroup
          lastGroup = p.group
          const on = draft.has(p.key)
          const source = fromRole.has(p.key) ? (on ? 'role' : 'revoked') : on ? 'granted' : null
          const locked = !editable || p.key === 'workspace:manage'
          return (
            <Fragment key={p.key}>
              {header && <div className="eyebrow border-b border-line bg-white/[.015] px-3.5 py-1.5">{p.group}</div>}
              <div className="flex items-center gap-3 border-b border-line px-3.5 py-2.5 last:border-0">
                <div className="min-w-0 flex-1">
                  <b className={cn('block text-[13px] font-medium', !on && 'text-muted')}>{p.label}</b>
                  <small className="block text-xs text-dim">{p.description}</small>
                </div>
                {source === 'granted' && <Pill tone="lime">granted</Pill>}
                {source === 'revoked' && <Pill tone="red">revoked</Pill>}
                {source === 'role' && <span className="text-[11px] text-dim">from role</span>}
                {locked ? (
                  on ? (
                    <Check className="size-4 text-accent" aria-label={`${p.label}: allowed`} />
                  ) : (
                    <span className="w-4 text-center text-dim" aria-label={`${p.label}: not allowed`}>
                      –
                    </span>
                  )
                ) : (
                  <Switch checked={on} onChange={(v) => toggle(p.key, v)} label={p.label} />
                )}
              </div>
            </Fragment>
          )
        })}
      </div>

      {editable && (
        <div className="sticky -bottom-4.5 -mx-4.5 -mb-4.5 flex flex-wrap items-center gap-2 border-t border-line bg-panel px-4.5 py-3">
          <small className="mr-auto text-xs text-dim">
            {granted.length || revoked.length ? `${granted.length} granted · ${revoked.length} revoked vs ${m.role}` : `Same as the ${m.role} role`}
          </small>
          <Button size="sm" disabled={!dirty && !granted.length && !revoked.length} onClick={() => setDraft(new Set(fromRole))}>
            <RotateCcw />
            Role defaults
          </Button>
          <Button size="sm" variant="primary" disabled={!dirty || save.isPending} onClick={() => save.mutate({ id: m.id, granted, revoked })}>
            <Check />
            Save access
          </Button>
        </div>
      )}
    </div>
  )
}

/** Same rules as the API: only the owner touches admins, and nobody changes the owner or themselves. */
function useCanChangeRole(m: Member) {
  const me = useMe()
  const canWrite = useCan('team:write')
  return canWrite && m.role !== 'owner' && m.id !== me.user.id && (m.role !== 'admin' || me.user.role === 'owner') && m.status !== 'suspended'
}
