import { Link } from '@tanstack/react-router'
import { ChevronsUpDown, Settings2 } from 'lucide-react'
import { Avatar } from '@/components/ui/misc'
import { NAV } from '@/lib/nav'
import { useMeta } from '@/lib/queries'
import { cn } from '@/lib/utils'

export function Sidebar({ open, onNavigate }: { open: boolean; onNavigate: () => void }) {
  const { data: meta } = useMeta()

  return (
    <aside
      className={cn(
        'z-40 flex h-screen flex-col overflow-y-auto border-r border-line bg-side px-3 py-3.5',
        'max-md:fixed max-md:inset-y-0 max-md:left-0 max-md:w-66 max-md:transition-transform max-md:duration-250',
        open ? 'max-md:translate-x-0 max-md:shadow-[30px_0_60px_rgb(0_0_0/.6)]' : 'max-md:-translate-x-full',
        'md:sticky md:top-0',
      )}
    >
      <div className="flex items-center gap-2.5 px-2 pt-1.5 pb-2.5">
        <div className="num grid size-8 place-items-center rounded-[9px] bg-accent text-[13px] font-bold text-accent-ink shadow-[0_8px_24px_-8px_color-mix(in_srgb,var(--color-accent)_70%,transparent)]">
          Go
        </div>
        <div>
          <b className="block font-semibold tracking-[-0.01em]">GoAdmin</b>
          <small className="num block text-[11px] text-dim">acme · {meta?.env ?? '…'}</small>
        </div>
        <ChevronsUpDown className="ml-auto size-4 text-dim" />
      </div>

      <nav>
        {NAV.map((group) => (
          <div key={group.label}>
            <div className="num px-2.5 pt-4 pb-1.5 text-[10.5px] font-medium tracking-widest text-dim uppercase">{group.label}</div>
            {group.items.map((item) => {
              const badge = item.badge && meta?.[item.badge]
              return (
                <Link
                  key={item.to}
                  to={item.to}
                  onClick={onNavigate}
                  activeOptions={{ exact: item.to === '/' }}
                  className="group relative flex items-center gap-2.5 rounded-[9px] px-2.5 py-2 font-medium text-muted transition-colors hover:bg-panel-2 hover:text-fg data-[status=active]:bg-panel-3 data-[status=active]:text-fg"
                >
                  <span className="absolute top-2 bottom-2 -left-3 hidden w-[3px] rounded-r-[3px] bg-accent shadow-[0_0_14px_var(--color-accent)] group-data-[status=active]:block" />
                  <item.icon className="size-4 group-data-[status=active]:text-accent" />
                  {item.label}
                  {badge ? (
                    <span
                      className={cn(
                        'num ml-auto rounded-full px-[7px] py-px text-[11px] font-medium',
                        item.badge === 'pendingOrders' ? 'bg-warn/16 text-warn' : 'bg-panel-3 text-muted group-data-[status=active]:bg-accent/14 group-data-[status=active]:text-accent',
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
          <div className="mt-2.5 flex h-4.5 items-end gap-0.5">
            {Array.from({ length: 30 }, (_, i) => (
              <i
                key={i}
                className={cn('flex-1 rounded-[2px] opacity-85', i === 17 ? 'bg-warn' : 'bg-accent')}
                style={{ height: `${i === 17 ? 55 : 80 + ((i * 37) % 20)}%` }}
              />
            ))}
          </div>
        </div>
        <div className="flex items-center gap-2.5 rounded-[10px] p-2 hover:bg-panel-2">
          <Avatar name="Anna Petrova" />
          <div className="min-w-0">
            <b className="block text-[13px] font-medium">Anna Petrova</b>
            <small className="block truncate text-xs text-dim">anna@acme.io</small>
          </div>
          <Settings2 className="ml-auto size-4 text-dim" />
        </div>
      </div>
    </aside>
  )
}
