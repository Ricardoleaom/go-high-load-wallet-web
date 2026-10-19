import { useMemo, type ReactNode } from 'react'

import { useSession } from '../auth/session'
import { createApiClient } from './client'
import { ApiContext } from './context'

// ApiProvider gives every component one client for the signed-in operator,
// rebuilt when the token renews (every few minutes); queries are keyed by
// what they ask, not by the client, so a renewal refetches nothing.
export function ApiProvider({ children, baseUrl }: { children: ReactNode; baseUrl?: string }) {
  const token = useSession().user?.accessToken
  const client = useMemo(() => createApiClient(() => token, baseUrl), [token, baseUrl])
  return <ApiContext value={client}>{children}</ApiContext>
}
