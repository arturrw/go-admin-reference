import type { CSSProperties, ReactNode } from 'react'
import { cn } from '@/lib/utils'

const TONES = {
  lime: 'var(--color-accent)',
  amber: 'var(--color-warn)',
  red: 'var(--color-danger)',
  blue: 'var(--color-info)',
  violet: 'var(--color-violet)',
  teal: 'var(--color-teal)',
  gray: '#9aa1ab',
} as const

export type Tone = keyof typeof TONES

export const toneColor = (t: Tone) => TONES[t]

export const STATUS_TONE: Record<string, Tone> = {
  active: 'lime',
  draft: 'gray',
  archived: 'violet',
  pending: 'amber',
  paid: 'lime',
  shipped: 'blue',
  delivered: 'teal',
  refunded: 'violet',
  failed: 'red',
  cancelled: 'gray',
  invited: 'blue',
  suspended: 'red',
  VIP: 'lime',
  Regular: 'gray',
  New: 'blue',
  'At risk': 'amber',
  owner: 'lime',
  admin: 'violet',
  editor: 'blue',
  support: 'teal',
  viewer: 'gray',
}

export function Pill({ tone = 'gray', dot = true, className, children }: { tone?: Tone; dot?: boolean; className?: string; children: ReactNode }) {
  return (
    <span className={cn('pill', className)} data-dot={dot || undefined} style={{ '--c': TONES[tone] } as CSSProperties}>
      {children}
    </span>
  )
}

export function StatusPill({ status, dot }: { status: string; dot?: boolean }) {
  return (
    <Pill tone={STATUS_TONE[status] ?? 'gray'} dot={dot}>
      {status}
    </Pill>
  )
}
