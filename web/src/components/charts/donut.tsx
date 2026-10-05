import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export const SERIES_COLORS = [
  'var(--color-accent)',
  'var(--color-info)',
  'var(--color-violet)',
  'var(--color-teal)',
  'var(--color-warn)',
  'var(--color-dim)',
]

/**
 * Ring chart. With `onActive` the segments are hoverable: the hovered one
 * thickens and the rest fade, and the parent decides what the centre shows.
 */
export function Donut({
  values,
  children,
  size = 132,
  active = null,
  onActive,
  labels,
}: {
  values: number[]
  children?: ReactNode
  size?: number
  active?: number | null
  onActive?: (i: number | null) => void
  /** Accessible names for the segments. */
  labels?: string[]
}) {
  const total = values.reduce((a, b) => a + b, 0) || 1
  const r = 48
  const c = 2 * Math.PI * r
  let offset = 0
  return (
    <div className="relative shrink-0" style={{ width: size, height: size }} onMouseLeave={() => onActive?.(null)}>
      <svg viewBox="0 0 120 120" className="size-full -rotate-90 overflow-visible" aria-hidden={!onActive}>
        <circle cx="60" cy="60" r={r} fill="none" stroke="var(--color-panel-2)" strokeWidth="14" />
        {values.map((v, i) => {
          const len = (v / total) * c
          const on = active === i
          const el = (
            <circle
              key={i}
              cx="60"
              cy="60"
              r={r}
              fill="none"
              stroke={SERIES_COLORS[i % SERIES_COLORS.length]}
              strokeWidth={on ? 19 : 14}
              strokeDasharray={`${Math.max(0, len - 2.5)} ${c}`}
              strokeDashoffset={-offset}
              opacity={active === null || on ? 1 : 0.32}
              className={cn('transition-[stroke-width,opacity] duration-200', onActive && 'cursor-pointer')}
              onMouseEnter={onActive && (() => onActive(i))}
              aria-label={labels?.[i]}
            />
          )
          offset += len
          return el
        })}
      </svg>
      <div className="pointer-events-none absolute inset-0 grid place-content-center text-center">{children}</div>
    </div>
  )
}

export function ProgressRing({ value, label }: { value: number; label: ReactNode }) {
  const r = 50
  const c = 2 * Math.PI * r
  return (
    <div className="relative size-32 shrink-0">
      <svg viewBox="0 0 120 120" className="size-full -rotate-90" aria-hidden>
        <circle cx="60" cy="60" r={r} fill="none" stroke="var(--color-panel-3)" strokeWidth="12" />
        <circle
          cx="60"
          cy="60"
          r={r}
          fill="none"
          stroke="var(--color-accent)"
          strokeWidth="12"
          strokeLinecap="round"
          strokeDasharray={`${c * Math.min(1, value)} ${c}`}
          style={{ filter: 'drop-shadow(0 0 6px color-mix(in srgb, var(--color-accent) 60%, transparent))' }}
        />
      </svg>
      <div className="num absolute inset-0 grid place-items-center text-xl font-semibold">{label}</div>
    </div>
  )
}
