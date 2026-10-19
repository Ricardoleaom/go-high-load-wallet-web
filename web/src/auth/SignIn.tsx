import { useSession } from './session'

export function SignIn() {
  const { signIn, status, error } = useSession()
  return (
    <main className="flex min-h-screen items-center justify-center px-4">
      <div className="w-full max-w-sm rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
        <h1 className="text-2xl font-semibold tracking-tight">Wallet console</h1>
        <p className="mt-2 text-sm text-slate-600">
          Operators sign in through the wallet&apos;s identity provider to watch wallets, transactions and load.
        </p>
        {status === 'error' && (
          <p role="alert" className="mt-4 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-800">
            Sign-in failed: {error ?? 'unknown error'}
          </p>
        )}
        <button
          type="button"
          onClick={signIn}
          className="mt-6 w-full rounded-lg bg-brand-700 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-800 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-700"
        >
          Sign in
        </button>
      </div>
    </main>
  )
}
