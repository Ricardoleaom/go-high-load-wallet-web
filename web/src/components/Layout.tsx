import { NavLink, Outlet } from 'react-router'

import { useSession } from '../auth/session'

const sections = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/wallets', label: 'Wallets', end: false },
  { to: '/transactions', label: 'Transactions', end: false },
  { to: '/load', label: 'Load lab', end: false },
] as const

export function Layout() {
  const { user, signOut } = useSession()
  return (
    <div className="min-h-screen">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-x-6 gap-y-2 px-4 py-3">
          <span className="font-semibold tracking-tight text-brand-700">Wallet console</span>
          <nav aria-label="Sections" className="flex flex-wrap gap-1">
            {sections.map((section) => (
              <NavLink
                key={section.to}
                to={section.to}
                end={section.end}
                className={({ isActive }) =>
                  `rounded-md px-3 py-1.5 text-sm font-medium ${
                    isActive ? 'bg-brand-50 text-brand-800' : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'
                  }`
                }
              >
                {section.label}
              </NavLink>
            ))}
          </nav>
          <div className="ml-auto flex items-center gap-3 text-sm">
            <span className="text-slate-600">{user?.name}</span>
            <button
              type="button"
              onClick={signOut}
              className="rounded-md border border-slate-300 px-3 py-1.5 font-medium text-slate-700 hover:bg-slate-100"
            >
              Sign out
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  )
}
