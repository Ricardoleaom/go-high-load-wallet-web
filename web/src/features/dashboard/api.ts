import { useQuery } from '@tanstack/react-query'

import { unwrap } from '../../api/client'
import { useApi } from '../../api/context'

// useOpsSummary follows GET /ops/summary, asking again once per sampling
// interval: there is nothing new before.
export function useOpsSummary() {
  const api = useApi()
  return useQuery({
    queryKey: ['ops-summary'],
    queryFn: () => unwrap(api.GET('/ops/summary')),
    refetchInterval: (query) => (query.state.data?.intervalSeconds ?? 5) * 1000,
  })
}
