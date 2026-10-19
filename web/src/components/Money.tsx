import type { Money as MoneyValue } from '../api/types'
import { formatMoney } from '../lib/format'

export function Money({ value }: { value: MoneyValue }) {
  return <span className="tabular-nums">{formatMoney(value)}</span>
}
