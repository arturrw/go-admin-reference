import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/ui/dialog'
import { api } from '@/lib/api'
import { errorMessage } from '@/lib/queries'
import { DangerRow } from './rows'

type Action = 'log' | 'sessions'

const COPY: Record<Action, { title: string; description: string; confirm: string }> = {
  log: {
    title: 'Clear the request log?',
    description: 'Every captured API call, with its headers and bodies, is removed from memory. The activity log is not touched.',
    confirm: 'Clear log',
  },
  sessions: {
    title: 'Sign everyone else out?',
    description: 'Every other member must sign in again. Your own session stays. API keys keep working; revoke them separately.',
    confirm: 'Sign everyone out',
  },
}

/** Owner-only actions. Each one asks first and is written to the activity log. */
export function DangerZone({ enabled }: { enabled: boolean }) {
  const qc = useQueryClient()
  const [asking, setAsking] = useState<Action | null>(null)
  const run = useMutation({
    mutationFn: (a: Action): Promise<{ cleared: number } | { signedOut: number }> => (a === 'log' ? api.clearRequestLog() : api.signOutEveryone()),
    onSuccess: (r, a) => {
      setAsking(null)
      if ('cleared' in r) toast.success(`Request log cleared (${r.cleared} entries)`)
      else toast.success(`Signed out ${r.signedOut} ${r.signedOut === 1 ? 'session' : 'sessions'}`)
      return qc.invalidateQueries({ queryKey: a === 'log' ? ['requests'] : ['team'] })
    },
    onError: (e) => toast.error(errorMessage(e)),
  })
  const copy = asking ? COPY[asking] : null

  return (
    <>
      <DangerRow title="Clear request log" description="Empties the in-memory log behind the Request log page." action="Clear" disabled={!enabled} onClick={() => setAsking('log')} />
      <DangerRow title="Sign everyone out" description="Ends every other session; members must sign in again." action="Sign out" disabled={!enabled} onClick={() => setAsking('sessions')} />
      <DangerRow title="Delete workspace" description="Not available in this reference build: it would wipe all data." action="Delete" disabled onClick={() => {}} />
      <ConfirmDialog
        open={!!copy}
        onOpenChange={(v) => !v && setAsking(null)}
        title={copy?.title ?? ''}
        description={copy?.description}
        confirmLabel={copy?.confirm}
        pending={run.isPending}
        onConfirm={() => asking && run.mutate(asking)}
      />
    </>
  )
}
