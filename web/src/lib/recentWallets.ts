const key = 'console.recentWallets'
const limit = 10

// The last wallets this browser opened or looked at: a convenience, so
// storage that is missing or full only means an empty list.
export function recentWallets(): string[] {
  try {
    const parsed: unknown = JSON.parse(window.localStorage.getItem(key) ?? '[]')
    return Array.isArray(parsed) ? parsed.filter((id): id is string => typeof id === 'string') : []
  } catch {
    return []
  }
}

export function rememberWallet(id: string): void {
  try {
    const next = [id, ...recentWallets().filter((known) => known !== id)].slice(0, limit)
    window.localStorage.setItem(key, JSON.stringify(next))
  } catch {
    // Storage unavailable: nothing to remember.
  }
}
