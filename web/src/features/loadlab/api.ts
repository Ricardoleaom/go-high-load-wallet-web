import { ApiError, networkError, toApiError } from '../../api/errors'
import { config } from '../../config'
import { parseSSE } from '../../lib/sse'
import type { Progress, Snapshot, StartRunRequest } from './types'

async function call<T>(path: string, token: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  headers.set('Authorization', `Bearer ${token}`)
  let response: Response
  try {
    response = await fetch(config.loadlabUrl + path, { ...init, headers })
  } catch (cause) {
    throw networkError(cause)
  }
  const body: unknown = await response.json().catch(() => undefined)
  if (!response.ok) {
    throw toApiError(response, body)
  }
  return body as T
}

export function startRun(token: string, request: StartRunRequest): Promise<Snapshot> {
  return call('/runs', token, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  })
}

export function stopRun(token: string, id: string): Promise<Snapshot> {
  return call(`/runs/${id}/stop`, token, { method: 'POST' })
}

export function getRun(token: string, id: string): Promise<Snapshot> {
  return call(`/runs/${id}`, token)
}

// streamRun follows a run's progress and final result over Server-Sent
// Events. EventSource cannot carry an Authorization header, so this uses
// fetch with a streaming body instead, parsed by parseSSE.
export async function* streamRun(token: string, id: string, signal?: AbortSignal): AsyncGenerator<Progress> {
  let response: Response
  try {
    response = await fetch(`${config.loadlabUrl}/runs/${id}/events`, {
      headers: { Authorization: `Bearer ${token}` },
      signal,
    })
  } catch (cause) {
    throw networkError(cause)
  }
  if (!response.ok || !response.body) {
    throw toApiError(response, await response.json().catch(() => undefined))
  }
  for await (const event of parseSSE(response.body)) {
    if (event.event === 'progress' || event.event === 'result') {
      yield JSON.parse(event.data) as Progress
    }
  }
}

export { ApiError }
