import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { BrowserRouter, Route, Routes } from 'react-router'

import { ApiProvider } from './api/ApiProvider'
import { SignIn } from './auth/SignIn'
import { useSession } from './auth/session'
import { Layout } from './components/Layout'
import { DashboardPage } from './features/dashboard/DashboardPage'
import { LoadLabPage } from './features/loadlab/LoadLabPage'
import { TransactionsPage } from './features/transactions/TransactionsPage'
import { WalletsPage } from './features/wallets/WalletsPage'

export function App() {
  const session = useSession()
  const [queryClient] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } } }),
  )

  if (session.status === 'loading') {
    return (
      <p role="status" className="p-8 text-sm text-slate-600">
        Signing in…
      </p>
    )
  }
  if (session.status !== 'signed-in') {
    return <SignIn />
  }
  return (
    <QueryClientProvider client={queryClient}>
      <ApiProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<Layout />}>
              <Route index element={<DashboardPage />} />
              <Route path="wallets/*" element={<WalletsPage />} />
              <Route path="transactions" element={<TransactionsPage />} />
              <Route path="load" element={<LoadLabPage />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ApiProvider>
    </QueryClientProvider>
  )
}
