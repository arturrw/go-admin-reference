import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { api, ApiError, type MemberInput, type OrderStatus, type Permission, type Product, type ProductInput } from './api'
import { money } from './format'

// Query keys are grouped by resource so a mutation can invalidate everything
// that depends on it with a single prefix.
export const keys = {
  meta: ['meta'] as const,
  runtime: ['runtime'] as const,
  live: ['live'] as const,
  logLevel: ['log-level'] as const,
  apiKeys: ['api-keys'] as const,
  settings: ['settings'] as const,
  deliveries: ['webhook-deliveries'] as const,
  dashboard: (range: number) => ['dashboard', range] as const,
  products: (f: object) => ['products', f] as const,
  orders: (f: object) => ['orders', f] as const,
  customers: (f: object) => ['customers', f] as const,
  team: (role: string) => ['team', role] as const,
  requests: (f: object) => ['requests', f] as const,
  request: (id: string) => ['request', id] as const,
  product: (id: number) => ['product', id] as const,
  order: (id: number) => ['order', id] as const,
  customer: (id: number) => ['customer', id] as const,
  roles: ['roles'] as const,
  activity: (f: object) => ['activity', f] as const,
  member: (id: number) => ['member', id] as const,
}

// Polled so maintenance mode (and the sidebar counters) reach open tabs.
export const useMeta = () => useQuery({ queryKey: keys.meta, queryFn: api.meta, staleTime: 10_000, refetchInterval: 15_000 })

export const useRuntime = (enabled = true) =>
  useQuery({ queryKey: keys.runtime, queryFn: api.runtime, refetchInterval: 2000, enabled })

export const useLive = () => useQuery({ queryKey: keys.live, queryFn: api.live, refetchInterval: 2000 })

export const useLogLevel = () => useQuery({ queryKey: keys.logLevel, queryFn: api.logLevel })

export const useSettings = (enabled = true) => useQuery({ queryKey: keys.settings, queryFn: api.settings, enabled })

export function usePatchSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.patchSettings,
    onSuccess: (s) => {
      qc.setQueryData(keys.settings, s)
      toast.success('Settings saved')
      return qc.invalidateQueries({ queryKey: keys.meta })
    },
    onError: onFormError,
  })
}

export const useDeliveries = (enabled = true) =>
  useQuery({ queryKey: keys.deliveries, queryFn: api.webhookDeliveries, enabled, refetchInterval: 4000 })

export function useTestWebhook() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.testWebhook,
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.deliveries }),
    onError,
  })
}

export function useRotateSecret() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.rotateWebhookSecret,
    onSuccess: (s) => {
      qc.setQueryData(keys.settings, s)
      toast.success('Signing secret rotated')
    },
    onError,
  })
}

export const useApiKeys = (enabled = true) => useQuery({ queryKey: keys.apiKeys, queryFn: api.apiKeys, enabled })

export function useCreateApiKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createApiKey,
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.apiKeys }),
    onError: onFormError,
  })
}

export function useRevokeApiKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.revokeApiKey,
    onSuccess: () => {
      toast.success('Key revoked')
      return qc.invalidateQueries({ queryKey: keys.apiKeys })
    },
    onError,
  })
}

export function useSetLogLevel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.setLogLevel,
    onSuccess: (res) => {
      qc.setQueryData(keys.logLevel, res)
      toast.success(`Log level set to ${res.level}`, { description: 'Applied to the running server. LOG_LEVEL is used again after a restart.' })
    },
    onError,
  })
}

export function useSetTarget() {
  return useMutation({
    mutationFn: api.setTarget,
    // The dashboard (and its target) is refetched after every mutation.
    onSuccess: (t) => toast.success(`${t.label} set to ${money(t.goalCents)}`),
    onError: onFormError,
  })
}

export function useImportProducts() {
  const invalidate = useInvalidate('products', 'product', 'dashboard', 'meta', 'live')
  return useMutation({
    mutationFn: api.importProducts,
    onSuccess: ({ created, updated }) => {
      toast.success(`Imported ${created + updated} products`, { description: `${created} created · ${updated} updated` })
      return invalidate()
    },
  })
}

