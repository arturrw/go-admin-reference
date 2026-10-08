import { Outlet, useRouterState } from '@tanstack/react-router'
import { ChevronRight, CircleHelp, PanelLeft, Search } from 'lucide-react'
import { lazy, Suspense, useEffect, useState } from 'react'
import { Toaster } from 'sonner'
import { Button } from '@/components/ui/button'
import { useMe } from '@/lib/auth'
import { NAV_ITEMS } from '@/lib/nav'
import { PeekProvider } from '@/lib/peek'
import { useMeta } from '@/lib/queries'
import { MaintenanceBanner, MaintenancePage } from './maintenance'
import { NotificationBell } from './notification-bell'
import { Sidebar } from './sidebar'

// cmdk is only needed once the menu is opened, so it is its own chunk.
const CommandMenu = lazy(() => import('./command-menu').then((m) => ({ default: m.CommandMenu })))
const HelpSheet = lazy(() => import('./help-sheet').then((m) => ({ default: m.HelpSheet })))

/** True while the user is typing, so single-key shortcuts must stay out of the way. */
const typing = (t: EventTarget | null) => t instanceof HTMLElement && (t.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName))

export function AppShell() {
  const { data: meta } = useMeta()
  const me = useMe()
  const [sideOpen, setSideOpen] = useState(false)
  const [cmdOpen, setCmdOpen] = useState(false)
  const [cmdUsed, setCmdUsed] = useState(false)
  const [help, setHelp] = useState(false)
  const openCmd = (v: boolean) => {
    if (v) setCmdUsed(true)
    setCmdOpen(v)
  }

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setCmdUsed(true)
        setCmdOpen((o) => !o)
      } else if (e.key === '?' && !e.ctrlKey && !e.metaKey && !e.altKey && !typing(e.target)) {
        e.preventDefault()
        setHelp(true)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const current = NAV_ITEMS.find((i) => (i.to === '/' ? pathname === '/' : pathname.startsWith(i.to)))

  // Owners and admins keep working (with a banner); everyone else sees a holding page.
  const locked = !!meta?.maintenance && me.user.role !== 'owner' && me.user.role !== 'admin'
  if (locked) return <MaintenancePage serviceName={meta?.serviceName} />

  return (
    <PeekProvider>
      <div className="grid min-h-screen md:grid-cols-[252px_minmax(0,1fr)]">
        <Sidebar open={sideOpen} onNavigate={() => setSideOpen(false)} />
        {sideOpen && <div className="fixed inset-0 z-30 bg-black/40 md:hidden" onClick={() => setSideOpen(false)} />}

        <div className="relative flex min-w-0 flex-col">
          {/* Soft accent glow behind the page header. */}
          <div
            aria-hidden
            className="pointer-events-none absolute inset-x-0 top-0 h-105"
            style={{
              background:
                'radial-gradient(700px 260px at 18% -40px, color-mix(in srgb, var(--color-accent) 8%, transparent), transparent 70%), radial-gradient(500px 220px at 85% -60px, color-mix(in srgb, var(--color-info) 7%, transparent), transparent 70%)',
            }}
          />
          <header className="sticky top-0 z-30 flex items-center gap-3 border-b border-line bg-bg/78 px-4 py-2.5 backdrop-blur-md md:px-7 md:py-3">
            <Button variant="ghost" size="icon" className="md:hidden" onClick={() => setSideOpen(true)} aria-label="Open menu">
              <PanelLeft />
            </Button>
            <div className="flex items-center gap-2 text-[13px] whitespace-nowrap text-dim">
              <span className="max-sm:hidden">{meta?.serviceName ?? ''}</span>
              <ChevronRight className="size-3.5 max-sm:hidden" />
              <b className="font-medium text-fg">{current?.label ?? 'Not found'}</b>
            </div>
            <button
              onClick={() => openCmd(true)}
              className="ml-auto flex h-[34px] items-center gap-2.5 rounded-[10px] border border-line-2 bg-panel px-3 text-[13px] text-dim transition-colors hover:border-accent/40 hover:text-muted max-md:size-[34px] max-md:justify-center max-md:px-0 md:min-w-65"
            >
              <Search className="size-4" />
              <span className="max-md:hidden">Search or jump to…</span>
              <kbd className="num ml-auto rounded-md border border-line-2 bg-panel-2 px-1.5 text-[11px] text-muted max-md:hidden">Ctrl K</kbd>
            </button>
            <NotificationBell />
            <Button variant="ghost" size="icon" aria-label="Help" onClick={() => setHelp(true)}>
              <CircleHelp />
            </Button>
          </header>

          <main className="relative w-full max-w-[1560px] px-4 pt-5 pb-12 md:px-7 md:pt-6.5">
            {meta?.maintenance && <MaintenanceBanner />}
            <Outlet />
          </main>
        </div>

        {help && (
          <Suspense>
            <HelpSheet onClose={() => setHelp(false)} />
          </Suspense>
        )}
        {cmdUsed && (
          <Suspense>
            <CommandMenu open={cmdOpen} onOpenChange={openCmd} />
          </Suspense>
        )}
        <Toaster
          theme="dark"
          position="bottom-right"
          toastOptions={{ className: 'bg-panel-2! border-line-2! text-fg! rounded-xl! font-sans!' }}
        />
      </div>
    </PeekProvider>
  )
}
