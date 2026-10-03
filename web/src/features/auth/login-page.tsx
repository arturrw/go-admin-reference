import { useQuery } from '@tanstack/react-query'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { ArrowRight, LogIn } from 'lucide-react'
import { type FormEvent, useState } from 'react'
import { Toaster } from 'sonner'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/input'
import { Avatar } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { api } from '@/lib/api'
import { useLogin } from '@/lib/auth'

export function LoginPage() {
  const { redirect } = useSearch({ from: '/login' })
  const navigate = useNavigate()
  const login = useLogin()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  // Only available in development builds of the API.
  const { data: demo } = useQuery({ queryKey: ['demo-accounts'], queryFn: api.demoAccounts, retry: false, staleTime: Infinity })

  const signIn = (e: string, p: string) =>
    login.mutate(
      { email: e, password: p },
      {
        onSuccess: () => {
          // Only follow same-app relative redirects.
          const to = redirect?.startsWith('/') && !redirect.startsWith('//') && !redirect.startsWith('/login') ? redirect : '/'
          navigate({ href: to, replace: true })
        },
      },
    )

  const submit = (e: FormEvent) => {
    e.preventDefault()
    signIn(email, password)
  }

  return (
    <div className="relative grid min-h-screen place-items-center px-4 py-10">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            'radial-gradient(600px 300px at 50% 0%, color-mix(in srgb, var(--color-accent) 10%, transparent), transparent 70%)',
        }}
      />
      <div className="relative w-full max-w-100">
        <div className="mb-7 flex items-center gap-2.5">
          <div className="num grid size-9 place-items-center rounded-[10px] bg-accent text-sm font-bold text-accent-ink shadow-glow">Go</div>
          <div>
            <b className="block text-lg font-semibold tracking-[-0.02em]">GoAdmin</b>
            <small className="text-dim">Sign in to the Acme workspace</small>
          </div>
        </div>

        <form onSubmit={submit} className="card flex flex-col gap-4 p-5.5">
          <Field label="Email">
            <Input type="email" autoComplete="username" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@acme.io" autoFocus />
          </Field>
          <Field label="Password">
            <Input type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </Field>
          {login.error && <p className="text-[13px] text-danger">{login.error.message}</p>}
          <Button variant="primary" type="submit" disabled={login.isPending} className="mt-1">
            <LogIn />
            {login.isPending ? 'Signing in…' : 'Sign in'}
          </Button>
        </form>

        {demo && (
          <div className="mt-5">
            <div className="eyebrow mb-2">Demo accounts · password “{demo.password}”</div>
            <div className="flex flex-col gap-1.5">
              {demo.accounts.map((a) => (
                <button
                  key={a.email}
                  onClick={() => signIn(a.email, demo.password)}
                  disabled={login.isPending}
                  className="group flex items-center gap-3 rounded-xl border border-line bg-panel px-3 py-2.5 text-left transition-colors hover:border-accent/35"
                >
                  <Avatar name={a.name} size={28} />
                  <div className="min-w-0 flex-1">
                    <b className="block text-[13px] font-medium">{a.name}</b>
                    <small className="block truncate text-xs text-dim">{a.email}</small>
                  </div>
                  <StatusPill status={a.role} />
                  <ArrowRight className="size-4 text-dim transition-transform group-hover:translate-x-0.5 group-hover:text-accent" />
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
      <Toaster theme="dark" position="bottom-right" />
    </div>
  )
}
