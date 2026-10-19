import type { TransactionStatus } from '../api/types'

const styles: Record<TransactionStatus, string> = {
  PROCESSED: 'bg-emerald-50 text-emerald-800 ring-emerald-600/20',
  PENDING_REFERENCE: 'bg-amber-50 text-amber-800 ring-amber-600/20',
  REJECTED: 'bg-red-50 text-red-800 ring-red-600/20',
  FAILED: 'bg-red-50 text-red-800 ring-red-600/20',
}

export function StatusBadge({ status }: { status: TransactionStatus }) {
  return (
    <span className={`inline-flex rounded-md px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${styles[status]}`}>
      {status}
    </span>
  )
}
