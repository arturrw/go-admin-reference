import { useNavigate, useSearch } from '@tanstack/react-router'
import { Check, Pencil, Send, ShieldCheck, ShieldOff, Trash2, UserPlus } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, TableCard } from '@/components/ui/card'
import { Field, Input } from '@/components/ui/input'
import { Avatar, PageHeader, Skeleton } from '@/components/ui/misc'
import { Pill, STATUS_TONE, StatusPill } from '@/components/ui/pill'
import { Segmented } from '@/components/ui/segmented'
import { Sheet } from '@/components/ui/sheet'
import { ApiError, type Member, type Role } from '@/lib/api'
import { capitalize, timeAgo } from '@/lib/format'
import { useDeleteMember, useSaveMember, useTeam } from '@/lib/queries'
import { cn } from '@/lib/utils'

const ROLE_INFO: Record<Role, string> = {
  owner: 'Full access, billing, can delete workspace',
  admin: 'Manage everything except billing',
  editor: 'Create & edit products and orders',
  support: 'View orders, issue refunds',
  viewer: 'Read-only access to dashboards',
}
const ROLES = Object.keys(ROLE_INFO) as Role[]

export function TeamPage() {
  const { edit } = useSearch({ from: '/team' })
  const navigate = useNavigate({ from: '/team' })
  const [role, setRole] = useState<Role | 'all'>('all')
  const { data, isPending } = useTeam(role)
  const { data: everyone } = useTeam('all')
  const remove = useDeleteMember()

  const editing = typeof edit === 'number' ? everyone?.items.find((m) => m.id === edit) : undefined
  const sheetOpen = edit === 'new' || !!editing
  const close = () => navigate({ search: {} })

  return (
    <>
      <PageHeader title="Team & roles" description="Who can access this admin and what they can do.">
        <Button variant="primary" onClick={() => navigate({ search: { edit: 'new' } })}>
          <UserPlus />
          Invite member
        </Button>
      </PageHeader>

      <div className="mb-3.5 grid grid-cols-1 gap-3.5 sm:grid-cols-2 xl:grid-cols-4">
        {(['owner', 'admin', 'editor', 'viewer'] as const).map((r) => (
          <Card key={r}>
            <div className="flex items-center gap-2">
              <Pill tone={STATUS_TONE[r]}>{r}</Pill>
              <span className="num ml-auto text-xs text-dim">{everyone?.items.filter((m) => m.role === r).length ?? '—'} members</span>
            </div>
            <p className="mt-2.5 text-[12.5px] text-muted">{ROLE_INFO[r]}</p>
          </Card>
        ))}
      </div>

      <div className="mb-3.5">
        <Segmented value={role} onChange={setRole} options={(['all', ...ROLES] as const).map((r) => ({ value: r, label: capitalize(r) }))} />
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
                <tr key={m.id}>
                  <td>
                    <div className="flex items-center gap-2.5">
                      <Avatar name={m.name} />
                      <div>
                        <b className="block font-medium">{m.name}</b>
                        <small className="block text-xs text-dim">{m.email}</small>
                      </div>
                    </div>
                  </td>
                  <td>
                    <StatusPill status={m.role} />
                  </td>
                  <td>{m.status === 'active' ? <span className="text-muted">Active</span> : <StatusPill status={m.status} />}</td>
                  <td>{m.mfa ? <ShieldCheck className="size-4 text-accent" /> : <ShieldOff className="size-4 text-dim" />}</td>
                  <td className="num text-muted">{m.lastActiveAt ? timeAgo(m.lastActiveAt) : '—'}</td>
                  <td className="num">
                    {m.role !== 'owner' && (
                      <>
                        <Button variant="ghost" size="icon-sm" aria-label={`Edit ${m.name}`} onClick={() => navigate({ search: { edit: m.id } })}>
                          <Pencil />
                        </Button>
                        <Button variant="ghost" size="icon-sm" aria-label={`Remove ${m.name}`} disabled={remove.isPending} onClick={() => remove.mutate(m.id)}>
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

      <MemberSheet key={edit === 'new' ? 'new' : editing ? `m${editing.id}` : 'closed'} member={editing} open={sheetOpen} onClose={close} />
    </>
  )
}

function MemberSheet({ member, open, onClose }: { member?: Member; open: boolean; onClose: () => void }) {
  const [name, setName] = useState(member?.name ?? '')
  const [email, setEmail] = useState(member?.email ?? '')
  const [role, setRole] = useState<Role>(member?.role ?? 'viewer')
  const save = useSaveMember()
  const errors = save.error instanceof ApiError ? save.error.fields : undefined

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
            {ROLES.filter((r) => r !== 'owner').map((r) => (
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
        {save.error && !errors && <p className="text-[13px] text-danger">{save.error.message}</p>}
      </form>
    </Sheet>
  )
}
