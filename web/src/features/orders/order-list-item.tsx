import { ChevronRight, Undo2 } from 'lucide-react'
import { Avatar } from '@/components/ui/misc'
import { StatusPill } from '@/components/ui/pill'
import type { Order } from '@/lib/api'
import { money, timeAgo } from '@/lib/format'

/** One order as a compact row, for phones where the orders table doesn't fit. */
export function OrderListItem({ o, onOpen }: { o: Order; onOpen?: () => void }) {
  const units = o.items.reduce((s, i) => s + i.qty, 0)
  return (
    <button
      type="button"
      onClick={onOpen}
      disabled={!onOpen}
      aria-label={`Open order #${o.id}`}
      data-testid="order-row"
      className="flex w-full items-center gap-3 border-b border-line px-3.5 py-3 text-left last:border-0 enabled:active:bg-panel-2"
    >
      <Avatar name={o.customer.name} size={34} />
      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <b className="truncate text-[13.5px] font-medium">{o.customer.name}</b>
          <span className="num ml-auto shrink-0 text-[13px] font-medium">{money(o.totalCents, 2)}</span>
        </div>
        <div className="mt-1 flex items-center gap-2 text-[11.5px] text-dim">
          <span className="num">#{o.id}</span>·<span className="num">{timeAgo(o.placedAt)}</span>·<span>{units} items</span>
          <span className="ml-auto shrink-0">
            <StatusPill status={o.status} />
          </span>
        </div>
        {o.refund && (
          <div className="mt-1 flex items-center gap-1 text-[11.5px] text-violet">
            <Undo2 className="size-3 shrink-0" />
            <span className="truncate">{o.refund.reason}</span>
          </div>
        )}
      </div>
      {onOpen && <ChevronRight className="size-4 shrink-0 text-dim" />}
    </button>
  )
}
