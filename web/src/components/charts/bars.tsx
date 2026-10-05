import { type ReactNode, useState } from 'react'
import { cn } from '@/lib/utils'

/**
 * Column chart that reacts to the pointer: the hovered bar brightens and
 * grows a glow, the others dim, and a tooltip shows its value. Each bar's hit
 * area is the full column height, so thin or short bars are easy to hover.
 */
export function Bars({
  heights,
  tip,
  tone,
  label,
  className,
  barClassName,
}: {
  /** Bar heights in percent of the chart (0–100). */
  heights: number[]
  /** Tooltip content for bar i. */
  tip: (i: number) => ReactNode
  /** Per-bar colour class; defaults to the accent gradient. */
  tone?: (i: number) => string
  label: string
  className?: string
  barClassName?: string
}) {
  const [hover, setHover] = useState<number | null>(null)
  const n = heights.length
  const leftPct = hover === null ? 0 : ((hover + 0.5) / n) * 100

  return (
    <div className={cn('relative flex items-end gap-[3px]', className)} role="img" aria-label={label} onMouseLeave={() => setHover(null)}>
      {heights.map((h, i) => (
        <div
          key={i}
          data-testid="bar"
          onMouseEnter={() => setHover(i)}
          // Bars sit inside clickable cards: hovering must not swallow the click.
          className="flex h-full flex-1 items-end"
        >
          <i
            className={cn(
              'block w-full min-h-[3px] rounded-t-[3px] rounded-b-[1px] transition-[height,opacity,filter,box-shadow] duration-300 ease-[cubic-bezier(.2,.8,.2,1)]',
              tone?.(i) ?? 'bg-linear-to-b from-accent to-accent/25',
              hover !== null && hover !== i && 'opacity-35',
              hover === i && 'brightness-125',
              barClassName,
            )}
            style={{
              height: `${h}%`,
              boxShadow: hover === i ? '0 0 14px -2px color-mix(in srgb, var(--color-accent) 55%, transparent)' : undefined,
            }}
          />
        </div>
      ))}
      {hover !== null && (
        <>
          <div className="pointer-events-none absolute inset-y-0 w-px bg-linear-to-b from-transparent via-line-2 to-transparent" style={{ left: `${leftPct}%` }} />
          <div
            role="tooltip"
            className="num pointer-events-none absolute bottom-full z-20 mb-1.5 rounded-lg border border-line-2 bg-panel-2/95 px-2.5 py-1.5 text-[11.5px] whitespace-nowrap shadow-float backdrop-blur-sm"
            style={{
              left: `${leftPct}%`,
              transform: leftPct < 18 ? 'translateX(-12px)' : leftPct > 82 ? 'translateX(calc(-100% + 12px))' : 'translateX(-50%)',
            }}
          >
            {tip(hover)}
          </div>
        </>
      )}
    </div>
  )
}
