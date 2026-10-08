import {
  ChevronRight,
  FileUp,
  ImagePlus,
  LogIn,
  type LucideIcon,
  MessageSquareText,
  Package,
  PackagePlus,
  Rocket,
  Settings2,
  ShieldAlert,
  Target,
  TriangleAlert,
  Truck,
  Undo2,
  UserCheck,
  UsersRound,
} from 'lucide-react'
import { IconTile } from '@/components/ui/misc'
import { type Tone, toneColor } from '@/components/ui/pill'
import type { Activity, ActivityKind, Permission } from '@/lib/api'
import { useCan } from '@/lib/auth'
import { timeAgo } from '@/lib/format'
import { type PeekKind, usePeek } from '@/lib/peek'
import { cn } from '@/lib/utils'

export const ACTIVITY_KIND: Record<ActivityKind, { icon: LucideIcon; tone: Tone; label: string }> = {
  product: { icon: Package, tone: 'blue', label: 'Products' },
  publish: { icon: PackagePlus, tone: 'lime', label: 'Publishing' },
  image: { icon: ImagePlus, tone: 'blue', label: 'Images' },
  import: { icon: FileUp, tone: 'teal', label: 'Imports' },
  order: { icon: Truck, tone: 'teal', label: 'Orders' },
  refund: { icon: Undo2, tone: 'red', label: 'Refunds' },
  note: { icon: MessageSquareText, tone: 'amber', label: 'Notes' },
  team: { icon: UsersRound, tone: 'violet', label: 'Team' },
  role: { icon: UserCheck, tone: 'lime', label: 'Roles & access' },
  target: { icon: Target, tone: 'lime', label: 'Targets' },
  settings: { icon: Settings2, tone: 'gray', label: 'Settings' },
  auth: { icon: LogIn, tone: 'gray', label: 'Sign-ins' },
  alert: { icon: ShieldAlert, tone: 'amber', label: 'Security alerts' },
  deploy: { icon: Rocket, tone: 'violet', label: 'Deploys' },
  stock: { icon: TriangleAlert, tone: 'amber', label: 'Inventory' },
}

const OPENS: Record<string, { kind: PeekKind; perm: Permission }> = {
  product: { kind: 'product', perm: 'products:read' },
  order: { kind: 'order', perm: 'orders:read' },
  customer: { kind: 'customer', perm: 'customers:read' },
  member: { kind: 'member', perm: 'team:read' },
}

/** One audit entry. Clicking it opens the record it touched, in place. */
export function ActivityItem({ a, timeline = true, showTime }: { a: Activity; timeline?: boolean; showTime?: boolean }) {
  const { icon, tone } = ACTIVITY_KIND[a.kind] ?? ACTIVITY_KIND.product
  const peek = usePeek()
  const target = OPENS[a.entity]
  const allowed = useCan(target?.perm ?? 'dashboard:read')
  const open = target && allowed && a.entityId ? () => peek(target.kind, a.entityId) : undefined
  const at = new Date(a.at)

  return (
    <div
      role={open ? 'button' : undefined}
      tabIndex={open ? 0 : undefined}
      onClick={open}
      onKeyDown={open && ((e) => (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), open()))}
      data-testid="activity-item"
      className={cn(
        'group relative -mx-2 flex gap-3 rounded-lg px-2 py-2',
        timeline && 'not-last:after:absolute not-last:after:top-10 not-last:after:-bottom-1 not-last:after:left-[22px] not-last:after:w-px not-last:after:bg-line-2',
        open && 'cursor-pointer transition-colors hover:bg-panel-2',
      )}
    >
      <IconTile icon={icon} color={toneColor(tone)} size={27} />
      <div className="min-w-0 flex-1">
        <p className="text-[13px] text-muted">
          <b className="font-medium text-fg">{a.actor}</b> {a.message}
        </p>
        <small className="num text-[11px] text-dim" title={at.toLocaleString()}>
          {showTime ? at.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' }) : timeAgo(a.at)}
        </small>
      </div>
      {open && <ChevronRight className="mt-1.5 size-3.5 shrink-0 text-dim opacity-0 transition-opacity group-hover:opacity-100" />}
    </div>
  )
}
