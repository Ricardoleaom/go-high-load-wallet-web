import { useQuery } from '@tanstack/react-query'
import { useState, type SyntheticEvent } from 'react'
import { Link, useSearchParams } from 'react-router'

import { unwrap } from '../../api/client'
import { useApi } from '../../api/context'
import type { Transaction } from '../../api/types'
import { ErrorNotice } from '../../components/ErrorNotice'
import { Money } from '../../components/Money'
import { StatusBadge } from '../../components/StatusBadge'
import { Button, Card, Field } from '../../components/ui'

// The search lives in the address (?id= or ?provider=&external=), so a
// result can be linked to, as the ledger does.
function useTransactionSearch(params: URLSearchParams) {
  const api = useApi()
  const id = params.get('id') ?? ''
  const provider = params.get('provider') ?? ''
  const external = params.get('external') ?? ''
  return useQuery({
    queryKey: ['transaction', id, provider, external],
    enabled: id !== '' || (provider !== '' && external !== ''),
    queryFn: () =>
      id !== ''
        ? unwrap(api.GET('/wagering/transactions/{id}', { params: { path: { id } } }))
        : unwrap(
            api.GET('/providers/{providerId}/wagering/transactions/{externalId}', {
              params: { path: { providerId: provider, externalId: external } },
            }),
          ),
  })
}

export function TransactionsPage() {
  const [params, setParams] = useSearchParams()
  const search = useTransactionSearch(params)
  const [id, setId] = useState(params.get('id') ?? '')
  const [provider, setProvider] = useState(params.get('provider') ?? '')
  const [external, setExternal] = useState(params.get('external') ?? '')

  const byID = (event: SyntheticEvent) => {
    event.preventDefault()
    setParams({ id: id.trim() })
  }
  const byExternal = (event: SyntheticEvent) => {
    event.preventDefault()
    setParams({ provider: provider.trim(), external: external.trim() })
  }

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-xl font-semibold">Transactions</h1>
      <div className="grid gap-6 md:grid-cols-2">
        <Card title="By ID">
          <form aria-label="Find by ID" onSubmit={byID} className="flex items-end gap-2">
            <div className="flex-1">
              <Field
                label="Transaction ID"
                id="transaction-id"
                value={id}
                onChange={(e) => {
                  setId(e.target.value)
                }}
                required
              />
            </div>
            <Button type="submit" variant="secondary">
              Find
            </Button>
          </form>
        </Card>
        <Card title="By provider and external ID">
          <form aria-label="Find by provider and external ID" onSubmit={byExternal} className="flex flex-col gap-3">
            <div className="grid grid-cols-2 gap-2">
              <Field
                label="Provider"
                id="provider"
                value={provider}
                onChange={(e) => {
                  setProvider(e.target.value)
                }}
                required
              />
              <Field
                label="External ID"
                id="external-id"
                value={external}
                onChange={(e) => {
                  setExternal(e.target.value)
                }}
                required
              />
            </div>
            <Button type="submit" variant="secondary" className="self-start">
              Find
            </Button>
          </form>
        </Card>
      </div>
      {search.isError && <ErrorNotice error={search.error} />}
      {search.data && <TransactionCard transaction={search.data} />}
    </div>
  )
}

function TransactionCard({ transaction }: { transaction: Transaction }) {
  const name = transaction.externalTransactionId ?? transaction.transactionId
  return (
    <Card title={`Transaction ${name}`}>
      <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2">
        <Row label="Status">
          <StatusBadge status={transaction.status} />
          {transaction.failureCode && <span className="ml-2 text-red-700">{transaction.failureCode}</span>}
        </Row>
        <Row label="Kind">{transaction.kind}</Row>
        <Row label="Amount">
          <Money value={transaction.money} />
        </Row>
        <Row label="Balance after">{transaction.balance ? <Money value={transaction.balance} /> : '—'}</Row>
        <Row label="Provider">{transaction.providerId ?? '—'}</Row>
        <Row label="Transaction ID">
          <span className="font-mono text-xs">{transaction.transactionId}</span>
        </Row>
        <Row label="Wallet">
          <Link to={`/wallets/${transaction.walletId}`} className="font-mono text-xs text-brand-700 hover:underline">
            {transaction.walletId}
          </Link>
        </Row>
        {transaction.referenceExternalTransactionId && (
          <Row label="Reverses">{transaction.referenceExternalTransactionId}</Row>
        )}
      </dl>
    </Card>
  )
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <dt className="text-xs font-medium text-slate-500">{label}</dt>
      <dd className="mt-0.5 break-all">{children}</dd>
    </div>
  )
}
