// Typed client for the Go API (/api/v1). Field names mirror the json tags in
// internal/domain. Money is always integer cents.

export type Category = 'Audio' | 'Wearables' | 'Lighting' | 'Home' | 'Computing' | 'Accessories'
export const CATEGORIES: Category[] = ['Audio', 'Wearables', 'Lighting', 'Home', 'Computing', 'Accessories']

export type ProductStatus = 'active' | 'draft' | 'archived'
export type OrderStatus = 'pending' | 'paid' | 'shipped' | 'delivered' | 'refunded' | 'failed'
export type Segment = 'VIP' | 'Regular' | 'New' | 'At risk'
export type Role = 'owner' | 'admin' | 'editor' | 'support' | 'viewer'
export type MemberStatus = 'active' | 'invited' | 'suspended'

export interface Product {
  id: number
  name: string
  sku: string
  category: Category
  priceCents: number
  stock: number
  status: ProductStatus
  sold30d: number
  rating: number
  hue: number
  trend: number[]
  description: string
  updatedAt: string
}

export type ProductInput = Pick<Product, 'name' | 'sku' | 'category' | 'priceCents' | 'stock' | 'status' | 'description'>

export interface ProductStats {
  total: number
  active: number
  lowStock: number
  outOfStock: number
  inventoryValueCents: number
  byCategory: Partial<Record<Category, number>>
}

export interface OrderItem {
  productId: number
  name: string
  sku: string
  category: Category
  hue: number
  qty: number
  priceCents: number
}

export interface Order {
  id: number
  customer: { id: number; name: string; email: string; country: string; segment: Segment }
  items: OrderItem[]
  totalCents: number
  status: OrderStatus
  payment: string
  placedAt: string
}

export interface Customer {
  id: number
  name: string
  email: string
  country: string
  orders: number
  ltvCents: number
  segment: Segment
  lastSeenAt: string
  createdAt: string
}

export interface Member {
  id: number
  name: string
  email: string
  role: Role
  status: MemberStatus
  mfa: boolean
  lastActiveAt: string | null
}

export type MemberInput = Pick<Member, 'name' | 'email' | 'role'>

export interface KPI {
  key: string
  label: string
  value: number
  unit: 'count' | 'percent' | 'cents'
  deltaPct: number
  trend: number[]
}

export interface Dashboard {
  rangeDays: number
  revenueCents: number
  prevRevenueCents: number
  revenue: { date: string; current: number; previous: number }[]
  kpis: KPI[]
  ordersHeatmap: number[][]
  categories: { category: Category; salesCents: number }[]
  topProducts: { id: number; name: string; category: Category; hue: number; sold: number; revenueCents: number }[]
  recentOrders: Order[]
  activity: { kind: string; actor: string; message: string; at: string }[]
  markets: { country: string; name: string; sharePct: number }[]
  target: { label: string; bookedCents: number; goalCents: number; pacePct: number }
}

export interface RuntimeStats {
  goroutines: number
  heapAllocMb: number
  heapSysMb: number
  numGc: number
  gcPauseMaxMs: number
  gomaxprocs: number
  uptimeSeconds: number
  goVersion: string
  platform: string
  revision: string
  requestsPerSec: number
  onlineUsers: number
}

export interface Meta {
  version: string
  env: string
  goVersion: string
  products: number
  pendingOrders: number
}

export interface RequestEntry {
  time: string
  method: string
  path: string
  status: number
  durationMs: number
  bytes: number
  requestId: string
  ip: string
}

export interface RequestStats {
  total: number
  successRate: number
  p95Ms: number
  client4xx: number
  server5xx: number
}

export class ApiError extends Error {
  readonly status: number
  readonly fields?: Record<string, string>

  constructor(status: number, message: string, fields?: Record<string, string>) {
    super(message)
    this.status = status
    this.fields = fields
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch('/api/v1' + path, {
    ...init,
    headers: { Accept: 'application/json', ...(init?.body ? { 'Content-Type': 'application/json' } : {}), ...init?.headers },
  })
  if (res.status === 204) return undefined as T
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new ApiError(res.status, body.error ?? res.statusText, body.fields)
  return body as T
}

const qs = (params: Record<string, string | number | undefined>) => {
  const s = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '' && v !== 'all') s.set(k, String(v))
  const str = s.toString()
  return str ? '?' + str : ''
}

const json = (method: string, body: unknown): RequestInit => ({ method, body: JSON.stringify(body) })

export const api = {
  meta: () => request<Meta>('/meta'),
  runtime: () => request<RuntimeStats>('/runtime'),
  dashboard: (range: number) => request<Dashboard>(`/dashboard${qs({ range })}`),

  products: (f: { q?: string; category?: string; status?: string; sort?: string }) =>
    request<{ items: Product[]; stats: ProductStats }>(`/products${qs(f)}`),
  createProduct: (in_: ProductInput) => request<Product>('/products', json('POST', in_)),
  updateProduct: (id: number, in_: ProductInput) => request<Product>(`/products/${id}`, json('PUT', in_)),
  deleteProduct: (id: number) => request<void>(`/products/${id}`, { method: 'DELETE' }),
  bulkProducts: (ids: number[], action: 'publish' | 'archive' | 'delete') =>
    request<{ affected: number }>('/products/bulk', json('POST', { ids, action })),

  orders: (f: { q?: string; status?: string; limit?: number }) =>
    request<{ items: Order[]; counts: Partial<Record<OrderStatus, number>> }>(`/orders${qs(f)}`),
  updateOrderStatus: (id: number, status: OrderStatus) => request<Order>(`/orders/${id}/status`, json('PATCH', { status })),

  customers: (f: { q?: string; segment?: string }) =>
    request<{ items: Customer[]; segments: Partial<Record<Segment, { count: number; ltvCents: number }>> }>(`/customers${qs(f)}`),

  team: (role?: string) => request<{ items: Member[] }>(`/team${qs({ role })}`),
  createMember: (in_: MemberInput) => request<Member>('/team', json('POST', in_)),
  updateMember: (id: number, in_: MemberInput) => request<Member>(`/team/${id}`, json('PUT', in_)),
  deleteMember: (id: number) => request<void>(`/team/${id}`, { method: 'DELETE' }),

  requests: (f: { q?: string; class?: string; limit?: number }) =>
    request<{ items: RequestEntry[]; stats: RequestStats }>(`/requests${qs(f)}`),
}
