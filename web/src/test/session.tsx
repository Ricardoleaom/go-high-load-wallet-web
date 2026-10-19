import type { ReactNode } from 'react'
import { vi } from 'vitest'

import { SessionContext, type Session } from '../auth/session'

export function signedOut(overrides: Partial<Session> = {}): Session {
  return { status: 'signed-out', signIn: vi.fn(), signOut: vi.fn(), ...overrides }
}

export function signedIn(overrides: Partial<Session> = {}): Session {
  return {
    status: 'signed-in',
    user: { name: 'operator', accessToken: 'operator-token' },
    signIn: vi.fn(),
    signOut: vi.fn(),
    ...overrides,
  }
}

export function WithSession({ session, children }: { session: Session; children: ReactNode }) {
  return <SessionContext value={session}>{children}</SessionContext>
}
