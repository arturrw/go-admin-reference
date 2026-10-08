// Typed client for the Go API (/api/v1). Field names mirror the json tags in
// internal/domain. Money is always integer cents.

export type Category = 'Audio' | 'Wearables' | 'Lighting' | 'Home' | 'Computing' | 'Accessories'
export const CATEGORIES: Category[] = ['Audio', 'Wearables', 'Lighting', 'Home', 'Computing', 'Accessories']

export type ProductStatus = 'active' | 'draft' | 'archived'
export type OrderStatus = 'pending' | 'paid' | 'shipped' | 'delivered' | 'refunded' | 'failed' | 'cancelled'
export type Segment = 'VIP' | 'Regular' | 'New' | 'At risk'
export type Role = 'owner' | 'admin' | 'editor' | 'support' | 'viewer'
export type MemberStatus = 'active' | 'invited' | 'suspended'
export type Permission =
  | 'dashboard:read'
  | 'products:read'
  | 'products:write'
  | 'orders:read'
  | 'orders:write'
  | 'customers:read'
  | 'customers:write'
  | 'team:read'
  | 'team:write'
  | 'requests:read'
  | 'settings:write'
  | 'workspace:manage'

export interface ProductImage {
  id: string
  url: string
  alt: string
  generated: boolean
  sizeBytes: number
}

export interface Product {
  id: number
  name: string
  sku: string
  category: Category
  vendor: string
  tags: string[]
  priceCents: number
  compareAtCents: number
  costCents: number
  stock: number
  weightGrams: number
  status: ProductStatus
  sold30d: number
  rating: number
  hue: number
  trend: number[]
  description: string
  images: ProductImage[]
  createdAt: string
  updatedAt: string
}

export type ProductInput = Pick<
  Product,
  'name' | 'sku' | 'category' | 'vendor' | 'tags' | 'priceCents' | 'compareAtCents' | 'costCents' | 'stock' | 'weightGrams' | 'status' | 'description'
>

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
  imageUrl: string
  qty: number
  priceCents: number
}

export const PAYMENT_METHODS = ['Visa •• 4242', 'Mastercard •• 5100', 'Apple Pay', 'PayPal', 'Amex •• 0005', 'Google Pay']

/** A sale entered by staff; prices come from the catalogue. */
export interface NewOrder {
  customerId: number
  payment: string
  items: { productId: number; qty: number }[]
}

export interface Order {
  id: number
  customer: { id: number; name: string; email: string; country: string; segment: Segment }
  items: OrderItem[]
  totalCents: number
  status: OrderStatus
  payment: string
  placedAt: string
  /** Set while the order is refunded: why, by whom and when. */
  refund: { reason: string; by: string; at: string } | null
  /** Only on a single order (GET /orders/{id}): what happened to it, oldest first. */
  events?: { status: OrderStatus; at: string; by: string }[]
  /** Only on a single order (GET /orders/{id}): what the customer was charged. */
  shippingCents?: number
  taxCents?: number
  grandTotalCents?: number
}

/** Presets offered in the refund dialog (mirrors domain.RefundReasons). */
export const REFUND_REASONS = [
  'Damaged in transit',
  'Wrong item sent',
  'Item not as described',
  'Arrived too late',
  'Customer changed their mind',
  'Duplicate order',
]

export interface CustomerNote {
  id: number
  author: string
  text: string
  at: string
}

export interface Customer {
  id: number
  name: string
  email: string
  phone: string
  country: string
  address: { line1: string; city: string; postalCode: string; country: string }
  orders: number
  ltvCents: number
  segment: Segment
  tags: string[]
  acceptsMarketing: boolean
  source: string
  notes: CustomerNote[]
  lastSeenAt: string
  lastOrderAt: string
  createdAt: string
}

export interface CustomerDetail {
  customer: Customer
  stats: {
    totalSpentCents: number
    orders: number
    avgOrderCents: number
    itemsBought: number
    refunds: number
    firstOrderAt: string
    lastOrderAt: string
  }
  orders: Order[]
  products: { productId: number; name: string; category: Category; hue: number; imageUrl: string; qty: number; spentCents: number }[]
  categories: { category: Category; salesCents: number }[]
  monthly: { month: string; cents: number }[]
}

export interface Member {
  id: number
  name: string
  email: string
  role: Role
  status: MemberStatus
  mfa: boolean
  lastActiveAt: string | null
  /** When a pending invitation lapses (invited members only). */
  inviteExpiresAt?: string | null
  /** Owner-set exceptions to the role. */
  granted: Permission[]
  revoked: Permission[]
}

export interface MemberDetail {
  member: Member
  /** Effective: role + granted − revoked. */
  permissions: Permission[]
  online: boolean
}

