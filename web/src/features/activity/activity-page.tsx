import { useSearch } from '@tanstack/react-router'
import { History } from 'lucide-react'
import { useDeferredValue, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { SearchInput, Select } from '@/components/ui/input'
import { Avatar, EmptyState, PageHeader, Skeleton, StatStrip } from '@/components/ui/misc'
import type { Activity, ActivityKind } from '@/lib/api'
import { int } from '@/lib/format'
import { useActivity, useTeam } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { ACTIVITY_KIND, ActivityItem } from './activity-item'

const PAGE = 50

const dayHeading = (iso: string) => {
  const d = new Date(iso)
  const today = new Date()
  const yesterday = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1)
  if (d.toDateString() === today.toDateString()) return 'Today'
  if (d.toDateString() === yesterday.toDateString()) return 'Yesterday'
  return d.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' })
}

/** Every change staff made in the admin, newest first, grouped by day. */
export function ActivityPage() {
  const search = useSearch({ from: '/app/activity' })
  const [actor, setActor] = useState(search.actor ?? 0)
  const [kind, setKind] = useState<ActivityKind | ''>('')
  const [q, setQ] = useState('')
  const [limit, setLimit] = useState(PAGE)
  const filter = { actor: actor || undefined, kind: kind || undefined, q: useDeferredValue(q) || undefined, limit }
  const { data, isPending, isFetching } = useActivity(filter)
  const { data: team } = useTeam('all')
  const { data: today } = useActivity({ limit: 200 })

  const reset = <T,>(set: (v: T) => void) => (v: T) => {
    set(v)
    setLimit(PAGE)
  }

  // Headline numbers over the latest 200 entries.
  const recent = today?.items ?? []
  const dayAgo = Date.now() - 86_400_000
  const last24 = recent.filter((a) => new Date(a.at).getTime() > dayAgo)
  const busiest = Object.entries(
    last24.filter((a) => a.actorId).reduce<Record<string, number>>((m, a) => ({ ...m, [a.actor]: (m[a.actor] ?? 0) + 1 }), {}),
  ).sort((a, b) => b[1] - a[1])[0]

  return (
    <>
      <PageHeader title="Activity log" description="Everything your team changed in this admin — products, orders, refunds, notes, roles and settings." />

      <StatStrip
        items={[
          { label: 'Entries', value: today ? int(today.total) : '—' },
          { label: 'Last 24 hours', value: int(last24.length) },
          { label: 'Changes (24h)', value: int(last24.filter((a) => a.kind !== 'auth').length) },
          { label: 'Sign-ins (24h)', value: int(last24.filter((a) => a.kind === 'auth').length) },
          { label: 'Most active (24h)', value: busiest ? <span className="text-[17px]">{busiest[0]}</span> : '—' },
        ]}
      />

      <div className="mb-3.5 flex flex-wrap items-center gap-2.5">
        <SearchInput placeholder="Search actions…" value={q} onChange={(e) => reset(setQ)(e.target.value)} />
        <Select className="w-48" aria-label="Member" value={actor} onChange={(e) => reset(setActor)(Number(e.target.value))}>
          <option value={0}>Everyone</option>
          {team?.items.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </Select>
        <Select className="w-44" aria-label="Type" value={kind} onChange={(e) => reset(setKind)(e.target.value as ActivityKind | '')}>
          <option value="">All types</option>
          {(Object.keys(ACTIVITY_KIND) as ActivityKind[]).map((k) => (
            <option key={k} value={k}>
              {ACTIVITY_KIND[k].label}
            </option>
          ))}
        </Select>
        {data && <span className="num ml-auto text-xs text-dim">{int(data.total)} entries</span>}
      </div>

      {actor > 0 && team && <ActorBanner name={team.items.find((m) => m.id === actor)?.name} onClear={() => reset(setActor)(0)} />}

      <Card className="p-0">
        {isPending ? (
          <Skeleton className="m-4 h-96" />
        ) : !data?.items.length ? (
          <EmptyState icon={History} title="No matching activity." />
        ) : (
          <div className={cn('transition-opacity', isFetching && 'opacity-70')}>
            {groupByDay(data.items).map(([day, items]) => (
              <section key={day}>
                <h2 className="eyebrow sticky top-[57px] z-10 border-b border-line bg-panel/95 px-4.5 py-2 backdrop-blur-sm">{day}</h2>
                <div className="flex flex-col px-4.5 py-1.5">
                  {items.map((a) => (
                    <ActivityItem key={a.id} a={a} showTime />
                  ))}
                </div>
              </section>
            ))}
            {data.items.length < data.total && (
              <div className="border-t border-line p-3 text-center">
                <Button size="sm" disabled={isFetching} onClick={() => setLimit((l) => l + PAGE)}>
                  Load {Math.min(PAGE, data.total - data.items.length)} more
                </Button>
              </div>
            )}
          </div>
        )}
      </Card>
    </>
  )
}

function ActorBanner({ name, onClear }: { name?: string; onClear: () => void }) {
  if (!name) return null
  return (
    <div className="mb-3.5 flex items-center gap-2.5 rounded-xl border border-line bg-panel px-3.5 py-2.5 text-[13px]">
      <Avatar name={name} size={24} />
      <span className="text-muted">
        Showing everything <b className="font-medium text-fg">{name}</b> did
      </span>
      <Button size="sm" variant="ghost" className="ml-auto" onClick={onClear}>
        Show everyone
      </Button>
    </div>
  )
}

function groupByDay(items: Activity[]) {
  const groups: [string, Activity[]][] = []
  for (const a of items) {
    const day = dayHeading(a.at)
    const last = groups[groups.length - 1]
    if (last?.[0] === day) last[1].push(a)
    else groups.push([day, [a]])
  }
  return groups
}
