import { useCallback, useEffect, useRef, useState } from 'react'

import { getRun, startRun, stopRun, streamRun } from './api'
import type { Point, Progress, Snapshot, StartRunRequest } from './types'

interface LoadRunState {
  snapshot: Progress | Snapshot | null
  points: Point[]
  starting: boolean
  startError: unknown
  streamError: unknown
}

const initialState: LoadRunState = {
  snapshot: null,
  points: [],
  starting: false,
  startError: null,
  streamError: null,
}

function appendPoint(points: Point[], point: Point | undefined): Point[] {
  if (!point) {
    return points
  }
  if (points.at(-1)?.second === point.second) {
    return points
  }
  return [...points, point]
}

// useLoadRun starts a load lab run, follows it over Server-Sent Events,
// and lets the operator stop it. If the stream drops before the run ends
// (Review Focus: a connection that falls over, a backgrounded tab), it
// falls back to polling GET /runs/{id} once a second until the run has a
// result, recovering the full point series from there.
export function useLoadRun(token: string | undefined) {
  const [state, setState] = useState(initialState)
  const controllerRef = useRef<AbortController | null>(null)
  const generationRef = useRef(0)

  const follow = useCallback(
    (id: string) => {
      if (!token) {
        return
      }
      controllerRef.current?.abort()
      const controller = new AbortController()
      controllerRef.current = controller
      const generation = (generationRef.current += 1)

      void (async () => {
        try {
          for await (const progress of streamRun(token, id, controller.signal)) {
            if (generationRef.current !== generation) {
              return
            }
            setState((s) => ({ ...s, snapshot: progress, points: appendPoint(s.points, progress.point) }))
          }
        } catch (err) {
          if (controller.signal.aborted || generationRef.current !== generation) {
            return
          }
          setState((s) => ({ ...s, streamError: err }))
          await pollUntilDone(token, id, generation, generationRef, setState)
        }
      })()
    },
    [token],
  )

  const start = useCallback(
    async (request: StartRunRequest) => {
      if (!token) {
        return
      }
      setState({ ...initialState, starting: true })
      try {
        const created = await startRun(token, request)
        setState((s) => ({ ...s, starting: false, snapshot: created }))
        follow(created.id)
      } catch (err) {
        setState((s) => ({ ...s, starting: false, startError: err }))
      }
    },
    [token, follow],
  )

  const stop = useCallback(async () => {
    if (!token || !state.snapshot) {
      return
    }
    await stopRun(token, state.snapshot.id)
  }, [token, state.snapshot])

  useEffect(() => () => controllerRef.current?.abort(), [])

  return { ...state, start, stop }
}

async function pollUntilDone(
  token: string,
  id: string,
  generation: number,
  generationRef: { current: number },
  setState: (updater: (s: LoadRunState) => LoadRunState) => void,
): Promise<void> {
  while (generationRef.current === generation) {
    try {
      const snapshot = await getRun(token, id)
      if (generationRef.current !== generation) {
        return
      }
      setState((s) => ({ ...s, snapshot, points: snapshot.series ?? s.points, streamError: null }))
      if (snapshot.result) {
        return
      }
    } catch {
      // The run may still be reachable next time; keep polling.
    }
    await new Promise((resolve) => setTimeout(resolve, 1000))
  }
}
