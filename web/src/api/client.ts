import createClient, { type Client } from 'openapi-fetch'

import { config } from '../config'
import { networkError, toApiError } from './errors'
import type { paths } from './schema'

export type ApiClient = Client<paths>

// createApiClient returns a client typed by the backend's contract that
// sends the operator's current token with every request. getToken is read
// per request, so a renewed token is used as soon as it exists.
export function createApiClient(getToken: () => string | undefined, baseUrl: string = config.apiUrl): ApiClient {
  const client = createClient<paths>({ baseUrl })
  client.use({
    onRequest({ request }) {
      const token = getToken()
      if (token) {
        request.headers.set('Authorization', `Bearer ${token}`)
      }
      return request
    },
  })
  return client
}

interface FetchResult<T> {
  data?: T
  error?: unknown
  response: Response
}

// unwrap turns openapi-fetch's {data, error} into a value or a thrown
// ApiError, which is what TanStack Query expects.
export async function unwrap<T>(call: Promise<FetchResult<T>>): Promise<T> {
  let result: FetchResult<T>
  try {
    result = await call
  } catch (cause) {
    throw networkError(cause)
  }
  if (!result.response.ok || result.data === undefined) {
    throw toApiError(result.response, result.error)
  }
  return result.data
}
