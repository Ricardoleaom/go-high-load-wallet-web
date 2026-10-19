import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'

import { config } from '../../config'
import { renderSignedIn } from '../../test/render'
import { server } from '../../test/server'
import { LoadLabPage } from './LoadLabPage'
import type { Progress, Snapshot } from './types'

const lab = config.loadlabUrl

function baseSnapshot(overrides: Partial<Snapshot> = {}): Snapshot {
  return {
    id: 'run-1',
    scenario: 'uniform',
    rate: 500,
    durationSeconds: 30,
    wallets: 50,
    phase: 'seeding',
    elapsedSeconds: 0,
    offered: 0,
    sent: 0,
    completed: 0,
    dropped: 0,
    errors: 0,
    replays: 0,
    inFlight: 0,
    statuses: {},
    latencyMs: { p50: 0, p95: 0, p99: 0, max: 0 },
    series: null,
    ...overrides,
  }
}

function sseBody(events: { event: string; data: unknown }[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  let i = 0
  return new ReadableStream({
    pull(controller) {
      if (i >= events.length) {
        controller.close()
        return
      }
      const e = events[i]
      i += 1
      if (!e) {
        return
      }
      controller.enqueue(encoder.encode(`event: ${e.event}\ndata: ${JSON.stringify(e.data)}\n\n`))
    },
  })
}

function mockRunLifecycle(created: Snapshot, events: { event: string; data: Progress }[]) {
  server.use(
    http.post(`${lab}/runs`, () => HttpResponse.json(created, { status: 201 })),
    http.get(`${lab}/runs/:id/events`, () => {
      return new HttpResponse(sseBody(events), { headers: { 'Content-Type': 'text/event-stream' } })
    }),
  )
}

async function startARun(durationSeconds = 5) {
  const form = screen.getByRole('form', { name: 'Start a run' })
  await userEvent.clear(within(form).getByLabelText('Duration (s)'))
  await userEvent.type(within(form).getByLabelText('Duration (s)'), String(durationSeconds))
  await userEvent.click(within(form).getByRole('button', { name: 'Start run' }))
}

describe('LoadLabPage', () => {
  it('starting a run shows its live numbers', async () => {
    const created = baseSnapshot()
    mockRunLifecycle(created, [
      {
        event: 'progress',
        data: { ...created, phase: 'running', offered: 20, sent: 20, completed: 18, point: { second: 1, offered: 20, sent: 20, completed: 18, dropped: 0, errors: 0, inFlight: 2, latencyMs: { p50: 3.1, p95: 5, p99: 8, max: 9 } } },
      },
    ])
    renderSignedIn(<LoadLabPage />, { route: '/load' })

    await startARun()

    const now = await screen.findByRole('region', { name: 'Right now' })
    await waitFor(() => {
      expect(within(now).getByText('Completed').nextElementSibling).toHaveTextContent('18')
    })
    expect(within(now).getByText('Offered').nextElementSibling).toHaveTextContent('20')
  })

  it('a run already in progress is reported', async () => {
    server.use(
      http.post(`${lab}/runs`, () =>
        HttpResponse.json({ code: 'RUN_IN_PROGRESS', error: 'a run is in progress; stop it or wait for it' }, { status: 409 }),
      ),
    )
    renderSignedIn(<LoadLabPage />, { route: '/load' })

    await startARun()

    expect(await screen.findByRole('alert')).toHaveTextContent('a run is in progress')
  })

  it('the result shows the verification', async () => {
    const created = baseSnapshot()
    const result = {
      phase: 'done' as const,
      offered: 100,
      sent: 100,
      completed: 100,
      dropped: 0,
      errors: 0,
      replays: 0,
      statuses: { '200': 100 },
      latencyMs: { p50: 3.1, p95: 5, p99: 8, max: 10 },
      loadSeconds: 5,
      completedPerSecond: 20,
      verification: { passed: true, walletsChecked: 50, mismatches: null, inconsistentReconciliations: 0, resolved: 0, unresolved: 0 },
    }
    mockRunLifecycle(created, [{ event: 'result', data: { ...created, phase: 'done', offered: 100, sent: 100, completed: 100, result } }])
    renderSignedIn(<LoadLabPage />, { route: '/load' })

    await startARun()

    const status = await screen.findByRole('status', { name: 'Verification' })
    expect(status).toHaveTextContent('Consistent')
    expect(status).toHaveTextContent('50 wallets')
  })

  it('dropped arrivals are shown', async () => {
    const created = baseSnapshot()
    mockRunLifecycle(created, [
      {
        event: 'progress',
        data: { ...created, phase: 'running', offered: 500, sent: 350, dropped: 150, point: { second: 1, offered: 500, sent: 350, completed: 300, dropped: 150, errors: 0, inFlight: 50, latencyMs: { p50: 3, p95: 5, p99: 8, max: 9 } } },
      },
    ])
    renderSignedIn(<LoadLabPage />, { route: '/load' })

    await startARun()

    await waitFor(() => {
      expect(screen.getByText(/150 arrivals were dropped/)).toBeInTheDocument()
    })
  })
})
