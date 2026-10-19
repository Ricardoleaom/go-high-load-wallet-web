import { createContext, use } from 'react'

export interface SessionUser {
  name: string
  accessToken: string
}

// Session is what the console needs from sign-in, whatever provides it:
// react-oidc-context in the app, a plain value in tests.
export interface Session {
  status: 'loading' | 'signed-out' | 'signed-in' | 'error'
  user?: SessionUser
  error?: string
  signIn: () => void
  signOut: () => void
}

export const SessionContext = createContext<Session | null>(null)

export function useSession(): Session {
  const session = use(SessionContext)
  if (!session) {
    throw new Error('useSession must be used inside a session provider')
  }
  return session
}
