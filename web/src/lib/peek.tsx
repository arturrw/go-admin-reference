import { useRouterState } from '@tanstack/react-router'
import { createContext, lazy, type ReactNode, Suspense, useCallback, useContext, useEffect, useMemo, useState } from 'react'

// "Peek" opens a customer, order, product or team member sheet on top of whatever page is
// showing, without navigating away. Sheets opened from inside a peek stack:
// closing one goes back to the previous.

export type PeekKind = 'customer' | 'order' | 'product' | 'member'
export interface PeekTarget {
  kind: PeekKind
  id: number
}

interface PeekApi {
  open: (kind: PeekKind, id: number) => void
  close: () => void
}

const PeekContext = createContext<PeekApi | null>(null)

const CustomerSheet = lazy(() => import('@/features/customers/customer-sheet').then((m) => ({ default: m.CustomerSheet })))
const OrderSheet = lazy(() => import('@/features/orders/order-sheet').then((m) => ({ default: m.OrderSheet })))
const ProductSheet = lazy(() => import('@/features/products/product-sheet').then((m) => ({ default: m.ProductSheet })))
const MemberDetailSheet = lazy(() => import('@/features/team/member-detail-sheet').then((m) => ({ default: m.MemberDetailSheet })))

export function PeekProvider({ children }: { children: ReactNode }) {
  const [stack, setStack] = useState<PeekTarget[]>([])
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  // A real navigation leaves the peeked sheets behind.
  useEffect(() => setStack([]), [pathname])

  const open = useCallback(
    (kind: PeekKind, id: number) =>
      setStack((s) => {
        const top = s[s.length - 1]
        return top?.kind === kind && top.id === id ? s : [...s.slice(-4), { kind, id }]
      }),
    [],
  )
  const close = useCallback(() => setStack((s) => s.slice(0, -1)), [])
  const api = useMemo(() => ({ open, close }), [open, close])
  const top = stack[stack.length - 1]

  return (
    <PeekContext.Provider value={api}>
      {children}
      <Suspense>
        {top?.kind === 'customer' && <CustomerSheet key={`c${top.id}`} customerId={top.id} onClose={close} />}
        {top?.kind === 'order' && <OrderSheet key={`o${top.id}`} orderId={top.id} onClose={close} />}
        {top?.kind === 'product' && <ProductSheet key={`p${top.id}`} productId={top.id} open onClose={close} onCreated={close} />}
        {top?.kind === 'member' && <MemberDetailSheet key={`m${top.id}`} memberId={top.id} onClose={close} />}
      </Suspense>
    </PeekContext.Provider>
  )
}

export function usePeek() {
  const ctx = useContext(PeekContext)
  if (!ctx) throw new Error('usePeek used outside PeekProvider')
  return ctx.open
}

/** Inline trigger that peeks an entity; renders plain content when `disabled`. */
export function PeekButton({
  kind,
  id,
  disabled,
  className,
  label,
  children,
  ...rest
}: {
  kind: PeekKind
  id: number
  disabled?: boolean
  className?: string
  label?: string
  children: ReactNode
  'data-testid'?: string
}) {
  const open = usePeek()
  if (disabled)
    return (
      <span className={className} {...rest}>
        {children}
      </span>
    )
  return (
    <button
      type="button"
      aria-label={label}
      className={className}
      {...rest}
      onClick={(e) => {
        e.stopPropagation()
        open(kind, id)
      }}
    >
      {children}
    </button>
  )
}
