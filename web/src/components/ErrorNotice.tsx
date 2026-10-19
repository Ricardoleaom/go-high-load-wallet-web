import { describeError } from '../api/errors'

export function ErrorNotice({ error }: { error: unknown }) {
  return (
    <p role="alert" className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-800">
      {describeError(error)}
    </p>
  )
}
