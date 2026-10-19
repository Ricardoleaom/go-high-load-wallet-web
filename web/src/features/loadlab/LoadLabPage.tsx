import { useState, type SyntheticEvent } from 'react'
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'

import { useSession } from '../../auth/session'
import { ErrorNotice } from '../../components/ErrorNotice'
import { Button, Card, Field, Stat } from '../../components/ui'
import { formatCount, formatMs } from '../../lib/format'
import { useLoadRun } from './useLoadRun'
import type { Scenario } from './types'

const scenarios: { value: Scenario; label: string; description: string }[] = [
  { value: 'uniform', label: 'Uniform', description: 'spread over every seeded wallet' },
  { value: 'hot-wallet', label: 'Hot wallet', description: 'every bet against one wallet' },
  { value: 'replay', label: 'Replay', description: 'half of the sends resend a completed bet' },
]

export function LoadLabPage() {
  const token = useSession().user?.accessToken
  const run = useLoadRun(token)
  const [scenario, setScenario] = useState<Scenario>('uniform')
  const [rate, setRate] = useState('500')
  const [duration, setDuration] = useState('30')
  const [wallets, setWallets] = useState('50')

  const inProgress = run.snapshot !== null && !run.snapshot.result
  const canStart = !inProgress

  const submit = (event: SyntheticEvent) => {
    event.preventDefault()
    void run.start({
      scenario,
      rate: Number(rate),
      durationSeconds: Number(duration),
      wallets: Number(wallets),
    })
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h1 className="text-xl font-semibold">Load lab</h1>
        <p className="text-sm text-slate-500">
          Offers load to the API from this browser&apos;s own load generator, in an open model: arrivals at a fixed
          rate, independent of how fast answers come back.
        </p>
      </div>

      <Card title="Start a run">
        <form aria-label="Start a run" onSubmit={submit} className="flex flex-wrap items-end gap-4">
          <div className="flex flex-col gap-1">
            <label htmlFor="scenario" className="text-xs font-medium text-slate-600">
              Scenario
            </label>
            <select
              id="scenario"
              value={scenario}
              disabled={!canStart}
              onChange={(e) => {
                setScenario(e.target.value as Scenario)
              }}
              className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm disabled:bg-slate-100"
            >
              {scenarios.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label}
                </option>
              ))}
            </select>
          </div>
          <Field
            label="Rate (req/s)"
            id="rate"
            inputMode="numeric"
            value={rate}
            disabled={!canStart}
            onChange={(e) => {
              setRate(e.target.value)
            }}
          />
          <Field
            label="Duration (s)"
            id="duration"
            inputMode="numeric"
            value={duration}
            disabled={!canStart}
            onChange={(e) => {
              setDuration(e.target.value)
            }}
          />
          <Field
            label="Wallets"
            id="wallets"
            inputMode="numeric"
            value={wallets}
            disabled={!canStart}
            onChange={(e) => {
              setWallets(e.target.value)
            }}
          />
          <Button type="submit" disabled={!canStart || run.starting}>
            {run.starting ? 'Starting…' : 'Start run'}
          </Button>
          {inProgress && (
            <Button type="button" variant="secondary" onClick={() => void run.stop()}>
              Stop
            </Button>
          )}
        </form>
        <p className="mt-2 text-xs text-slate-500">
          {scenarios.find((s) => s.value === scenario)?.description}
        </p>
        {run.startError !== null && <ErrorNotice error={run.startError} />}
      </Card>

      {run.snapshot && (
        <>
          <Card title="Right now">
            <dl className="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-7">
              <Stat label="Phase">{run.snapshot.phase}</Stat>
              <Stat label="Offered">{formatCount(run.snapshot.offered)}</Stat>
              <Stat label="Sent">{formatCount(run.snapshot.sent)}</Stat>
              <Stat label="Completed">{formatCount(run.snapshot.completed)}</Stat>
              <Stat label="Dropped">{formatCount(run.snapshot.dropped)}</Stat>
              <Stat label="Errors">{formatCount(run.snapshot.errors)}</Stat>
              <Stat label="Replays">{formatCount(run.snapshot.replays)}</Stat>
            </dl>
            {run.snapshot.dropped > 0 && (
              <p className="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-900">
                {formatCount(run.snapshot.dropped)} arrivals were dropped: this browser could not offer the asked
                rate, or the API was too busy to keep up (past the in-flight limit). The numbers above are what was
                actually sent.
              </p>
            )}
            {run.streamError !== null && (
              <p className="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-900">
                The live stream dropped; recovering from the run&apos;s own state every second.
              </p>
            )}
          </Card>

          {run.points.length > 0 && (
            <div className="grid gap-4 lg:grid-cols-2">
              <OfferedVsCompletedChart points={run.points} />
              <ClientLatencyChart points={run.points} />
            </div>
          )}

          {run.snapshot.result && <ResultCard result={run.snapshot.result} />}
        </>
      )}
    </div>
  )
}

