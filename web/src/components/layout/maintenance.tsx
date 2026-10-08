import { Link } from '@tanstack/react-router'
import { LogOut, Wrench } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useCan, useLogout } from '@/lib/auth'

/** Shown to owners and admins while maintenance mode is on. */
export function MaintenanceBanner() {
  const canSettings = useCan('settings:write')
  return (
    <div role="status" className="mb-4 flex items-center gap-2.5 rounded-xl border border-warn/30 bg-warn/8 px-3.5 py-2.5 text-[13px] text-warn" data-testid="maintenance-banner">
      <Wrench className="size-4 shrink-0" />
      <span className="min-w-0 flex-1">
        Maintenance mode is on. Everyone except owners and admins is locked out, and API keys get 503.
      </span>
      {canSettings && (
        <Link to="/settings" className="shrink-0 font-medium underline underline-offset-2">
          Turn off
        </Link>
      )}
    </div>
  )
}

/** Replaces the app for everyone else while maintenance mode is on. */
export function MaintenancePage({ serviceName }: { serviceName?: string }) {
  const logout = useLogout()
  return (
    <div className="grid min-h-screen place-items-center px-6">
      <div className="flex max-w-md flex-col items-center gap-3 text-center" data-testid="maintenance-page">
        <span className="grid size-12 place-items-center rounded-2xl bg-warn/12 text-warn">
          <Wrench className="size-6" />
        </span>
        <h1 className="text-xl font-semibold">Down for maintenance</h1>
        <p className="text-muted">
          {serviceName ? `${serviceName} is` : 'The workspace is'} being updated. Please check back in a few minutes; this page refreshes by itself.
        </p>
        <Button onClick={() => logout.mutate()}>
          <LogOut />
          Sign out
        </Button>
      </div>
    </div>
  )
}
