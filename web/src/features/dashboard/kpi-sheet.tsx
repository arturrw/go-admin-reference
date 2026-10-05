import { useState } from 'react'
import { AreaChart } from '@/components/charts/area-chart'
import { Delta, IconTile } from '@/components/ui/misc'
import { Sheet } from '@/components/ui/sheet'
import { Tabs } from '@/components/ui/tabs'
import type { KPI } from '@/lib/api'
import { shortDate } from '@/lib/format'
import { cn } from '@/lib/utils'
import { formatKpi, formatUnit, isAdditive, kpiStyle } from './kpi'

/** Full-size view of the KPI cards: daily chart, period stats and a per-day table. */
export function KpiSheet({ kpis, initial, rangeDays, onClose }: { kpis: KPI[]; initial: string; rangeDays: number; onClose: () => void }) {
  const [key, setKey] = useState(initial)
  const k = kpis.find((x) => x.key === key) ?? kpis[0]
  const { icon: Icon, color } = kpiStyle(k.key)
  const tint = k.deltaPct < 0 ? 'var(--color-danger)' : color
  const fmt = (v: number) => formatUnit(k.unit, v)

  const cur = k.series.map((p) => p.current)
  const prev = k.series.map((p) => p.previous)
  const agg = (xs: number[]) => (isAdditive(k) ? xs.reduce((a, b) => a + b, 0) : xs.reduce((a, b) => a + b, 0) / Math.max(xs.length, 1))
  const peak = k.series.reduce((a, b) => (b.current > a.current ? b : a), k.series[0])
  const low = k.series.reduce((a, b) => (b.current < a.current ? b : a), k.series[0])
  const curAgg = agg(cur)
  const prevAgg = agg(prev)
  const maxDay = Math.max(...cur, 1)

  return (
    <Sheet open onOpenChange={(v) => !v && onClose()} size="lg" title={k.label} description={`Daily breakdown · last ${rangeDays} days vs the ${rangeDays} days before`}>
      <Tabs value={k.key} onChange={setKey} tabs={kpis.map((x) => ({ value: x.key, label: x.label }))} />

      <div className="flex flex-wrap items-end gap-3.5">
        <IconTile icon={Icon} color={color} size={40} className="rounded-xl" />
        <div>
          <div className="eyebrow">{isAdditive(k) ? 'Total' : 'Current'}</div>
          <div className="num text-[32px] leading-none font-semibold tracking-[-0.04em]">{formatKpi(k)}</div>
        </div>
        <div className="flex items-center gap-2 pb-1">
          <Delta value={k.deltaPct} />
          <span className="text-[12.5px] text-dim">vs previous period</span>
        </div>
      </div>

      {k.series.length > 1 && (
        <div className="rounded-xl border border-line bg-panel-2/40 p-3.5 pb-2.5">
          <div className="mb-1 flex items-center gap-3.5 text-xs text-muted">
            <span className="flex items-center gap-1.5">
              <i className="h-[3px] w-2.5 rounded-xs" style={{ background: tint }} />
              This period
            </span>
            <span className="flex items-center gap-1.5">
              <i className="h-[3px] w-2.5 rounded-xs bg-dim" />
              Previous
            </span>
          </div>
          <AreaChart points={k.series} label={k.label} color={tint} format={fmt} axisFormat={(v) => formatUnit(k.unit, v, true)} gutter={28} />
        </div>
      )}

      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
        <Stat label={isAdditive(k) ? 'This period' : 'Period average'} value={fmt(curAgg)} />
        <Stat label={isAdditive(k) ? 'Previous period' : 'Previous average'} value={fmt(prevAgg)} muted />
        <Stat label="Best day" value={fmt(peak.current)} sub={shortDate(peak.date)} />
        <Stat label="Weakest day" value={fmt(low.current)} sub={shortDate(low.date)} />
      </div>

      <div className="overflow-hidden rounded-xl border border-line">
        <table className="data-table">
          <thead>
            <tr>
              <th>Day</th>
              <th className="w-[40%]" />
              <th className="num">{k.label}</th>
              <th className="num">Previous</th>
              <th className="num">Change</th>
            </tr>
          </thead>
          <tbody>
            {[...k.series].reverse().map((p) => {
              const ch = p.previous ? (p.current / p.previous - 1) * 100 : 0
              return (
                <tr key={p.date}>
                  <td className="font-mono text-muted tabular-nums">{new Date(p.date).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })}</td>
                  <td>
                    <div className="h-1.5 overflow-hidden rounded bg-panel-3">
                      <i className="block h-full rounded" style={{ width: `${(p.current / maxDay) * 100}%`, background: tint }} />
                    </div>
                  </td>
                  <td className="num">{fmt(p.current)}</td>
                  <td className="num text-muted">{fmt(p.previous)}</td>
                  <td className={cn('num', ch >= 0 ? 'text-accent' : 'text-danger')}>
                    {ch >= 0 ? '+' : ''}
                    {ch.toFixed(1)}%
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </Sheet>
  )
}

function Stat({ label, value, sub, muted }: { label: string; value: string; sub?: string; muted?: boolean }) {
  return (
    <div className="rounded-xl border border-line bg-panel-2/40 px-3 py-2.5">
      <small className="block text-[11.5px] text-dim">{label}</small>
      <b className={cn('num text-[15px] font-medium', muted && 'text-muted')}>{value}</b>
      {sub && <small className="num block text-[11px] text-dim">{sub}</small>}
    </div>
  )
}
