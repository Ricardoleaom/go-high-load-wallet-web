import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { unwrap } from '../../api/client'
import { useApi } from '../../api/context'
import type { Money } from '../../api/types'

const ledgerPageSize = 20

export function useWallet(id: string) {
  const api = useApi()
  return useQuery({
    queryKey: ['wallet', id],
    queryFn: () => unwrap(api.GET('/wallets/{id}', { params: { path: { id } } })),
  })
}

export function useLedger(id: string) {
  const api = useApi()
  return useInfiniteQuery({
    queryKey: ['ledger', id],
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) =>
      unwrap(
        api.GET('/wallets/{id}/ledger', {
          params: { path: { id }, query: { limit: ledgerPageSize, ...(pageParam ? { cursor: pageParam } : {}) } },
        }),
      ),
    getNextPageParam: (page) => page.nextCursor,
  })
}

export function useOpenWallet() {
  const api = useApi()
  return useMutation({
    mutationFn: (input: { playerId: string; initialBalance: Money }) =>
      unwrap(api.POST('/wallets', { body: input })),
  })
}

export function useReconcile(id: string) {
  const api = useApi()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => unwrap(api.POST('/wallets/{id}/reconciliation', { params: { path: { id } } })),
    // The comparison read the wallet again; so does the page.
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['wallet', id] }),
  })
}
