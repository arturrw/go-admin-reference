const DAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
const GRID = 'grid grid-cols-[28px_repeat(24,minmax(0,1fr))] gap-[3px] max-sm:grid-cols-[24px_repeat(24,minmax(0,1fr))] max-sm:gap-0.5'

const cellColor = (intensity: number) =>
  `color-mix(in srgb, var(--color-accent) ${Math.round(6 + intensity * 94)}%, var(--color-panel-2))`

/** 7×24 activity grid (rows Mon..Sun, columns hours). */
export function Heatmap({ data }: { data: number[][] }) {
  const max = Math.max(1, ...data.flat())
  return (
    <div>
      <div className={`${GRID} items-center`}>
        {data.map((row, d) => [
          <span key={`l${d}`} className="num text-[10px] text-dim">
            {DAYS[d]}
          </span>,
          ...row.map((v, h) => (
            <div
              key={`${d}-${h}`}
              title={`${DAYS[d]} ${String(h).padStart(2, '0')}:00 — ${v} orders`}
              className="aspect-square rounded-[3px] transition-transform hover:relative hover:z-10 hover:scale-135 hover:outline hover:outline-accent"
              style={{ background: cellColor(v / max) }}
            />
          )),
        ])}
      </div>
      <div className={`${GRID} mt-1.5`}>
        <span />
        {Array.from({ length: 24 }, (_, h) => (
          <span key={h} className="num text-center text-[9.5px] text-dim">
            {h % 6 === 0 ? String(h).padStart(2, '0') : ''}
          </span>
        ))}
      </div>
    </div>
  )
}

export function HeatmapScale() {
  return (
    <div className="num flex items-center gap-1 text-[10.5px] text-dim">
      less
      {[0.02, 0.25, 0.5, 0.75, 1].map((a) => (
        <i key={a} className="size-3 rounded-[3px]" style={{ background: cellColor(a) }} />
      ))}
      more
    </div>
  )
}
