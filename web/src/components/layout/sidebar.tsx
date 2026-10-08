import { Link, useRouterState } from '@tanstack/react-router'
import { LogOut } from 'lucide-react'
import type { MouseEvent } from 'react'
import { Button } from '@/components/ui/button'
import { Avatar } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import { useLogout, useMe } from '@/lib/auth'
import { visibleNav } from '@/lib/nav'
import { useMeta } from '@/lib/queries'
import { cn } from '@/lib/utils'

export function Sidebar({ open, onNavigate }: { open: boolean; onNavigate: () => void }) {
  const { data: meta } = useMeta()
  const me = useMe()
  const logout = useLogout()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const searchStr = useRouterState({ select: (s) => s.location.searchStr })
  // Clicking the page you're already on scrolls it back to the top. Navigating
  // to the same URL would let scroll restoration put the old position back.
  const go = (to: string) => (e: MouseEvent) => {
    if (pathname === to && !searchStr) {
      e.preventDefault()
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
    onNavigate()
  }

  return (
    <aside
      className={cn(
        'z-40 flex h-screen flex-col overflow-y-auto border-r border-line bg-side px-3 py-3.5',
        'max-md:fixed max-md:inset-y-0 max-md:left-0 max-md:w-66 max-md:transition-transform max-md:duration-250',
        open ? 'max-md:translate-x-0 max-md:shadow-[30px_0_60px_rgb(0_0_0/.6)]' : 'max-md:-translate-x-full',
        'md:sticky md:top-0',
      )}
    >
      <Link to="/" onClick={go('/')} aria-label="GoAdmin — go to dashboard" className="flex items-center gap-2.5 rounded-xl px-2 pt-1.5 pb-2.5 transition-opacity hover:opacity-85">
        <div className="num grid size-8 place-items-center rounded-[9px] bg-accent text-[13px] font-bold text-accent-ink shadow-[0_8px_24px_-8px_color-mix(in_srgb,var(--color-accent)_70%,transparent)]">
          Go
        </div>
        <div>
          <b className="block font-semibold tracking-[-0.01em]">GoAdmin</b>
          <small className="num block text-[11px] text-dim">{meta?.serviceName ?? '…'} · {meta?.env ?? '…'}</small>
        </div>
      </Link>

      <nav>
        {visibleNav(me.permissions).map((group) => (
          <div key={group.label}>
            <div className="num px-2.5 pt-4 pb-1.5 text-[10.5px] font-medium tracking-widest text-dim uppercase">{group.label}</div>
            {group.items.map((item) => {
              const badge = item.badge && meta?.[item.badge]
              return (
                <Link
                  key={item.to}
                  to={item.to}
                  onClick={go(item.to)}
                  activeOptions={{ exact: item.to === '/', includeSearch: false }}
                  className="group relative flex items-center gap-2.5 rounded-[9px] px-2.5 py-2 font-medium text-muted transition-colors hover:bg-panel-2 hover:text-fg data-[status=active]:bg-panel-3 data-[status=active]:text-fg"
                >
                  <span className="absolute top-2 bottom-2 -left-3 hidden w-[3px] rounded-r-[3px] bg-accent shadow-[0_0_14px_var(--color-accent)] group-data-[status=active]:block" />
                  <item.icon className="size-4 group-data-[status=active]:text-accent" />
                  {item.label}
                  {badge ? (
                    <span
                      className={cn(
                        'num ml-auto rounded-full px-[7px] py-px text-[11px] font-medium',
                        item.badge === 'pendingOrders'
                          ? 'bg-warn/16 text-warn'
                          : 'bg-panel-3 text-muted group-data-[status=active]:bg-accent/14 group-data-[status=active]:text-accent',
                      )}
                    >
                      {badge}
                    </span>
                  ) : null}
                </Link>
              )
            })}
          </div>
        ))}
      </nav>

      <div className="mt-auto flex flex-col gap-2.5 pt-3.5">
        <div className="rounded-xl border border-line bg-panel p-3">
          <div className="flex items-center gap-2 text-[12.5px]">
            <span className="live-dot" />
            <b className="font-medium">All systems normal</b>
          </div>
          <div className="num mt-1 text-[11.5px] text-muted">
            api {meta?.version ?? '…'} · {meta?.goVersion ?? '…'}
          </div>
        </div>
        <div className="flex items-center gap-2.5 rounded-[10px] p-2">
          <Avatar name={me.user.name} />
          <div className="min-w-0 flex-1">
            <b className="block truncate text-[13px] font-medium" data-testid="current-user">
              {me.user.name}
            </b>
            <StatusPill status={me.user.role} />
          </div>
          <Button variant="ghost" size="icon-sm" aria-label="Sign out" title="Sign out" onClick={() => logout.mutate()}>
            <LogOut />
          </Button>
        </div>
      </div>
    </aside>
  )
}