export const useDashboard = (range: number) =>
  useQuery({ queryKey: keys.dashboard(range), queryFn: () => api.dashboard(range), placeholderData: keepPreviousData })

export const useProducts = (f: Parameters<typeof api.products>[0]) =>
  useQuery({ queryKey: keys.products(f), queryFn: () => api.products(f), placeholderData: keepPreviousData })

export const useOrders = (f: Parameters<typeof api.orders>[0], enabled = true) =>
  useQuery({ queryKey: keys.orders(f), queryFn: () => api.orders(f), placeholderData: keepPreviousData, enabled })

export const useProduct = (id: number | undefined) =>
  useQuery({ queryKey: keys.product(id ?? 0), queryFn: () => api.product(id!), enabled: !!id })

export const useOrder = (id: number | undefined) =>
  useQuery({ queryKey: keys.order(id ?? 0), queryFn: () => api.order(id!), enabled: !!id })

export const useCustomer = (id: number | undefined) =>
  useQuery({ queryKey: keys.customer(id ?? 0), queryFn: () => api.customer(id!), enabled: !!id })

export const useRequestEntry = (id: string | undefined) =>
  useQuery({ queryKey: keys.request(id ?? ''), queryFn: () => api.request(id!), enabled: !!id, staleTime: Infinity })

export const useRoles = () => useQuery({ queryKey: keys.roles, queryFn: api.roles, staleTime: Infinity })

export const useActivity = (f: Parameters<typeof api.activity>[0], enabled = true) =>
  useQuery({ queryKey: keys.activity(f), queryFn: () => api.activity(f), placeholderData: keepPreviousData, enabled })

export const useCustomers = (f: Parameters<typeof api.customers>[0]) =>
  useQuery({ queryKey: keys.customers(f), queryFn: () => api.customers(f), placeholderData: keepPreviousData })

export const useTeam = (role: string, enabled = true) =>
  useQuery({ queryKey: keys.team(role), queryFn: () => api.team(role), placeholderData: keepPreviousData, enabled })

export const useRequests = (f: Parameters<typeof api.requests>[0], live: boolean) =>
  useQuery({
    queryKey: keys.requests(f),
    queryFn: () => api.requests(f),
    placeholderData: keepPreviousData,
    refetchInterval: live ? 1500 : false,
  })

export function errorMessage(err: unknown) {
  if (err instanceof ApiError && err.fields) {
    return Object.entries(err.fields)
      .map(([k, v]) => `${k} ${v}`)
      .join(', ')
  }
  return err instanceof Error ? err.message : 'Something went wrong'
}

function useInvalidate(...prefixes: string[]) {
  const qc = useQueryClient()
  return () => Promise.all(prefixes.map((p) => qc.invalidateQueries({ queryKey: [p] })))
}

const onError = (err: unknown) => toast.error(errorMessage(err))

/** Forms show field errors inline; only toast the rest (403, 500…). */
const onFormError = (err: unknown) => {
  if (!(err instanceof ApiError && err.fields)) toast.error(errorMessage(err))
}

export function useSaveProduct() {
  const invalidate = useInvalidate('products', 'product', 'dashboard', 'meta')
  return useMutation({
    mutationFn: ({ id, input }: { id?: number; input: ProductInput }) => (id ? api.updateProduct(id, input) : api.createProduct(input)),
    onSuccess: (_, { id }) => {
      toast.success(id ? 'Product updated' : 'Product created')
      return invalidate()
    },
    onError: onFormError,
  })
}

/** Image mutations write the returned product straight into the cache. */
function useImageMutation<V>(fn: (v: V) => Promise<Product>, message: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: (p) => {
      qc.setQueryData(keys.product(p.id), p)
      toast.success(message)
      return qc.invalidateQueries({ queryKey: ['products'] })
    },
    onError,
  })
}

export const useUploadImage = () =>
  useImageMutation(({ id, file }: { id: number; file: File }) => api.uploadImage(id, file), 'Image uploaded')
