import { WebStorageStateStore } from 'oidc-client-ts'
import { useMemo, type ReactNode } from 'react'
import { AuthProvider, useAuth } from 'react-oidc-context'

import { config } from '../config'
import { SessionContext, type Session } from './session'

// The console is a public client: the authorization code flow with PKCE
// (oidc-client-ts uses S256), no client secret. Tokens live in
// sessionStorage, so they end with the tab, and are renewed in the
// background before they expire.
const oidcConfig = {
  authority: config.oidcAuthority,
  client_id: config.oidcClientId,
  redirect_uri: window.location.origin + '/',
  post_logout_redirect_uri: window.location.origin + '/',
  scope: 'openid profile',
  automaticSilentRenew: true,
  userStore: new WebStorageStateStore({ store: window.sessionStorage }),
  // Drop ?code=&state= from the address bar once they are used.
  onSigninCallback: () => {
    window.history.replaceState({}, document.title, window.location.pathname)
  },
}

function SessionFromOidc({ children }: { children: ReactNode }) {
  const auth = useAuth()

  const session = useMemo<Session>(() => {
    const signIn = () => void auth.signinRedirect()
    const signOut = () => void auth.signoutRedirect()
    if (auth.isLoading || auth.activeNavigator) {
      return { status: 'loading', signIn, signOut }
    }
    if (auth.error) {
      return { status: 'error', error: auth.error.message, signIn, signOut }
    }
    if (auth.isAuthenticated && auth.user) {
      const profile = auth.user.profile
      return {
        status: 'signed-in',
        user: {
          name: profile.preferred_username ?? profile.name ?? profile.sub,
          accessToken: auth.user.access_token,
        },
        signIn,
        signOut,
      }
    }
    return { status: 'signed-out', signIn, signOut }
  }, [auth])

  return <SessionContext value={session}>{children}</SessionContext>
}

export function OidcSession({ children }: { children: ReactNode }) {
  return (
    <AuthProvider {...oidcConfig}>
      <SessionFromOidc>{children}</SessionFromOidc>
    </AuthProvider>
  )
}
