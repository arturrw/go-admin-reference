import { useId } from 'react'
import { smoothPath, svgId } from '@/lib/chart'
import { cn } from '@/lib/utils'

export function Sparkline({ data, color = 'var(--color-accent)', className }: { data: number[]; color?: string; className?: string }) {
  const id = svgId(useId())
  if (data.length < 2) return <svg className={cn('h-7 w-21', className)} />
  const max = Math.max(...data)
  const min = Math.min(...data)
  const pts = data.map((v, i) => [(i / (data.length - 1)) * 100, 27 - ((v - min) / (max - min || 1)) * 24] as const)
  const line = smoothPath(pts)
  return (
    <svg className={cn('h-7 w-21 overflow-visible', className)} viewBox="0 0 100 30" preserveAspectRatio="none" aria-hidden>
      <defs>
        <linearGradient id={id} x1="0" x2="0" y1="0" y2="1">
          <stop offset="0" stopColor={color} stopOpacity=".35" />
          <stop offset="1" stopColor={color} stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={`${line} L100,30 L0,30Z`} fill={`url(#${id})`} />
      <path d={line} fill="none" stroke={color} strokeWidth="1.6" vectorEffect="non-scaling-stroke" />
    </svg>
  )
}
