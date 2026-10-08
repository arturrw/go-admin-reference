import { Link } from '@tanstack/react-router'
import { ArrowRight, Bell, BellOff, CheckCheck } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { ActivityItem } from '@/features/activity/activity-item'
import { useCan } from '@/lib/auth'
import { useNotifications, useReadNotifications } from '@/lib/queries'
import { cn } from '@/lib/utils'

/**
 * The bell: what others did (and security alerts about your account) since you
 * last looked. Opening it shows the unread ones highlighted; closing it marks
 * everything read.
 */
export function NotificationBell() {
  const { data } = useNotifications()
  const read = useReadNotifications()
  const canActivity = useCan('team:read')
  const [open, setOpen] = useState(false)
  const root = useRef<HTMLDivElement>(null)
  const unread = data?.unread ?? 0

  const close = () => {
    setOpen(false)
    if (unread > 0) read.mutate()
  }

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => !root.current?.contains(e.target as Node) && close()
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && close()
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
    // close() reads the latest unread count through the closure on each render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, unread])

  return (
    <div ref={root} className="relative">
      <Button variant="ghost" size="icon" aria-label={unread ? `Notifications, ${unread} unread` : 'Notifications'} aria-expanded={open} className="relative" onClick={() => (open ? close() : setOpen(true))}>
        <Bell />
        {unread > 0 && (
          <span data-testid="unread-count" className="num absolute top-1 right-1 grid min-w-4 place-items-center rounded-full border-2 border-bg bg-accent px-0.5 text-[9.5px] leading-3.5 font-semibold text-accent-ink">
            {unread > 9 ? '9+' : unread}
          </span>
        )}
      </Button>

      {open && (
        <div role="dialog" aria-label="Notifications" className="absolute top-full right-0 z-40 mt-2 flex max-h-[min(560px,75vh)] w-[min(400px,calc(100vw-24px))] animate-pop-in-right max-sm:fixed max-sm:inset-x-3 max-sm:top-14 max-sm:mt-0 max-sm:w-auto flex-col overflow-hidden rounded-2xl border border-line-2 bg-panel shadow-float">
          <div className="flex items-center gap-2 border-b border-line px-4 py-3">
            <b className="text-sm font-semibold">Notifications</b>
            {unread > 0 && <span className="num rounded-full bg-accent/14 px-1.5 text-[11px] text-accent">{unread} new</span>}
            <Button size="sm" variant="ghost" className="ml-auto" disabled={!unread || read.isPending} onClick={() => read.mutate()}>
              <CheckCheck />
              Mark all read
            </Button>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto px-2 py-1.5" onClick={close}>
            {data?.items.length === 0 && (
              <div className="flex flex-col items-center gap-2 px-6 py-10 text-center text-[13px] text-dim">
                <BellOff className="size-6" />
                Nothing new. Changes made by your team show up here.
              </div>
            )}
            {data?.items.map((n) => (
              <div key={n.id} data-testid="notification" data-unread={n.unread} className={cn('relative rounded-lg', n.unread && 'bg-accent/5')}>
                {n.unread && <i className="absolute top-4 left-0.5 size-1.5 rounded-full bg-accent" aria-label="unread" />}
                <ActivityItem a={n} timeline={false} />
              </div>
            ))}
          </div>
          {canActivity && (
            <Link to="/activity" onClick={close} className="flex items-center justify-center gap-1.5 border-t border-line px-4 py-2.5 text-[12.5px] text-muted hover:bg-panel-2 hover:text-fg">
              View the full activity log <ArrowRight className="size-3.5" />
            </Link>
          )}
        </div>
      )}
    </div>
  )
}
