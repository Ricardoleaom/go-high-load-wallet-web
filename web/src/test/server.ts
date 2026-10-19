import { setupServer } from 'msw/node'
import { afterAll, afterEach, beforeAll } from 'vitest'

// One MSW server for every test file that imports it; each test adds the
// handlers it needs with server.use().
export const server = setupServer()

beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' })
})
afterEach(() => {
  server.resetHandlers()
})
afterAll(() => {
  server.close()
})
