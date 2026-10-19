import { useEffect } from 'react'
import { Link, useParams } from 'react-router'

import { ErrorNotice } from '../../components/ErrorNotice'
import { Money } from '../../components/Money'
import { Button, Card, Stat } from '../../components/ui'
import { rememberWallet } from '../../lib/recentWallets'
import { useLedger, useReconcile, useWallet } from './api'

export function WalletDetail() {
  const id = useParams().id ?? ''
  const wallet = useWallet(id)
  const ledger = useLedger(id)
  const reconcile = useReconcile(id)

  useEffect(() => {
    if (wallet.isSuccess) {
      rememberWallet(id)
    }
  }, [id, wallet.isSuccess])

  const entries = ledger.data?.pages.flatMap((page) => page.entries) ?? []

  return (
    <div className="flex flex-col gap-6">
      <div>
        <Link to="/wallets" className="text-sm text-brand-700 hover:underline">
          ← Wallets
        </Link>
        <h1 className="mt-1 break-all text-xl font-semibold">Wallet {id}</h1>
      </div>

      {wallet.isError && <ErrorNotice error={wallet.error} />}
      {wallet.data && (
        <Card>
          <dl className="grid gap-4 sm:grid-cols-3">
            <Stat label="Balance">
              <Money value={wallet.data.balance} />
            </Stat>
            <Stat label="Version">{wallet.data.version}</Stat>
            <div>
              <dt className="text-xs font-medium text-slate-500">Player</dt>
              <dd className="mt-1 break-all font-mono text-sm">{wallet.data.playerId}</dd>
            </div>
          </dl>
          <div className="mt-5 flex flex-wrap items-center gap-4">
            <Button
              type="button"
              variant="secondary"
              disabled={reconcile.isPending}
              onClick={() => {
                reconcile.mutate()
              }}
            >
              {reconcile.isPending ? 'Reconciling…' : 'Reconcile'}
            </Button>
            {reconcile.isError && <ErrorNotice error={reconcile.error} />}
            {reconcile.data && (
              <div
                role="status"
                aria-label="Reconciliation"
                className={`rounded-lg px-3 py-2 text-sm ${
                  reconcile.data.consistent ? 'bg-emerald-50 text-emerald-900' : 'bg-red-50 text-red-900'
                }`}
              >
                <strong>{reconcile.data.consistent ? 'Consistent' : 'Inconsistent'}</strong>: stored{' '}
                <Money value={reconcile.data.storedBalance} />, from the ledger{' '}
                <Money value={reconcile.data.calculatedBalance} />, difference{' '}
                <Money value={reconcile.data.difference} /> over {reconcile.data.checkedEntries} ledger{' '}
                {reconcile.data.checkedEntries === 1 ? 'entry' : 'entries'}
              </div>
            )}
          </div>
        </Card>
      )}

      <Card title="Ledger">
        {ledger.isError && <ErrorNotice error={ledger.error} />}
        <div className="overflow-x-auto">
          <table aria-label="Ledger" className="w-full min-w-[40rem] text-left text-sm">
            <thead className="text-xs text-slate-500">
              <tr>
                <th className="py-2 pr-4 font-medium">When</th>
                <th className="py-2 pr-4 font-medium">Direction</th>
                <th className="py-2 pr-4 text-right font-medium">Amount</th>
                <th className="py-2 pr-4 text-right font-medium">Balance after</th>
                <th className="py-2 font-medium">Transaction</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {entries.map((entry) => (
                <tr key={entry.id}>
                  <td className="py-2 pr-4 whitespace-nowrap tabular-nums">
                    {new Date(entry.createdAt).toLocaleString('en-GB')}
                  </td>
                  <td className="py-2 pr-4">{entry.direction}</td>
                  <td className="py-2 pr-4 text-right">
                    {entry.direction === 'DEBIT' ? '−' : '+'}
                    <Money value={entry.amount} />
                  </td>
                  <td className="py-2 pr-4 text-right">
                    <Money value={entry.balanceAfter} />
                  </td>
                  <td className="py-2 font-mono text-xs">
                    <Link to={`/transactions?id=${entry.transactionId}`} className="text-brand-700 hover:underline">
                      {entry.transactionId}
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {ledger.isSuccess && entries.length === 0 && <p className="py-2 text-sm text-slate-500">No entries yet.</p>}
        {ledger.hasNextPage && (
          <Button
            type="button"
            variant="secondary"
            className="mt-4"
            disabled={ledger.isFetchingNextPage}
            onClick={() => void ledger.fetchNextPage()}
          >
            {ledger.isFetchingNextPage ? 'Loading…' : 'Load more'}
          </Button>
        )}
      </Card>
    </div>
  )
}
