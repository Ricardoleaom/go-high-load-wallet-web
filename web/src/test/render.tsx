import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router'

import { ApiProvider } from '../api/ApiProvider'
import { signedIn, WithSession } from './session'

// renderSignedIn renders ui as the console does for a signed-in operator:
// session, API client, a fresh query cache without retries, and a router
// at route, with ui mounted at path.
export function renderSignedIn(ui: ReactNode, { route = '/', path = '*' }: { route?: string; path?: string } = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <WithSession session={signedIn()}>
      <QueryClientProvider client={queryClient}>
        <ApiProvider>
          <MemoryRouter initialEntries={[route]}>
            <Routes>
              <Route path={path} element={ui} />
            </Routes>
          </MemoryRouter>
        </ApiProvider>
      </QueryClientProvider>
    </WithSession>,
  )
}
