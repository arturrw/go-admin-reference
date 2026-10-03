import '@fontsource-variable/geist'
import '@fontsource-variable/geist-mono'
import './index.css'
import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { ApiError } from './lib/api'
import { restoreAccent } from './lib/theme'
import { createAppRouter } from './router'

restoreAccent()

// A 401 anywhere means the session expired: go to login, then drop the data
// of the pages we left. (A 401 on `me` itself is handled by the route guard.)
const onAuthError = (err: unknown, key?: readonly unknown[]) => {
  if (!(err instanceof ApiError && err.status === 401) || key?.[0] === 'me' || location.pathname === '/login') return
  router
    .navigate({ to: '/login', search: { redirect: location.pathname + location.search } })
    // Only queries nobody renders any more — the login page's own queries stay.
    .then(() => queryClient.removeQueries({ predicate: (q) => q.getObserversCount() === 0 }))
}

const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: (err, query) => onAuthError(err, query.queryKey) }),
  mutationCache: new MutationCache({ onError: (err) => onAuthError(err) }),
  defaultOptions: {
    queries: {
      staleTime: 15_000,
      refetchOnWindowFocus: false,
      retry: (n, err) => !(err instanceof ApiError && err.status < 500) && n < 1,
    },
  },
})

const router = createAppRouter(queryClient)

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
)
