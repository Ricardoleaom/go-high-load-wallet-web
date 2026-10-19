import { describe, expect, it } from 'vitest'

import { parseSSE } from './sse'

function streamFromChunks(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  let i = 0
  return new ReadableStream({
    pull(controller) {
      if (i >= chunks.length) {
        controller.close()
        return
      }
      controller.enqueue(encoder.encode(chunks[i]))
      i += 1
    },
  })
}

async function collect<T>(iterable: AsyncIterable<T>): Promise<T[]> {
  const items: T[] = []
  for await (const item of iterable) items.push(item)
  return items
}

describe('parseSSE', () => {
  it('splits events across chunks', async () => {
    // The "data:" line itself is split mid-value by the chunk boundary,
    // and the blank line that ends the event arrives in a later chunk.
    const stream = streamFromChunks([
      'event: progress\ndata: {"a":1',
      '}\n',
      '\nevent: result\ndata: {"b":2}\n\n',
    ])

    const events = await collect(parseSSE(stream))

    expect(events).toEqual([
      { event: 'progress', data: '{"a":1}' },
      { event: 'result', data: '{"b":2}' },
    ])
  })

  it('defaults to the "message" event when none is named', async () => {
    const stream = streamFromChunks(['data: hello\n\n'])

    const events = await collect(parseSSE(stream))

    expect(events).toEqual([{ event: 'message', data: 'hello' }])
  })

  it('joins multiple data lines with a newline, as the spec requires', async () => {
    const stream = streamFromChunks(['data: line one\ndata: line two\n\n'])

    const events = await collect(parseSSE(stream))

    expect(events).toEqual([{ event: 'message', data: 'line one\nline two' }])
  })

  it('ignores a blank keep-alive with no data', async () => {
    const stream = streamFromChunks([': keep-alive\n\n', 'event: progress\ndata: {}\n\n'])

    const events = await collect(parseSSE(stream))

    expect(events).toEqual([{ event: 'progress', data: '{}' }])
  })
})
