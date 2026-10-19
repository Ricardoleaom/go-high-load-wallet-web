import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { beforeEach, describe, expect, it } from 'vitest'

import { config } from '../../config'
import { renderSignedIn } from '../../test/render'
import { server } from '../../test/server'
import { WalletsPage } from './WalletsPage'

const api = config.apiUrl
const walletID = '0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1'
const playerID = '0192f28f-5dc0-7d58-bdb2-814ad6a0f4a2'
const brl = (amount: string) => ({ amount, currency: 'BRL' })

function entry(n: number) {
  return {
    id: `0192f28f-5dc0-7d58-bdb2-${String(n).padStart(12, '0')}`,
    transactionId: `0192f28f-5dc0-7d58-bdb3-${String(n).padStart(12, '0')}`,
    direction: 'DEBIT',
    amount: brl('1.00'),
    balanceBefore: brl(`${100 - n + 1}.00`),
    balanceAfter: brl(`${100 - n}.00`),
    createdAt: `2026-10-18T21:00:${String(n).padStart(2, '0')}Z`,
  }
}

beforeEach(() => {
  window.localStorage.clear()
})

describe('WalletsPage', () => {
  it('opening a wallet shows it and remembers it', async () => {
    let opened: unknown
    server.use(
      http.post(`${api}/wallets`, async ({ request }) => {
        opened = await request.json()
        return HttpResponse.json({ id: walletID, playerId: playerID, balance: brl('250.00'), version: 1 }, { status: 201 })
      }),
      http.get(`${api}/wallets/${walletID}`, () =>
        HttpResponse.json({ id: walletID, playerId: playerID, balance: brl('250.00'), version: 1 }),
      ),
      http.get(`${api}/wallets/${walletID}/ledger`, () => HttpResponse.json({ entries: [] })),
    )
    renderSignedIn(<WalletsPage />, { route: '/wallets', path: '/wallets/*' })

    const form = screen.getByRole('form', { name: 'Open a wallet' })
    await userEvent.clear(within(form).getByLabelText('Player ID'))
    await userEvent.type(within(form).getByLabelText('Player ID'), playerID)
    await userEvent.clear(within(form).getByLabelText('Initial balance'))
    await userEvent.type(within(form).getByLabelText('Initial balance'), '250.00')
    await userEvent.click(within(form).getByRole('button', { name: 'Open wallet' }))

    expect(await screen.findByRole('heading', { name: /Wallet 0192f28f/ })).toBeInTheDocument()
    expect(opened).toEqual({ playerId: playerID, initialBalance: brl('250.00') })
    expect(await screen.findByText('R$250.00')).toBeInTheDocument()
    expect(window.localStorage.getItem('console.recentWallets')).toContain(walletID)
  })

  it('a duplicate player and currency shows the conflict', async () => {
    server.use(
      http.post(`${api}/wallets`, () =>
        HttpResponse.json(
          { code: 'WALLET_CONFLICT', error: 'a wallet already exists for this player and currency' },
          { status: 409 },
        ),
      ),
    )
    renderSignedIn(<WalletsPage />, { route: '/wallets', path: '/wallets/*' })

    await userEvent.click(screen.getByRole('button', { name: 'Open wallet' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('a wallet already exists for this player and currency')
  })

  it('the ledger loads the next page from the cursor', async () => {
    const cursors: (string | null)[] = []
    server.use(
      http.get(`${api}/wallets/${walletID}`, () =>
        HttpResponse.json({ id: walletID, playerId: playerID, balance: brl('97.00'), version: 4 }),
      ),
      http.get(`${api}/wallets/${walletID}/ledger`, ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('cursor')
        cursors.push(cursor)
        return cursor === null
          ? HttpResponse.json({ entries: [entry(1), entry(2)], nextCursor: 'page-2' })
          : HttpResponse.json({ entries: [entry(3)] })
      }),
    )
    renderSignedIn(<WalletsPage />, { route: `/wallets/${walletID}`, path: '/wallets/*' })

    const ledger = await screen.findByRole('table', { name: 'Ledger' })
    await waitFor(() => {
      expect(within(ledger).getAllByRole('row')).toHaveLength(3)
    })
    await userEvent.click(screen.getByRole('button', { name: 'Load more' }))

    await waitFor(() => {
      expect(within(ledger).getAllByRole('row')).toHaveLength(4)
    })
    expect(cursors).toEqual([null, 'page-2'])
    expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument()
  })

  it('reconciliation shows consistent and the balances', async () => {
    server.use(
      http.get(`${api}/wallets/${walletID}`, () =>
        HttpResponse.json({ id: walletID, playerId: playerID, balance: brl('97.00'), version: 4 }),
      ),
      http.get(`${api}/wallets/${walletID}/ledger`, () => HttpResponse.json({ entries: [] })),
      http.post(`${api}/wallets/${walletID}/reconciliation`, () =>
        HttpResponse.json({
          walletId: walletID,
          storedBalance: brl('97.00'),
          calculatedBalance: brl('97.00'),
          difference: brl('0.00'),
          consistent: true,
          checkedEntries: 4,
        }),
      ),
    )
    renderSignedIn(<WalletsPage />, { route: `/wallets/${walletID}`, path: '/wallets/*' })

    await userEvent.click(await screen.findByRole('button', { name: 'Reconcile' }))

    const result = await screen.findByRole('status', { name: 'Reconciliation' })
    expect(result).toHaveTextContent('Consistent')
    expect(result).toHaveTextContent('4 ledger entries')
  })
})
