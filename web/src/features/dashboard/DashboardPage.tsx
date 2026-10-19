import type { OpsPoint } from '../../api/types'
import { ErrorNotice } from '../../components/ErrorNotice'
import { TimeChart } from '../../components/TimeChart'
import { Card, Stat } from '../../components/ui'
import { formatCount, formatMs, formatRate } from '../../lib/format'
import { useOpsSummary } from './api'

export function DashboardPage() {
  const summary = useOpsSummary()
  const points = summary.data?.points ?? []
  const latest = points.at(-1)

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h1 className="text-xl font-semibold">Dashboard</h1>
        <p className="text-sm text-slate-500">
          The API&apos;s own numbers for the last five minutes
          {summary.data ? `, one point every ${summary.data.intervalSeconds} s` : ''}.
        </p>
      </div>
      {summary.isError && <ErrorNotice error={summary.error} />}
      {summary.isSuccess && !latest && (
        <p className="rounded-lg bg-slate-100 px-4 py-3 text-sm text-slate-600">
          The service is warming up: the first point appears after two samples.
        </p>
      )}
      {latest && (
        <>
          <RightNow point={latest} />
          <div className="grid gap-4 lg:grid-cols-2">
            <TimeChart
              title="Throughput (per second)"
              rows={points}
              series={[
                { key: 'requestsPerSecond', label: 'requests', color: '#0f766e' },
                { key: 'betsPerSecond', label: 'bets', color: '#2563eb' },
                { key: 'serverErrorsPerSecond', label: '5xx', color: '#dc2626' },
              ]}
            />
            <TimeChart
              title="Bet latency"
              unit=" ms"
              rows={points}
              series={[
                { key: 'betP50Ms', label: 'p50', color: '#0f766e' },
                { key: 'betP99Ms', label: 'p99', color: '#d97706' },
              ]}
            />
            <TimeChart
              title="Outbox backlog (events)"
              rows={points}
              series={[{ key: 'outboxPending', label: 'pending', color: '#7c3aed' }]}
            />
            <TimeChart
              title="Connection pool (connections)"
              rows={points}
              series={[
                { key: 'poolInUse', label: 'in use', color: '#0f766e' },
                { key: 'poolMax', label: 'max', color: '#94a3b8' },
              ]}
            />
          </div>
        </>
      )}
    </div>
  )
}

function RightNow({ point }: { point: OpsPoint }) {
  return (
    <Card title="Right now">
      <dl className="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-8">
        <Stat label="Requests/s">{formatRate(point.requestsPerSecond)}</Stat>
        <Stat label="Bets/s">{formatRate(point.betsPerSecond)}</Stat>
        <Stat label="Bet p50">{formatMs(point.betP50Ms)}</Stat>
        <Stat label="Bet p99">{formatMs(point.betP99Ms)}</Stat>
        <Stat label="5xx/s">{formatRate(point.serverErrorsPerSecond)}</Stat>
        <Stat label="Outbox backlog">{formatCount(point.outboxPending)}</Stat>
        <Stat label="Pool in use">
          {formatCount(point.poolInUse)} / {formatCount(point.poolMax)}
        </Stat>
        <Stat label="Pool waits/s">{formatRate(point.poolWaitsPerSecond)}</Stat>
      </dl>
      <p className="mt-3 text-xs text-slate-500">
        Outbox: oldest {formatCount(point.outboxOldestSeconds)} s, {formatRate(point.eventsPublishedPerSecond)} events
        published/s. Transactions: {formatRate(point.transactionsPerSecond)}/s.
      </p>
    </Card>
  )
}
