import { SignIn } from './auth/SignIn'
import { useSession } from './auth/session'

export function App() {
  const session = useSession()

  if (session.status === 'loading') {
    return (
      <p role="status" className="p-8 text-sm text-slate-600">
        Signing in…
      </p>
    )
  }
  if (session.status !== 'signed-in') {
    return <SignIn />
  }
  return <p className="p-8">Signed in as {session.user?.name}</p>
}
