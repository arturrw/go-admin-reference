import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import { LinkIcon, UserCheck } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { Toaster } from 'sonner'
import { Button, buttonVariants } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/input'
import { Avatar, Skeleton } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { ApiError, api } from '@/lib/api'
import { meQuery } from '@/lib/auth'
import { cn } from '@/lib/utils'

/** The page an invitation link opens: pick a password and join. */
export function InvitePage() {
  const { token } = useParams({ from: '/invite/$token' })
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { data: info, isPending, error } = useQuery({ queryKey: ['invite', token], queryFn: () => api.inviteInfo(token), retry: false })
  const [name, setName] = useState<string | null>(null)
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const accept = useMutation({
    mutationFn: () => api.acceptInvite(token, { name: (name ?? info?.name ?? '').trim(), password }),
    onSuccess: (me) => {
      qc.clear() // nothing cached belongs to the new member
      qc.setQueryData(meQuery.queryKey, me)
      navigate({ to: '/', replace: true })
    },
  })
  const fields = accept.error instanceof ApiError ? accept.error.fields : undefined
  const mismatch = confirm !== '' && confirm !== password

  const submit = (e: FormEvent) => {
    e.preventDefault()
    if (!mismatch) accept.mutate()
  }

  return (
    <div className="relative grid min-h-screen place-items-center px-4 py-10">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0"
        style={{ background: 'radial-gradient(600px 300px at 50% 0%, color-mix(in srgb, var(--color-accent) 10%, transparent), transparent 70%)' }}
      />
      <div className="relative w-full max-w-100">
        <div className="mb-7 flex items-center gap-2.5">
          <div className="num grid size-9 place-items-center rounded-[10px] bg-accent text-sm font-bold text-accent-ink shadow-glow">Go</div>
          <div>
            <b className="block text-lg font-semibold tracking-[-0.02em]">{info?.serviceName ?? 'GoAdmin'}</b>
            <small className="text-dim">You've been invited to join</small>
          </div>
        </div>

        {isPending ? (
          <Skeleton className="h-72" />
        ) : error || !info ? (
          <div className="card flex flex-col items-center gap-3 p-6 text-center" data-testid="invite-invalid">
            <span className="grid size-10 place-items-center rounded-xl bg-danger/12 text-danger">
              <LinkIcon className="size-5" />
            </span>
            <b className="text-base font-semibold">This invitation can't be used</b>
            <p className="text-[13px] text-muted">The link has expired, was replaced by a newer one, or has already been used. Ask whoever invited you for a new link.</p>
            <Link to="/login" className={cn(buttonVariants(), 'mt-1')}>
              Go to sign in
            </Link>
          </div>
        ) : (
          <form onSubmit={submit} className="card flex flex-col gap-4 p-5.5">
            <div className="flex items-center gap-3 rounded-xl border border-line bg-panel-2/60 px-3 py-2.5">
              <Avatar name={info.name} size={34} />
              <div className="min-w-0 flex-1">
                <b className="block truncate text-[13.5px] font-medium">{info.email}</b>
                <small className="text-xs text-dim">Joining as</small>
              </div>
              <StatusPill status={info.role} />
            </div>
            <Field label="Your name" error={fields?.name}>
              <Input value={name ?? info.name} onChange={(e) => setName(e.target.value)} autoComplete="name" maxLength={80} />
            </Field>
            <Field label="Choose a password" error={fields?.password} hint="At least 8 characters.">
              <Input type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} autoFocus />
            </Field>
            <Field label="Repeat the password" error={mismatch ? 'The passwords do not match.' : undefined}>
              <Input type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
            </Field>
            {accept.error && !fields && <p className="text-[13px] text-danger">{accept.error.message}</p>}
            <Button variant="primary" type="submit" disabled={accept.isPending || password.length < 8 || confirm === '' || mismatch} className="mt-1">
              <UserCheck />
              {accept.isPending ? 'Creating your account…' : 'Create account and sign in'}
            </Button>
          </form>
        )}
      </div>
      <Toaster theme="dark" position="bottom-right" />
    </div>
  )
}