/** A one-time invitation link; the server can't send email, so the inviter delivers it. */
export interface Invite {
  token: string
  url: string
  expiresAt: string
}

export interface InviteInfo {
  name: string
  email: string
  role: Role
  serviceName: string
  expiresAt: string
}

export type MemberInput = Pick<Member, 'name' | 'email' | 'role'>

export interface Me {
  user: Member
  permissions: Permission[]
}

export interface RolesInfo {
  roles: Role[]
  permissions: { key: Permission; group: string; label: string; description: string }[]
  matrix: Record<Role, Permission[]>
}

export interface KPI {
  key: string
  label: string
  value: number
  unit: 'count' | 'percent' | 'cents'
  deltaPct: number
  /** A fixed sample: the orders cannot tell it (no storefront traffic behind this app). */
  synthetic?: boolean
  trend: number[]
  series: { date: string; current: number; previous: number }[]
}

export interface Dashboard {
  rangeDays: number
  revenueCents: number
  prevRevenueCents: number
  revenue: { date: string; current: number; previous: number }[]
  kpis: KPI[]
  ordersHeatmap: number[][]
  categories: { category: Category; salesCents: number }[]
  topProducts: { id: number; name: string; category: Category; hue: number; imageUrl: string; sold: number; revenueCents: number }[]
  recentOrders: Order[]
  activity: Activity[]
  markets: { country: string; name: string; sharePct: number }[]
  target: Target
}

export type ActivityKind =
  | 'product'
  | 'customer'
  | 'publish'
  | 'image'
  | 'import'
  | 'order'
  | 'refund'
  | 'note'
  | 'team'
  | 'role'
  | 'target'
  | 'settings'
  | 'auth'
  | 'alert'
  | 'deploy'
  | 'stock'

/** One audit-log entry; `entity` + `entityId` say what a click opens. */
export interface Activity {
  id: number
  kind: ActivityKind
  actorId: number
  actor: string
  message: string
  entity: '' | 'product' | 'order' | 'customer' | 'member'
  entityId: number
  at: string
}

/** The current quarter's revenue goal; the owner can change goalCents. */
export interface Target {
  label: string
  quarter: string
  period: string
  bookedCents: number
  goalCents: number
  pacePct: number
  updatedBy: string
  updatedAt: string | null
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

export interface LiveStats {
  at: string
  requestsPerSec: number
  onlineUsers: number
  activeCarts: number
  checkoutsPerMin: number
  conversionPct: number
  avgSessionSec: number
  bounceRatePct: number
  history: { at: string; requestsPerSec: number; onlineUsers: number; hotViewers: number }[]
  devices: { name: string; pct: number }[]
  sources: { name: string; pct: number }[]
  topPages: { path: string; viewers: number }[]
  hot: {
    id: number
    name: string
    category: Category
    hue: number
    imageUrl: string
    sku: string
    priceCents: number
    stock: number
    sold30d: number
    rating: number
    viewers: number
    inCarts: number
    soldToday: number
    revenueTodayCents: number
    conversionPct: number
    shareOfTrafficPct: number
  } | null
}

/** Workspace settings, edited in Settings → General. */
export interface WorkspaceSettings {
  serviceName: string
  publicBaseUrl: string
  maintenance: boolean
  auditLog: boolean
  loginAlerts: boolean
  /** Effective session lifetime: SESSION_TTL until an admin picks one. */
  sessionTtlSeconds: number
  webhookUrl: string
  webhooksSigned: boolean
  webhookSecret: string
  listenAddr: string
  env: string
}

/** One webhook event and how sending it went (kept in memory, newest first). */
export interface WebhookDelivery {
  id: string
  type: string
  url: string
  ok: boolean
  status: number
  error?: string
  attempts: number
  durationMs: number
  at: string
}

/** What staff enter to add a customer by hand. */
export interface CustomerInput {
  name: string
  email: string
  phone: string
  country: string
  address: { line1: string; city: string; postalCode: string; country: string }
  tags: string[]
  acceptsMarketing: boolean
  source: string
}

export type ApiKeyScope = 'read' | 'write'

/** A server-to-server key. The secret is only returned when it is created. */
export interface ApiKey {
  id: number
  name: string
  scope: ApiKeyScope
  last4: string
  masked: string
  createdBy: string
  createdAt: string
  lastUsedAt: string | null
}

export interface Notification extends Activity {
  unread: boolean
}

export type LogLevel = 'debug' | 'info' | 'warn' | 'error'

export interface ImportRowError {
  row: number
  sku?: string
  field: string
  message: string
}

export interface ImportResult {
  created: number
  updated: number
  errors: ImportRowError[]
}

export interface Meta {
  serviceName: string
  maintenance: boolean
  version: string
  env: string
  goVersion: string
  products: number
  pendingOrders: number
}

export interface RequestSummary {
  id: string
  time: string
  method: string
  path: string
  status: number
  durationMs: number
  bytes: number
  ip: string
  actor: string
}

export interface RequestEntry extends RequestSummary {
  query: string
  proto: string
  userAgent: string
  actorRole: string
  route: string
  requestHeaders: Record<string, string>
  responseHeaders: Record<string, string>
  requestBody: string
  responseBody: string
  requestBytes: number
  bodyTruncated: boolean
}

export interface RequestStats {
  total: number
  successRate: number
  p50Ms: number
  p95Ms: number
  p99Ms: number
  client4xx: number
  server5xx: number
  methods: Record<string, number>
  perMinute: number[]
  endpoints: { route: string; count: number; avgMs: number; p95Ms: number; errors: number }[]
}

export class ApiError extends Error {
  readonly status: number
  readonly fields?: Record<string, string>
  /** The decoded error response, for endpoints that return more than a message. */
  readonly body?: Record<string, unknown>

