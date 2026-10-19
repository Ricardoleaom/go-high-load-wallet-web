import type { ComponentProps, ReactNode } from 'react'

export function Card({ title, children, className = '' }: { title?: string; children: ReactNode; className?: string }) {
  return (
    <section
      aria-label={title}
      className={`rounded-xl border border-slate-200 bg-white p-5 shadow-sm ${className}`}
    >
      {title && <h2 className="mb-4 text-sm font-semibold text-slate-900">{title}</h2>}
      {children}
    </section>
  )
}

export function Field({ label, id, ...input }: { label: string; id: string } & ComponentProps<'input'>) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-xs font-medium text-slate-600">
        {label}
      </label>
      <input
        id={id}
        className="rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-xs focus:border-brand-600 focus:outline-2 focus:outline-brand-600/30"
        {...input}
      />
    </div>
  )
}

export function Button({ variant = 'primary', className = '', ...props }: { variant?: 'primary' | 'secondary' } & ComponentProps<'button'>) {
  const look =
    variant === 'primary'
      ? 'bg-brand-700 text-white hover:bg-brand-800 disabled:bg-slate-400'
      : 'border border-slate-300 bg-white text-slate-700 hover:bg-slate-100 disabled:text-slate-400'
  return (
    <button
      className={`rounded-lg px-4 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-700 disabled:cursor-not-allowed ${look} ${className}`}
      {...props}
    />
  )
}

export function Stat({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt className="text-xs font-medium text-slate-500">{label}</dt>
      <dd className="mt-1 text-lg font-semibold text-slate-900 tabular-nums">{children}</dd>
    </div>
  )
}
