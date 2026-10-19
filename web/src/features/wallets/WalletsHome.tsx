import { useState, type SyntheticEvent } from 'react'
import { Link, useNavigate } from 'react-router'

import { ErrorNotice } from '../../components/ErrorNotice'
import { Button, Card, Field } from '../../components/ui'
import { recentWallets, rememberWallet } from '../../lib/recentWallets'
import { useOpenWallet } from './api'

const currencies = ['BRL', 'USD', 'EUR'] as const

export function WalletsHome() {
  const navigate = useNavigate()
  const openWallet = useOpenWallet()
  const [playerId, setPlayerId] = useState<string>(() => crypto.randomUUID())
  const [amount, setAmount] = useState('1000.00')
  const [currency, setCurrency] = useState<string>('BRL')
  const [lookup, setLookup] = useState('')
  const [recent] = useState(recentWallets)

  const submitOpen = (event: SyntheticEvent) => {
    event.preventDefault()
    openWallet.mutate(
      { playerId: playerId.trim(), initialBalance: { amount: amount.trim(), currency } },
      {
        onSuccess: (wallet) => {
          rememberWallet(wallet.id)
          void navigate(`/wallets/${wallet.id}`)
        },
      },
    )
  }

  const submitLookup = (event: SyntheticEvent) => {
    event.preventDefault()
    const id = lookup.trim()
    if (id) {
      void navigate(`/wallets/${id}`)
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-xl font-semibold">Wallets</h1>
      <div className="grid gap-6 md:grid-cols-2">
        <Card title="Open a wallet">
          <form aria-label="Open a wallet" onSubmit={submitOpen} className="flex flex-col gap-4">
            <div className="flex items-end gap-2">
              <div className="flex-1">
                <Field
                  label="Player ID"
                  id="player-id"
                  value={playerId}
                  onChange={(e) => {
                    setPlayerId(e.target.value)
                  }}
                  required
                />
              </div>
              <Button
                type="button"
                variant="secondary"
                onClick={() => {
                  setPlayerId(crypto.randomUUID())
                }}
              >
                New player
              </Button>
            </div>
            <div className="grid grid-cols-[1fr_auto] gap-2">
              <Field
                label="Initial balance"
                id="initial-balance"
                inputMode="decimal"
                pattern="[0-9]+(\.[0-9]{1,2})?"
                value={amount}
                onChange={(e) => {
                  setAmount(e.target.value)
                }}
                required
              />
              <div className="flex flex-col gap-1">
                <label htmlFor="currency" className="text-xs font-medium text-slate-600">
                  Currency
                </label>
                <select
                  id="currency"
                  value={currency}
                  onChange={(e) => {
                    setCurrency(e.target.value)
                  }}
                  className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm"
                >
                  {currencies.map((code) => (
                    <option key={code}>{code}</option>
                  ))}
                </select>
              </div>
            </div>
            {openWallet.isError && <ErrorNotice error={openWallet.error} />}
            <Button type="submit" disabled={openWallet.isPending}>
              {openWallet.isPending ? 'Opening…' : 'Open wallet'}
            </Button>
          </form>
        </Card>
        <div className="flex flex-col gap-6">
          <Card title="Find a wallet">
            <form aria-label="Find a wallet" onSubmit={submitLookup} className="flex items-end gap-2">
              <div className="flex-1">
                <Field
                  label="Wallet ID"
                  id="wallet-lookup"
                  value={lookup}
                  onChange={(e) => {
                    setLookup(e.target.value)
                  }}
                  placeholder="0192f28f-…"
                />
              </div>
              <Button type="submit" variant="secondary">
                Show
              </Button>
            </form>
          </Card>
          <Card title="Recent wallets">
            {recent.length === 0 ? (
              <p className="text-sm text-slate-500">Wallets you open or look at appear here.</p>
            ) : (
              <ul className="flex flex-col gap-1 text-sm">
                {recent.map((id) => (
                  <li key={id}>
                    <Link to={`/wallets/${id}`} className="font-mono text-brand-700 hover:underline">
                      {id}
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
      </div>
    </div>
  )
}