  constructor(status: number, message: string, fields?: Record<string, string>, body?: Record<string, unknown>) {
    super(message)
    this.status = status
    this.fields = fields
    this.body = body
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const isForm = init?.body instanceof FormData
  const res = await fetch('/api/v1' + path, {
    ...init,
    credentials: 'same-origin',
    headers: { Accept: 'application/json', ...(init?.body && !isForm ? { 'Content-Type': 'application/json' } : {}), ...init?.headers },
  })
  if (res.status === 204) return undefined as T
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new ApiError(res.status, body.error ?? res.statusText, body.fields, body)
  return body as T
}

const qs = (params: Record<string, string | number | undefined>) => {
  const s = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '' && v !== 'all') s.set(k, String(v))
  const str = s.toString()
  return str ? '?' + str : ''
}

const json = (method: string, body: unknown): RequestInit => ({ method, body: JSON.stringify(body) })

/** URL of a CSV export endpoint with the same filters as its list. */
export const exportUrl = (resource: 'products' | 'orders' | 'customers', f: Record<string, string | number | undefined> = {}) =>
  `/api/v1/${resource}/export${qs(f)}`

export const api = {
  login: (email: string, password: string) => request<Me>('/auth/login', json('POST', { email, password })),
  logout: () => request<void>('/auth/logout', { method: 'POST' }),
  me: () => request<Me>('/auth/me'),
  demoAccounts: () => request<{ password: string; accounts: { name: string; email: string; role: Role }[] }>('/auth/demo-accounts'),
  roles: () => request<RolesInfo>('/roles'),

  meta: () => request<Meta>('/meta'),
  runtime: () => request<RuntimeStats>('/runtime'),
  live: () => request<LiveStats>('/live'),
  clearRequestLog: () => request<{ cleared: number }>('/danger/clear-request-log', { method: 'POST' }),
  signOutEveryone: () => request<{ signedOut: number }>('/danger/sign-out-everyone', { method: 'POST' }),
  webhookDeliveries: () => request<{ items: WebhookDelivery[] }>('/settings/webhook/deliveries'),
  testWebhook: () => request<WebhookDelivery>('/settings/webhook/test', { method: 'POST' }),
  rotateWebhookSecret: () => request<WorkspaceSettings>('/settings/webhook/rotate-secret', { method: 'POST' }),
  notifications: () => request<{ items: Notification[]; unread: number }>('/notifications'),
  readNotifications: () => request<void>('/notifications/read', { method: 'POST' }),
  settings: () => request<WorkspaceSettings>('/settings'),
  patchSettings: (patch: Partial<Pick<WorkspaceSettings, 'serviceName' | 'publicBaseUrl' | 'maintenance' | 'auditLog' | 'loginAlerts' | 'sessionTtlSeconds' | 'webhookUrl' | 'webhooksSigned'>>) => request<WorkspaceSettings>('/settings', json('PATCH', patch)),
  apiKeys: () => request<{ items: ApiKey[] }>('/settings/api-keys'),
  createApiKey: (in_: { name: string; scope: ApiKeyScope }) => request<{ key: ApiKey; masked: string; secret: string }>('/settings/api-keys', json('POST', in_)),
  revokeApiKey: (id: number) => request<void>(`/settings/api-keys/${id}`, { method: 'DELETE' }),
  logLevel: () => request<{ level: LogLevel }>('/settings/log-level'),
  setLogLevel: (level: LogLevel) => request<{ level: LogLevel }>('/settings/log-level', json('PUT', { level })),
  dashboard: (range: number) => request<Dashboard>(`/dashboard${qs({ range })}`),
  setTarget: (goalCents: number) => request<Target>('/target', json('PUT', { goalCents })),

  products: (f: { q?: string; category?: string; status?: string; sort?: string }) =>
    request<{ items: Product[]; stats: ProductStats }>(`/products${qs(f)}`),
  product: (id: number) => request<Product>(`/products/${id}`),
  createProduct: (in_: ProductInput) => request<Product>('/products', json('POST', in_)),
  updateProduct: (id: number, in_: ProductInput) => request<Product>(`/products/${id}`, json('PUT', in_)),
  deleteProduct: (id: number) => request<void>(`/products/${id}`, { method: 'DELETE' }),
  bulkProducts: (ids: number[], action: 'publish' | 'archive' | 'delete') =>
    request<{ affected: number }>('/products/bulk', json('POST', { ids, action })),
  importProducts: (file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<ImportResult>('/products/import', { method: 'POST', body: fd })
  },
  uploadImage: (id: number, file: File) => {
    const fd = new FormData()
    fd.append('file', file)
    return request<Product>(`/products/${id}/images`, { method: 'POST', body: fd })
  },
  deleteImage: (id: number, imageId: string) => request<Product>(`/products/${id}/images/${encodeURIComponent(imageId)}`, { method: 'DELETE' }),
  setPrimaryImage: (id: number, imageId: string) =>
    request<Product>(`/products/${id}/images/${encodeURIComponent(imageId)}/primary`, { method: 'POST' }),

  orders: (f: { q?: string; status?: string; limit?: number; offset?: number; customer?: number; from?: string; to?: string }) =>
    request<{ items: Order[]; counts: Partial<Record<OrderStatus, number>>; total: number }>(`/orders${qs(f)}`),
  order: (id: number) => request<Order>(`/orders/${id}`),
  editOrderItems: (id: number, items: NewOrder['items']) => request<Order>(`/orders/${id}/items`, json('PUT', { items })),
  createOrder: (in_: NewOrder) => request<Order>('/orders', json('POST', in_)),
  updateOrderStatus: (id: number, status: OrderStatus, reason?: string) => request<Order>(`/orders/${id}/status`, json('PATCH', { status, reason })),

  customers: (f: { q?: string; segment?: string }) =>
    request<{ items: Customer[]; segments: Partial<Record<Segment, { count: number; ltvCents: number }>> }>(`/customers${qs(f)}`),
  customer: (id: number) => request<CustomerDetail>(`/customers/${id}`),
  createCustomer: (in_: CustomerInput) => request<Customer>('/customers', json('POST', in_)),
  addCustomerNote: (id: number, text: string) => request<CustomerNote>(`/customers/${id}/notes`, json('POST', { text })),
  deleteCustomerNote: (id: number, noteId: number) => request<void>(`/customers/${id}/notes/${noteId}`, { method: 'DELETE' }),

  team: (role?: string) => request<{ items: Member[] }>(`/team${qs({ role })}`),
  createMember: (in_: MemberInput) => request<Member & { invite: Invite }>('/team', json('POST', in_)),
  resendInvite: (id: number) => request<{ invite: Invite }>(`/team/${id}/invite`, { method: 'POST' }),
  inviteInfo: (token: string) => request<InviteInfo>(`/auth/invite/${encodeURIComponent(token)}`),
  acceptInvite: (token: string, body: { name: string; password: string }) => request<Me>(`/auth/invite/${encodeURIComponent(token)}`, json('POST', body)),
  updateMember: (id: number, in_: MemberInput) => request<Member>(`/team/${id}`, json('PUT', in_)),
  deleteMember: (id: number) => request<void>(`/team/${id}`, { method: 'DELETE' }),
  member: (id: number) => request<MemberDetail>(`/team/${id}`),
  setMemberAccess: (id: number, access: { granted: Permission[]; revoked: Permission[] }) =>
    request<MemberDetail>(`/team/${id}/access`, json('PUT', access)),
  setMemberStatus: (id: number, status: 'active' | 'suspended') => request<MemberDetail>(`/team/${id}/status`, json('PUT', { status })),

  activity: (f: { actor?: number; kind?: string; q?: string; limit?: number; offset?: number }) =>
    request<{ items: Activity[]; total: number; kinds: ActivityKind[] }>(`/activity${qs(f)}`),

  requests: (f: { q?: string; class?: string; method?: string; limit?: number }) =>
    request<{ items: RequestSummary[]; stats: RequestStats }>(`/requests${qs(f)}`),
  request: (id: string) => request<RequestEntry>(`/requests/${encodeURIComponent(id)}`),
}
