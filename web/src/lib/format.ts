import type { Money } from '../api/types'

const formatters = new Map<string, Intl.NumberFormat>()

// formatMoney shows an amount in its currency. The API sends decimal
// strings with two fraction digits, which a double holds exactly for any
// balance a wallet can have, so formatting through Number is safe.
export function formatMoney(money: Money): string {
  let formatter = formatters.get(money.currency)
  if (!formatter) {
    formatter = new Intl.NumberFormat('en-US', { style: 'currency', currency: money.currency })
    formatters.set(money.currency, formatter)
  }
  return formatter.format(Number(money.amount))
}

const numberFormat = new Intl.NumberFormat('en-US', { maximumFractionDigits: 1 })
const integerFormat = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 })
const msFormat = new Intl.NumberFormat('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 })

// formatRate shows a per-second rate with at most one decimal.
export function formatRate(value: number): string {
  return numberFormat.format(value)
}

export function formatCount(value: number): string {
  return integerFormat.format(value)
}

// formatMs shows a latency, or a dash when there was nothing to measure.
export function formatMs(value: number | null | undefined): string {
  return value === null || value === undefined ? '—' : `${msFormat.format(value)} ms`
}

export function formatClock(iso: string): string {
  return new Date(iso).toLocaleTimeString('en-GB')
}
