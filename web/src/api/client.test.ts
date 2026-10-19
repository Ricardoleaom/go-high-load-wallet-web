import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'

import { config } from '../config'
import { server } from '../test/server'
import { createApiClient, unwrap } from './client'
import { ApiError } from './errors'

const walletID = '0192f28f-5dc0-7d58-bdb2-814ad6a0f4a1'
const wallet = {
  id: walletID,
  playerId: '0192f28f-5dc0-7d58-bdb2-814ad6a0f4a2',
  balance: { amount: '10.00', currency: 'BRL' },
  version: 1,
}

describe('API client', () => {
  it("sends the operator's bearer token", async () => {
    let authorization: string | null = null
    server.use(
      http.get(`${config.apiUrl}/wallets/:id`, ({ request }) => {
        authorization = request.headers.get('Authorization')
        return HttpResponse.json(wallet)
      }),
    )
    const api = createApiClient(() => 'operator-token')

    const got = await unwrap(api.GET('/wallets/{id}', { params: { path: { id: walletID } } }))

    expect(authorization).toBe('Bearer operator-token')
    expect(got.balance.amount).toBe('10.00')
  })

  it('an error body becomes an ApiError with its code', async () => {
    server.use(
      http.get(`${config.apiUrl}/wallets/:id`, () =>
        HttpResponse.json({ code: 'WALLET_NOT_FOUND', error: 'wallet not found' }, { status: 404 }),
      ),
    )
    const api = createApiClient(() => 'operator-token')

    const failure = await unwrap(api.GET('/wallets/{id}', { params: { path: { id: walletID } } })).catch(
      (e: unknown) => e,
    )

    expect(failure).toBeInstanceOf(ApiError)
    expect(failure).toMatchObject({ status: 404, code: 'WALLET_NOT_FOUND', message: 'wallet not found' })
  })

  it('a network failure becomes an ApiError with status 0', async () => {
    server.use(http.get(`${config.apiUrl}/wallets/:id`, () => HttpResponse.error()))
    const api = createApiClient(() => 'operator-token')

    const failure = await unwrap(api.GET('/wallets/{id}', { params: { path: { id: walletID } } })).catch(
      (e: unknown) => e,
    )

    expect(failure).toBeInstanceOf(ApiError)
    expect(failure).toMatchObject({ status: 0, code: 'NETWORK_ERROR' })
  })

  it('a body that is not the error shape still becomes an ApiError', async () => {
    server.use(http.get(`${config.apiUrl}/wallets/:id`, () => new HttpResponse('upstream down', { status: 502 })))
    const api = createApiClient(() => 'operator-token')

    const failure = await unwrap(api.GET('/wallets/{id}', { params: { path: { id: walletID } } })).catch(
      (e: unknown) => e,
    )

    expect(failure).toMatchObject({ status: 502, code: 'HTTP_502' })
  })
})
