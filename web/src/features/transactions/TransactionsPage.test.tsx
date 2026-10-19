import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'

import { config } from '../../config'
import { renderSignedIn } from '../../test/render'
import { server } from '../../test/server'
import { TransactionsPage } from './TransactionsPage'

const api = config.apiUrl

describe('TransactionsPage', () => {
  it('finding a transaction by provider and external ID', async () => {
    server.use(
      http.get(`${api}/providers/provider-a/wagering/transactions/bet-42`, () =>
        HttpResponse.json({
          transactionId: '0192f28f-5dc0-7d58-bdb3-000000000042',
          providerId: 'provider-a',
          externalTransactionId: 'bet-42',
          walletId: '0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1',
          kind: 'BET',
          status: 'PROCESSED',
          money: { amount: '5.00', currency: 'BRL' },
          balance: { amount: '95.00', currency: 'BRL' },
        }),
      ),
    )
    renderSignedIn(<TransactionsPage />, { route: '/transactions' })

    const form = screen.getByRole('form', { name: 'Find by provider and external ID' })
    await userEvent.type(within(form).getByLabelText('Provider'), 'provider-a')
    await userEvent.type(within(form).getByLabelText('External ID'), 'bet-42')
    await userEvent.click(within(form).getByRole('button', { name: 'Find' }))

    const card = await screen.findByRole('region', { name: 'Transaction bet-42' })
    expect(card).toHaveTextContent('PROCESSED')
    expect(card).toHaveTextContent('BET')
    expect(card).toHaveTextContent('R$95.00')
    expect(within(card).getByRole('link', { name: /0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1/ })).toHaveAttribute(
      'href',
      '/wallets/0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1',
    )
  })

  it('a missing transaction says not found', async () => {
    server.use(
      http.get(`${api}/wagering/transactions/:id`, () =>
        HttpResponse.json({ code: 'TRANSACTION_NOT_FOUND', error: 'transaction not found' }, { status: 404 }),
      ),
    )
    renderSignedIn(<TransactionsPage />, { route: '/transactions' })

    const form = screen.getByRole('form', { name: 'Find by ID' })
    await userEvent.type(within(form).getByLabelText('Transaction ID'), '0192f28f-5dc0-7d58-bdb3-000000000099')
    await userEvent.click(within(form).getByRole('button', { name: 'Find' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('transaction not found')
  })
})
