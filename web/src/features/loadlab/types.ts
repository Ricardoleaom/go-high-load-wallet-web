// Types mirroring the load lab's own JSON (loadlab/internal/load and
// internal/server), which has no published contract to generate from.

export type Scenario = 'uniform' | 'hot-wallet' | 'replay'

export type Phase = 'seeding' | 'running' | 'draining' | 'verifying' | 'done' | 'stopped' | 'failed'

export interface Latency {
  p50: number
  p95: number
  p99: number
  max: number
}

export interface Point {
  second: number
  offered: number
  sent: number
  completed: number
  dropped: number
  errors: number
  inFlight: number
  latencyMs: Latency
}

export interface Mismatch {
  walletId: string
  expectedCents: number
  actualCents: number
}

export interface Verification {
  passed: boolean
  walletsChecked: number
  mismatches: Mismatch[] | null
  inconsistentReconciliations: number
  resolved: number
  unresolved: number
  error?: string
}

export interface Result {
  phase: Phase
  offered: number
  sent: number
  completed: number
  dropped: number
  errors: number
  replays: number
  statuses: Record<string, number>
  latencyMs: Latency
  loadSeconds: number
  completedPerSecond: number
  verification: Verification
  error?: string
}

export interface Snapshot {
  id: string
  scenario: Scenario
  rate: number
  durationSeconds: number
  wallets: number
  phase: Phase
  elapsedSeconds: number
  offered: number
  sent: number
  completed: number
  dropped: number
  errors: number
  replays: number
  inFlight: number
  statuses: Record<string, number>
  latencyMs: Latency
  series: Point[] | null
  result?: Result
}

// Progress is what the event stream sends: a snapshot without its series,
// plus the second just closed.
export type Progress = Omit<Snapshot, 'series'> & { point?: Point }

export interface StartRunRequest {
  scenario: Scenario
  rate: number
  durationSeconds: number
  wallets: number
}
