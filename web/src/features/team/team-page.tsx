import { useNavigate, useSearch } from '@tanstack/react-router'
import { Check, Minus, Pencil, Send, ShieldCheck, ShieldOff, Trash2, UserPlus } from 'lucide-react'
import { type FormEvent, Fragment, useState } from 'react'
import { Button } from '@/components/ui/button'
import { TableCard } from '@/components/ui/card'
import { Field, Input } from '@/components/ui/input'
import { Avatar, PageHeader, Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { Segmented } from '@/components/ui/segmented'
import { ConfirmDialog } from '@/components/ui/dialog'
import { Sheet } from '@/components/ui/sheet'
import { ApiError, type Member, type Role } from '@/lib/api'
import { useCan, useMe } from '@/lib/auth'
import { capitalize, timeAgo } from '@/lib/format'
import { usePeek } from '@/lib/peek'
import { useDeleteMember, useRoles, useSaveMember, useTeam } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { ROLE_INFO } from './member-detail-sheet'

/** Matches domain.OnlineWithin: a request in the last five minutes. */
const isOnline = (m: Member) => m.status === 'active' && !!m.lastActiveAt && Date.now() - new Date(m.lastActiveAt).getTime() < 5 * 60_000

const ROLES = Object.keys(ROLE_INFO) as Role[]

export function TeamPage() {
  const { edit } = useSearch({ from: '/app/team' })
  const navigate = useNavigate({ from: '/team' })
  const me = useMe()
  const canWrite = useCan('team:write')
  const [role, setRole] = useState<Role | 'all'>('all')
  const { data, isPending } = useTeam(role)
  const { data: everyone } = useTeam('all')
  const remove = useDeleteMember()
  const peek = usePeek()
  const [removing, setRemoving] = useState<Member | null>(null)

  // Mirrors the server rules: the owner is untouchable, only the owner manages admins, nobody removes themselves.
  const canManage = (m: Member) => canWrite && m.role !== 'owner' && (m.role !== 'admin' || me.user.role === 'owner')

  const editing = typeof edit === 'number' ? everyone?.items.find((m) => m.id === edit) : undefined
  const sheetOpen = canWrite && (edit === 'new' || !!editing)
  const close = () => navigate({ search: {} })

  return (
    <>
      <PageHeader title="Team & roles" description="Who can access this admin and what each role is allowed to do.">
        {canWrite && (
          <Button variant="primary" onClick={() => navigate({ search: { edit: 'new' } })}>
            <UserPlus />
            Invite member
          </Button>
        )}
      </PageHeader>

      <PermissionMatrix myRole={me.user.role} counts={Object.fromEntries(ROLES.map((r) => [r, everyone?.items.filter((m) => m.role === r).length ?? 0]))} />

      <div className="mt-6 mb-3.5 flex items-center gap-3">
        <h2 className="text-[15px] font-medium">Members</h2>
        <Segmented className="ml-auto" value={role} onChange={setRole} options={(['all', ...ROLES] as const).map((r) => ({ value: r, label: capitalize(r) }))} />
      </div>

      {isPending ? (
        <Skeleton className="h-100" />
      ) : (
        <TableCard>
          <table className="data-table">
            <thead>
              <tr>
                <th>Member</th>
                <th>Role</th>
                <th>Status</th>
                <th>2FA</th>
                <th className="num">Last active</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {data?.items.map((m) => (
                <tr
                  key={m.id}
                  role="button"
                  tabIndex={0}
                  aria-label={`Open ${m.name}`}
                  onClick={() => peek('member', m.id)}
                  onKeyDown={(e) => e.target === e.currentTarget && (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), peek('member', m.id))}
                  className={cn('cursor-pointer', m.id === me.user.id && 'bg-accent/4')}
                >
                  <td>
                    <div className="flex items-center gap-2.5">
                      <div className="relative">
                        <Avatar name={m.name} />
                        {isOnline(m) && <span className="absolute -right-px -bottom-px size-2.5 rounded-full border-2 border-panel bg-accent" title="Online" />}
                      </div>
                      <div>
                        <b className="block font-medium">
                          {m.name}
                          {m.id === me.user.id && <span className="ml-1.5 text-xs font-normal text-dim">(you)</span>}
                        </b>
                        <small className="block text-xs text-dim">{m.email}</small>
                      </div>
                    </div>
                  </td>
                  <td>
                    <StatusPill status={m.role} />
                  </td>
                  <td>{m.status === 'active' ? <span className="text-muted">Active</span> : <StatusPill status={m.status} />}</td>
                  <td>{m.mfa ? <ShieldCheck className="size-4 text-accent" /> : <ShieldOff className="size-4 text-dim" />}</td>
                  <td className="num text-muted">{isOnline(m) ? <span className="text-accent">online</span> : m.lastActiveAt ? timeAgo(m.lastActiveAt) : '—'}</td>
                  <td className="num" onClick={(e) => e.stopPropagation()}>
                    {canManage(m) && m.id !== me.user.id && (
                      <>
                        <Button variant="ghost" size="icon-sm" aria-label={`Edit ${m.name}`} onClick={() => navigate({ search: { edit: m.id } })}>
                          <Pencil />
                        </Button>
                        <Button variant="ghost" size="icon-sm" aria-label={`Remove ${m.name}`} disabled={remove.isPending} onClick={() => setRemoving(m)}>
                          <Trash2 />
                        </Button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableCard>
      )}

      <MemberSheet key={String(edit ?? 'closed')} member={editing} open={sheetOpen} onClose={close} />
      <ConfirmDialog
        open={!!removing}
        onOpenChange={(v) => !v && setRemoving(null)}
        title={`Remove ${removing?.name ?? 'member'}?`}
        description="They lose access immediately and are signed out everywhere. Their activity stays in the log."
        confirmLabel="Remove"
        pending={remove.isPending}
        onConfirm={() => removing && remove.mutate(removing.id, { onSuccess: () => setRemoving(null) })}
      />
    </>
  )
}

/** Roles × permissions, straight from GET /api/v1/roles — the same matrix the API enforces. */
function PermissionMatrix({ myRole, counts }: { myRole: Role; counts: Record<string, number> }) {
  const { data } = useRoles()
  if (!data) return <Skeleton className="h-96" />
  let lastGroup = ''
  return (
    <TableCard>
      <table className="data-table" data-testid="permission-matrix">
        <thead>
          <tr>
            <th className="min-w-55">Permission</th>
            {data.roles.map((r) => (
              <th key={r} className={cn('text-center!', r === myRole && 'bg-accent/8! text-accent!')}>
                <div className="flex flex-col items-center gap-0.5">
                  {r}
                  <span className="text-[10px] tracking-normal normal-case opacity-70">{counts[r] ?? 0} members</span>
                </div>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.permissions.map((p) => {
            const header = p.group !== lastGroup
            lastGroup = p.group
            return (
              <Fragment key={p.key}>
                {header && (
                  <tr className="hover:bg-transparent!">
                    <td colSpan={data.roles.length + 1} className="eyebrow bg-white/[.015] py-1.5!">
                      {p.group}
                    </td>
                  </tr>
                )}
                <tr>
                  <td>
                    <b className="block font-medium">{p.label}</b>
                    <small className="block text-xs whitespace-normal text-dim">{p.description}</small>
                  </td>
                  {data.roles.map((r) => (
                    <td key={r} className={cn('text-center', r === myRole && 'bg-accent/4')}>
                      {data.matrix[r].includes(p.key) ? (
                        <Check className="inline size-4 text-accent" aria-label={`${r}: allowed`} />
                      ) : (
                        <Minus className="inline size-4 text-dim/60" aria-label={`${r}: not allowed`} />
                      )}
                    </td>
                  ))}
                </tr>
              </Fragment>
            )
          })}
          <tr className="hover:bg-transparent!">
            <td className="text-xs whitespace-normal text-dim" colSpan={data.roles.length + 1}>
              Extra rules: the owner can’t be edited or removed, only the owner can grant or manage the admin role, and nobody can remove themselves. Your role is
              highlighted.
            </td>
          </tr>
        </tbody>
      </table>
    </TableCard>
  )
}

function MemberSheet({ member, open, onClose }: { member?: Member; open: boolean; onClose: () => void }) {
  const me = useMe()
  const [name, setName] = useState(member?.name ?? '')
  const [email, setEmail] = useState(member?.email ?? '')
  const [role, setRole] = useState<Role>(member?.role ?? 'viewer')
  const save = useSaveMember()
  const errors = save.error instanceof ApiError ? save.error.fields : undefined
  const assignable = ROLES.filter((r) => r !== 'owner' && (r !== 'admin' || me.user.role === 'owner'))

  const submit = (e: FormEvent) => {
    e.preventDefault()
    save.mutate({ id: member?.id, input: { name, email, role } }, { onSuccess: onClose })
  }

  return (
    <Sheet
      open={open}
      onOpenChange={(o) => !o && onClose()}
      title={member ? 'Edit member' : 'Invite member'}
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" type="submit" form="member-form" disabled={save.isPending}>
            {member ? <Check /> : <Send />}
            {member ? 'Save' : 'Send invite'}
          </Button>
        </>
      }
    >
      <form id="member-form" onSubmit={submit} className="flex flex-col gap-4">
        <Field label="Full name" error={errors?.name}>
          <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        </Field>
        <Field label="Email" error={errors?.email}>
          <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="name@acme.io" />
        </Field>
        <Field label="Role" error={errors?.role}>
          <div role="radiogroup" className="flex flex-col gap-2">
            {assignable.map((r) => (
              <button
                key={r}
                type="button"
                role="radio"
                aria-checked={role === r}
                onClick={() => setRole(r)}
                className={cn(
                  'flex items-center justify-between gap-3.5 rounded-xl border bg-panel px-3.5 py-3 text-left transition-colors',
                  role === r ? 'border-accent/45' : 'border-line hover:border-line-2',
                )}
              >
                <div>
                  <b className="block font-medium capitalize">{r}</b>
                  <small className="text-xs text-dim">{ROLE_INFO[r]}</small>
                </div>
                <span className={cn('grid size-4 place-items-center rounded-full border-[1.5px]', role === r ? 'border-accent' : 'border-line-2')}>
                  {role === r && <i className="size-2 rounded-full bg-accent" />}
                </span>
              </button>
            ))}
          </div>
        </Field>
      </form>
    </Sheet>
  )
}
