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

// A 401 anywhere means the session expired: drop cached data and go to login.
const onAuthError = (err: unknown) => {
  if (err instanceof ApiError && err.status === 401 && location.pathname !== '/login') {
    queryClient.clear()
    router.navigate({ to: '/login', search: { redirect: location.pathname + location.search } })
  }
}

const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: onAuthError }),
  mutationCache: new MutationCache({ onError: onAuthError }),
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
