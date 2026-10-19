// ApiError is every failure the console shows: the API's own error body
// ({code, error}) when there is one, an HTTP status otherwise, and status 0
// when the request never got an answer.
export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

function isErrorBody(body: unknown): body is { code: string; error: string } {
  return (
    typeof body === 'object' &&
    body !== null &&
    typeof (body as { code?: unknown }).code === 'string' &&
    typeof (body as { error?: unknown }).error === 'string'
  )
}

export function toApiError(response: Response, body: unknown): ApiError {
  if (isErrorBody(body)) {
    return new ApiError(response.status, body.code, body.error)
  }
  return new ApiError(response.status, `HTTP_${response.status}`, response.statusText || `HTTP ${response.status}`)
}

export function networkError(cause: unknown): ApiError {
  const detail = cause instanceof Error ? cause.message : String(cause)
  return new ApiError(0, 'NETWORK_ERROR', `the service did not answer (${detail})`)
}

export function describeError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.status === 0 ? error.message : `${error.message} (${error.code})`
  }
  return error instanceof Error ? error.message : 'unexpected error'
}
