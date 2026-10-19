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