function ResultCard({ result }: { result: NonNullable<ReturnType<typeof useLoadRun>['snapshot']>['result'] }) {
  if (!result) {
    return null
  }
  const v = result.verification
  return (
    <Card title="Result">
      <div
        role="status"
        aria-label="Verification"
        className={`rounded-lg px-4 py-3 text-sm ${v.passed ? 'bg-emerald-50 text-emerald-900' : 'bg-red-50 text-red-900'}`}
      >
        <strong>{v.passed ? 'Consistent' : 'Inconsistent'}</strong>: checked {formatCount(v.walletsChecked)} wallet
        {v.walletsChecked === 1 ? '' : 's'}
        {v.mismatches && v.mismatches.length > 0 && (
          <ul className="mt-2 list-disc pl-5">
            {v.mismatches.map((m) => (
              <li key={m.walletId} className="font-mono text-xs">
                {m.walletId}: expected {m.expectedCents}¢, got {m.actualCents}¢
              </li>
            ))}
          </ul>
        )}
        {v.inconsistentReconciliations > 0 && (
          <p className="mt-1 text-xs">{v.inconsistentReconciliations} wallet(s) failed reconciliation</p>
        )}
        {v.unresolved > 0 && <p className="mt-1 text-xs">{v.unresolved} answer(s) could not be resolved</p>}
      </div>
      <dl className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Stat label="Completed/s">{result.completedPerSecond.toFixed(1)}</Stat>
        <Stat label="p50">{formatMs(result.latencyMs.p50)}</Stat>
        <Stat label="p99">{formatMs(result.latencyMs.p99)}</Stat>
        <Stat label="Replays">{formatCount(result.replays)}</Stat>
      </dl>
    </Card>
  )
}

function OfferedVsCompletedChart({ points }: { points: { second: number; offered: number; completed: number; dropped: number }[] }) {
  return (
    <figure className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <figcaption className="mb-2 text-sm font-semibold text-slate-900">Offered vs completed (per second)</figcaption>
      <div className="h-48">
        <ResponsiveContainer width="100%" height="100%" initialDimension={{ width: 480, height: 192 }}>
          <LineChart data={points} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <CartesianGrid stroke="#e2e8f0" strokeDasharray="3 3" />
            <XAxis dataKey="second" fontSize={11} stroke="#64748b" label={{ value: 's', position: 'insideBottomRight', offset: -2, fontSize: 11 }} />
            <YAxis width={48} fontSize={11} stroke="#64748b" />
            <Tooltip />
            <Legend wrapperStyle={{ fontSize: 12 }} />
            <Line type="monotone" dataKey="offered" name="offered" stroke="#94a3b8" strokeWidth={2} dot={{ r: 1.5 }} isAnimationActive={false} />
            <Line type="monotone" dataKey="completed" name="completed" stroke="#0f766e" strokeWidth={2} dot={{ r: 1.5 }} isAnimationActive={false} />
            <Line type="monotone" dataKey="dropped" name="dropped" stroke="#dc2626" strokeWidth={2} dot={{ r: 1.5 }} isAnimationActive={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </figure>
  )
}

function ClientLatencyChart({ points }: { points: { second: number; latencyMs: { p50: number; p99: number } }[] }) {
  const rows = points.map((p) => ({ second: p.second, p50: p.latencyMs.p50, p99: p.latencyMs.p99 }))
  return (
    <figure className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <figcaption className="mb-2 text-sm font-semibold text-slate-900">
        Client-measured latency (per second)
      </figcaption>
      <div className="h-48">
        <ResponsiveContainer width="100%" height="100%" initialDimension={{ width: 480, height: 192 }}>
          <LineChart data={rows} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <CartesianGrid stroke="#e2e8f0" strokeDasharray="3 3" />
            <XAxis dataKey="second" fontSize={11} stroke="#64748b" label={{ value: 's', position: 'insideBottomRight', offset: -2, fontSize: 11 }} />
            <YAxis width={48} fontSize={11} stroke="#64748b" unit=" ms" />
            <Tooltip />
            <Legend wrapperStyle={{ fontSize: 12 }} />
            <Line type="monotone" dataKey="p50" name="p50" stroke="#0f766e" strokeWidth={2} dot={{ r: 1.5 }} isAnimationActive={false} />
            <Line type="monotone" dataKey="p99" name="p99" stroke="#d97706" strokeWidth={2} dot={{ r: 1.5 }} isAnimationActive={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </figure>
  )
}
