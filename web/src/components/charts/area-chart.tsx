import { type MouseEvent, useId, useRef, useState } from 'react'
import { Delta } from '@/components/ui/misc'
import { smoothPath, svgId } from '@/lib/chart'
import { compact, money, shortDate } from '@/lib/format'
import { cn } from '@/lib/utils'

export interface SeriesPoint {
  date: string
  current: number
  previous?: number
}

const W = 1000
const H = 228

const dayLabel = (iso: string) => new Date(iso).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })

/**
 * Area chart: glowing current series over a dashed previous period (when the
 * points have one). Defaults format values as cents, for revenue.
 *
 * `gutter` reserves room on the left for the y-axis labels; without it they
 * sit on top of the plot. Hover state is computed against the plot box only,
 * so the crosshair, dot and tooltip always line up with the curve.
 */
export function AreaChart({
  points,
  label = 'Revenue',
  color = 'var(--color-accent)',
  format = (v: number) => money(v),
  axisFormat = (v: number) => compact(v / 100),
  xFormat = shortDate,
  tipDate = dayLabel,
  gutter = 0,
  onPick,
  className,
}: {
  points: SeriesPoint[]
  label?: string
  color?: string
  format?: (v: number) => string
  axisFormat?: (v: number) => string
  xFormat?: (iso: string) => string
  tipDate?: (iso: string) => string
  gutter?: number
  /** Click on a point (e.g. to open that day). */
  onPick?: (index: number) => void
  className?: string
}) {
  const gid = svgId(useId())
  const plotRef = useRef<HTMLDivElement>(null)
  const [hover, setHover] = useState<number | null>(null)
  if (points.length < 2) return <div className="h-62" />

  const n = points.length
  const hasPrev = points.every((p) => p.previous !== undefined)
  const max = Math.max(...points.flatMap((p) => [p.current, p.previous ?? 0])) * 1.1 || 1
  const x = (i: number) => (i / (n - 1)) * W
  const y = (v: number) => H - (v / max) * H
  const line = smoothPath(points.map((p, i) => [x(i), y(p.current)] as const))
  const prevLine = hasPrev ? smoothPath(points.map((p, i) => [x(i), y(p.previous!)] as const)) : ''
  const ticks = [0, 0.25, 0.5, 0.75, 1]

  // Each point owns the band halfway to its neighbours, so the tooltip only
  // changes once the cursor is closer to the next day than to the current one.
  const onMove = (e: MouseEvent) => {
    const rect = plotRef.current!.getBoundingClientRect()
    const f = (e.clientX - rect.left) / rect.width
    if (f < 0 || f > 1) return setHover(null)
    setHover(Math.min(n - 1, Math.max(0, Math.round(f * (n - 1)))))
  }

  const hp = hover === null ? null : points[hover]
  const leftPct = hover === null ? 0 : (hover / (n - 1)) * 100
  const topPx = hp ? y(hp.current) : 0

  return (
    <div className={cn('mt-3.5', className)} style={{ paddingLeft: gutter }}>
      <div
        ref={plotRef}
        className={cn('relative', onPick && hp && 'cursor-pointer')}
        onMouseMove={onMove}
        onMouseLeave={() => setHover(null)}
        onClick={() => hover !== null && onPick?.(hover)}
      >
        <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" className="block w-full overflow-visible" style={{ height: H }} role="img" aria-label={`${label} over time`}>
          <defs>
            <linearGradient id={`${gid}f`} x1="0" x2="0" y1="0" y2="1">
              <stop offset="0" stopColor={color} stopOpacity=".32" />
              <stop offset=".7" stopColor={color} stopOpacity=".04" />
              <stop offset="1" stopColor={color} stopOpacity="0" />
            </linearGradient>
            <filter id={`${gid}g`} x="-10%" y="-50%" width="120%" height="200%">
              <feGaussianBlur stdDeviation="6" />
            </filter>
          </defs>
          {ticks.map((t) => (
            <line key={t} x1="0" x2={W} y1={H * t} y2={H * t} stroke="rgb(255 255 255 / .05)" strokeDasharray="3 5" vectorEffect="non-scaling-stroke" />
          ))}
          {hasPrev && <path d={prevLine} fill="none" stroke="var(--color-dim)" strokeWidth="1.5" strokeDasharray="4 5" vectorEffect="non-scaling-stroke" />}
          <path d={`${line} L${W},${H} L0,${H}Z`} fill={`url(#${gid}f)`} />
          <path d={line} fill="none" stroke={color} strokeWidth="6" opacity=".35" filter={`url(#${gid}g)`} vectorEffect="non-scaling-stroke" />
          <path d={line} fill="none" stroke={color} strokeWidth="2" vectorEffect="non-scaling-stroke" />
        </svg>

        {ticks.slice(0, 4).map((t) => (
          <span
            key={t}
            className={cn('num pointer-events-none absolute -translate-y-1/2 text-[10.5px] text-dim', gutter ? 'right-full pr-2' : 'left-0')}
            style={{ top: t * H }}
          >
            {axisFormat(max * (1 - t))}
          </span>
        ))}

        {hp && (
          <>
            <div className="pointer-events-none absolute top-0 w-px bg-linear-to-b from-line-2 to-transparent" style={{ left: `${leftPct}%`, height: H }} />
            {hasPrev && (
              <div
                className="pointer-events-none absolute size-[7px] -translate-1/2 rounded-full border border-dim bg-bg"
                style={{ left: `${leftPct}%`, top: y(hp.previous!) }}
              />
            )}
            <div
              data-testid="chart-dot"
              className="pointer-events-none absolute size-[11px] -translate-1/2 rounded-full border-3 border-bg"
              style={{ left: `${leftPct}%`, top: topPx, background: color, boxShadow: `0 0 0 4px color-mix(in srgb, ${color} 25%, transparent)` }}
            />
            <div
              role="tooltip"
              className="pointer-events-none absolute z-10 min-w-38 rounded-[10px] border border-line-2 bg-panel-2/92 px-2.5 py-2 text-xs shadow-[0_12px_30px_-10px_rgb(0_0_0/.6)] backdrop-blur-sm"
              style={{
                left: `${leftPct}%`,
                top: Math.min(Math.max(0, topPx - 40), H - 90),
                transform: leftPct > 70 ? 'translateX(calc(-100% - 14px))' : 'translateX(14px)',
              }}
            >
              <div className="num mb-1 text-[11px] text-dim">{tipDate(hp.date)}</div>
              <div className="flex justify-between gap-3">
                <span className="text-muted">{label}</span>
                <b className="num font-medium">{format(hp.current)}</b>
              </div>
              {hp.previous !== undefined && (
                <>
                  <div className="flex justify-between gap-3">
                    <span className="text-muted">Previous</span>
                    <b className="num font-medium text-muted">{format(hp.previous)}</b>
                  </div>
                  {hp.previous > 0 && (
                    <div className="mt-1 flex justify-end">
                      <Delta value={(hp.current / hp.previous - 1) * 100} />
                    </div>
                  )}
                </>
              )}
              {onPick && <div className="mt-1.5 border-t border-line pt-1.5 text-[10.5px] text-dim">Click for details</div>}
            </div>
          </>
        )}
      </div>

      <div className="num mt-1.5 flex justify-between text-[10.5px] text-dim">
        {[0, 0.2, 0.4, 0.6, 0.8, 1].map((f) => (
          <span key={f}>{xFormat(points[Math.round(f * (n - 1))].date)}</span>
        ))}
      </div>
    </div>
  )
}
