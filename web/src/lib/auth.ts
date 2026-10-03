import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { api, type Permission } from './api'

export const meQuery = queryOptions({
  queryKey: ['me'],
  queryFn: api.me,
  staleTime: 60_000,
  retry: false,
})

/** The signed-in member. Only call inside the authenticated layout. */
export function useMe() {
  const { data } = useQuery(meQuery)
  if (!data) throw new Error('useMe used outside the authenticated layout')
  return data
}

export function useCan(perm: Permission) {
  const { data } = useQuery(meQuery)
  return !!data?.permissions.includes(perm)
}

export function useLogin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ email, password }: { email: string; password: string }) => api.login(email, password),
    onSuccess: (me) => {
      // Drop everything cached for the previous user before seeding the new one.
      qc.clear()
      qc.setQueryData(meQuery.queryKey, me)
    },
  })
}

export function useLogout() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  return useMutation({
    mutationFn: api.logout,
    onSettled: async () => {
      await navigate({ to: '/login' })
      qc.clear()
    },
  })
}
