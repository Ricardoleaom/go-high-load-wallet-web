import { screen, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'

import type { OpsPoint } from '../../api/types'
import { config } from '../../config'
import { renderSignedIn } from '../../test/render'
import { server } from '../../test/server'
import { DashboardPage } from './DashboardPage'

function point(at: string, overrides: Partial<OpsPoint> = {}): OpsPoint {
  return {
    at,
    requestsPerSecond: 0,
    betsPerSecond: 0,
    serverErrorsPerSecond: 0,
    transactionsPerSecond: 0,
    eventsPublishedPerSecond: 0,
    betP50Ms: null,
    betP99Ms: null,
    outboxPending: 0,
    outboxOldestSeconds: 0,
    poolInUse: 0,
    poolMax: 10,
    poolWaitsPerSecond: 0,
    ...overrides,
  }
}

function serveSummary(points: OpsPoint[]) {
  server.use(http.get(`${config.apiUrl}/ops/summary`, () => HttpResponse.json({ intervalSeconds: 5, points })))
}

describe('DashboardPage', () => {
  it("shows the latest point's numbers", async () => {
    serveSummary([
      point('2026-10-18T21:00:00Z', { requestsPerSecond: 100, betP99Ms: 80 }),
      point('2026-10-18T21:00:05Z', {
        requestsPerSecond: 1021.5,
        betsPerSecond: 1015.25,
        serverErrorsPerSecond: 0.4,
        betP50Ms: 3.21,
        betP99Ms: 42.04,
        outboxPending: 353867,
        outboxOldestSeconds: 231,
        poolInUse: 9,
      }),
    ])
    renderSignedIn(<DashboardPage />)

    const now = await screen.findByRole('region', { name: 'Right now' })
    const value = (label: string) => within(now).getByText(label).nextElementSibling?.textContent
    expect(value('Requests/s')).toBe('1,021.5')
    expect(value('Bets/s')).toBe('1,015.3')
    expect(value('Bet p50')).toBe('3.2 ms')
    expect(value('Bet p99')).toBe('42.0 ms')
    expect(value('5xx/s')).toBe('0.4')
    expect(value('Outbox backlog')).toBe('353,867')
    expect(value('Pool in use')).toBe('9 / 10')
    expect(screen.getByText(/oldest 231 s/)).toBeInTheDocument()
  })

  it('an empty window says the service is warming up', async () => {
    serveSummary([])
    renderSignedIn(<DashboardPage />)

    expect(await screen.findByText(/warming up/i)).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Right now' })).not.toBeInTheDocument()
  })

  it('quantiles without bets show a dash', async () => {
    serveSummary([point('2026-10-18T21:00:05Z')])
    renderSignedIn(<DashboardPage />)

    const now = await screen.findByRole('region', { name: 'Right now' })
    expect(within(now).getByText('Bet p50').nextElementSibling).toHaveTextContent('—')
    expect(within(now).getByText('Bet p99').nextElementSibling).toHaveTextContent('—')
  })
})