export const useDeleteImage = () =>
  useImageMutation(({ id, imageId }: { id: number; imageId: string }) => api.deleteImage(id, imageId), 'Image removed')
export const useSetPrimaryImage = () =>
  useImageMutation(({ id, imageId }: { id: number; imageId: string }) => api.setPrimaryImage(id, imageId), 'Cover image updated')

export function useAddCustomerNote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, text }: { id: number; text: string }) => api.addCustomerNote(id, text),
    onSuccess: (_, { id }) => {
      toast.success('Note added')
      return qc.invalidateQueries({ queryKey: keys.customer(id) })
    },
    onError,
  })
}

export function useDeleteCustomerNote() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, noteId }: { id: number; noteId: number }) => api.deleteCustomerNote(id, noteId),
    onSuccess: (_, { id }) => {
      toast.success('Note deleted')
      return qc.invalidateQueries({ queryKey: keys.customer(id) })
    },
    onError,
  })
}

export function useDeleteProduct() {
  const invalidate = useInvalidate('products', 'product', 'dashboard', 'meta')
  return useMutation({
    mutationFn: (id: number) => api.deleteProduct(id),
    onSuccess: () => {
      toast.success('Product deleted')
      return invalidate()
    },
    onError,
  })
}

export function useBulkProducts() {
  const invalidate = useInvalidate('products', 'dashboard', 'meta')
  return useMutation({
    mutationFn: ({ ids, action }: { ids: number[]; action: 'publish' | 'archive' | 'delete' }) => api.bulkProducts(ids, action),
    onSuccess: ({ affected }, { action }) => {
      toast.success(`${affected} products ${action === 'delete' ? 'deleted' : action === 'publish' ? 'published' : 'archived'}`)
      return invalidate()
    },
    onError,
  })
}

export function useUpdateOrderStatus() {
  const invalidate = useInvalidate('orders', 'order', 'customer', 'customers', 'dashboard', 'meta')
  return useMutation({
    mutationFn: ({ id, status, reason }: { id: number; status: OrderStatus; reason?: string }) => api.updateOrderStatus(id, status, reason),
    onSuccess: (o) => {
      toast.success(o.refund ? `Order #${o.id} refunded` : `Order #${o.id} marked as ${o.status}`, o.refund ? { description: o.refund.reason } : undefined)
      return invalidate()
    },
    onError: onFormError,
  })
}

export function useSaveMember() {
  const invalidate = useInvalidate('team', 'member')
  return useMutation({
    mutationFn: ({ id, input }: { id?: number; input: MemberInput }) => (id ? api.updateMember(id, input) : api.createMember(input)),
    onSuccess: (m, { id }) => {
      toast.success(id ? 'Member updated' : `Invite sent to ${m.email}`)
      return invalidate()
    },
    onError: onFormError,
  })
}

export const useMember = (id: number) => useQuery({ queryKey: keys.member(id), queryFn: () => api.member(id), refetchInterval: 30_000 })

export function useSetMemberAccess() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...access }: { id: number; granted: Permission[]; revoked: Permission[] }) => api.setMemberAccess(id, access),
    onSuccess: (d) => {
      qc.setQueryData(keys.member(d.member.id), d)
      toast.success(`Access updated for ${d.member.name}`)
      return qc.invalidateQueries({ queryKey: ['team'] })
    },
    onError,
  })
}

export function useSetMemberStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: number; status: 'active' | 'suspended' }) => api.setMemberStatus(id, status),
    onSuccess: (d) => {
      qc.setQueryData(keys.member(d.member.id), d)
      toast.success(d.member.status === 'suspended' ? `Suspended ${d.member.name}` : `Reactivated ${d.member.name}`)
      return qc.invalidateQueries({ queryKey: ['team'] })
    },
    onError,
  })
}

export function useDeleteMember() {
  const invalidate = useInvalidate('team')
  return useMutation({
    mutationFn: (id: number) => api.deleteMember(id),
    onSuccess: () => {
      toast.success('Member removed')
      return invalidate()
    },
    onError,
  })
}
