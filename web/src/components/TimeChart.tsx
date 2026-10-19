import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'

import { formatClock } from '../lib/format'

export interface Series<Key extends string> {
  key: Key
  label: string
  color: string
}

// TimeChart draws a few series over time from rows keyed by `at`.
export function TimeChart<Key extends string>({
  title,
  rows,
  series,
  unit,
}: {
  title: string
  rows: ({ at: string } & Partial<Record<Key, number | null>>)[]
  series: Series<Key>[]
  unit?: string
}) {
  return (
    <figure className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <figcaption className="mb-2 text-sm font-semibold text-slate-900">{title}</figcaption>
      <div className="h-48">
        <ResponsiveContainer width="100%" height="100%" initialDimension={{ width: 480, height: 192 }}>
          <LineChart data={rows} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <CartesianGrid stroke="#e2e8f0" strokeDasharray="3 3" />
            <XAxis dataKey="at" tickFormatter={formatClock} minTickGap={40} fontSize={11} stroke="#64748b" />
            <YAxis width={56} fontSize={11} stroke="#64748b" unit={unit} />
            <Tooltip labelFormatter={(label) => (typeof label === 'string' ? formatClock(label) : '')} />
            <Legend wrapperStyle={{ fontSize: 12 }} />
            {series.map((s) => (
              <Line
                key={s.key}
                type="monotone"
                dataKey={s.key}
                name={s.label}
                stroke={s.color}
                strokeWidth={2}
                // A point between two gaps (a lone sample with bets) has no
                // line to sit on; the dot keeps it visible.
                dot={{ r: 1.5 }}
                isAnimationActive={false}
                connectNulls={false}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </figure>
  )
}
